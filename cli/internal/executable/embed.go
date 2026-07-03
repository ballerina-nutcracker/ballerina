// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package executable handles BIR embedding for bal build output and startup detection.
//
// A compiled Ballerina executable is an unmodified bal binary with a BIR payload
// and a 16-byte trailer appended to it:
//
//	[bal binary bytes] [BIR payload] [8-byte payload offset] [8-byte magic]
//
// At startup, the binary reads its own last 16 bytes. Finding the magic means it
// is running as a compiled program — it deserializes the payload and runs the BIR
// instead of entering the CLI.
package executable

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"ballerina-lang-go/bir"
	bircodec "ballerina-lang-go/bir/codec"
	balctx "ballerina-lang-go/context"
	"ballerina-lang-go/semtypes"
)

const (
	magic       = "BALEXE\x00\x01"
	trailerSize = 16 // 8-byte payload offset + 8-byte magic
)

// Pack writes a self-contained Ballerina executable to outPath.
//
// The output is: [stub bytes] [BIR payload] [16-byte trailer]
//
// stubPath is the path to the runner binary for the target platform — typically
// the currently running bal binary (os.Executable()) for same-platform builds.
// outPath is created (with parent directories) and made executable.
func Pack(stubPath string, birPkgs []*bir.BIRPackage, tyEnv semtypes.Env, outPath string) error {
	stub, err := os.Open(stubPath)
	if err != nil {
		return fmt.Errorf("opening runner stub: %w", err)
	}
	defer func() { _ = stub.Close() }()

	stubInfo, err := stub.Stat()
	if err != nil {
		return fmt.Errorf("stat runner stub: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	// Ensure execute bits are set even when the file already existed (O_TRUNC
	// reuses the inode and does not apply the mode argument).
	if err := os.Chmod(outPath, 0o755); err != nil {
		_ = out.Close()
		return fmt.Errorf("setting output file permissions: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, stub); err != nil {
		return fmt.Errorf("copying stub: %w", err)
	}
	payloadOffset := stubInfo.Size()

	payload, err := marshalPayload(birPkgs, tyEnv)
	if err != nil {
		return err
	}
	if _, err := out.Write(payload); err != nil {
		return fmt.Errorf("writing BIR payload: %w", err)
	}

	trailer := make([]byte, trailerSize)
	binary.LittleEndian.PutUint64(trailer[:8], uint64(payloadOffset))
	copy(trailer[8:], magic)
	if _, err := out.Write(trailer); err != nil {
		return fmt.Errorf("writing trailer: %w", err)
	}
	return nil
}

// TryLoad checks whether the running binary has embedded BIR.
//
// Returns (pkgs, tyEnv, nil) if an embedded program is found.
// Returns (nil, nil, nil) if this is a plain bal binary with no embedded BIR.
// Returns (nil, nil, err) if the magic is present but the payload is corrupt.
func TryLoad() ([]*bir.BIRPackage, semtypes.Env, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, nil
	}
	f, err := os.Open(exe)
	if err != nil {
		return nil, nil, nil
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil || info.Size() < int64(trailerSize) {
		return nil, nil, nil
	}

	trailer := make([]byte, trailerSize)
	if _, err := f.ReadAt(trailer, info.Size()-int64(trailerSize)); err != nil {
		return nil, nil, nil
	}
	if string(trailer[8:]) != magic {
		return nil, nil, nil
	}

	rawOffset := binary.LittleEndian.Uint64(trailer[:8])
	// Guard against corrupt offsets that wrap to negative when cast to int64,
	// which would make payloadSize a large positive value and cause OOM.
	if rawOffset > uint64(info.Size()-int64(trailerSize)) {
		return nil, nil, fmt.Errorf("invalid embedded payload offset %d", rawOffset)
	}
	payloadOffset := int64(rawOffset)
	payloadSize := info.Size() - payloadOffset - int64(trailerSize)
	if payloadSize <= 0 {
		return nil, nil, fmt.Errorf("invalid embedded payload size %d", payloadSize)
	}

	payload := make([]byte, payloadSize)
	if _, err := f.ReadAt(payload, payloadOffset); err != nil {
		return nil, nil, fmt.Errorf("reading embedded payload: %w", err)
	}

	pkgs, tyEnv, err := unmarshalPayload(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("corrupt embedded program: %w", err)
	}
	return pkgs, tyEnv, nil
}

func marshalPayload(birPkgs []*bir.BIRPackage, tyEnv semtypes.Env) ([]byte, error) {
	// Format: [uint32 count] ([uint32 len] [BIR bytes])*
	if len(birPkgs) > math.MaxUint32 {
		return nil, fmt.Errorf("too many BIR packages: %d", len(birPkgs))
	}
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, uint32(len(birPkgs)))
	buf := append([]byte(nil), count...)

	for _, pkg := range birPkgs {
		data, err := bircodec.Marshal(tyEnv, pkg)
		if err != nil {
			return nil, fmt.Errorf("serializing %s: %w", pkg.PackageID.PkgName.Value(), err)
		}
		if len(data) > math.MaxUint32 {
			return nil, fmt.Errorf("serialized BIR package too large: %d bytes", len(data))
		}
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(len(data)))
		buf = append(buf, lenBytes...)
		buf = append(buf, data...)
	}
	return buf, nil
}

func unmarshalPayload(payload []byte) ([]*bir.BIRPackage, semtypes.Env, error) {
	if len(payload) < 4 {
		return nil, nil, fmt.Errorf("payload too short")
	}

	tyEnv := semtypes.CreateTypeEnv()
	env := balctx.NewCompilerEnvironment(tyEnv, false)
	ctx := balctx.NewCompilerContext(env)

	count := int(binary.BigEndian.Uint32(payload[:4]))
	if count > (len(payload)-4)/4 {
		return nil, nil, fmt.Errorf("invalid package count %d", count)
	}
	pos := 4
	pkgs := make([]*bir.BIRPackage, 0, count)

	for i := range count {
		if pos+4 > len(payload) {
			return nil, nil, fmt.Errorf("truncated at package %d", i)
		}
		pkgLen := int(binary.BigEndian.Uint32(payload[pos : pos+4]))
		pos += 4

		if pos+pkgLen > len(payload) {
			return nil, nil, fmt.Errorf("truncated BIR at package %d", i)
		}
		pkg, err := bircodec.Unmarshal(ctx, payload[pos:pos+pkgLen])
		if err != nil {
			return nil, nil, fmt.Errorf("deserializing package %d: %w", i, err)
		}
		pos += pkgLen
		pkgs = append(pkgs, pkg)
	}
	return pkgs, tyEnv, nil
}
