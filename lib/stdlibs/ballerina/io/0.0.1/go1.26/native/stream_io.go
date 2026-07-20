// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

func bytesToBlockList(byteArrTy semtypes.SemType, atom *semtypes.ListAtomicType, data []byte) *values.List {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return values.NewList(byteArrTy, atom, false, nil, 0, items)
}

// readLineCRLF reads a single line from r, splitting on "\n", "\r" and "\r\n"
// and stripping the terminator. ok is false only for a clean end-of-stream
// (no bytes read before EOF); a non-nil error means the read failed.
func readLineCRLF(r *bufio.Reader) (line string, ok bool, err error) {
	var sb strings.Builder
	readAny := false
	for {
		b, readErr := r.ReadByte()
		if readErr != nil {
			if readErr == io.EOF {
				if !readAny {
					return "", false, nil
				}
				return sb.String(), true, nil
			}
			return "", false, readErr
		}
		readAny = true
		if b == '\n' {
			return sb.String(), true, nil
		}
		if b == '\r' {
			nb, peekErr := r.ReadByte()
			if peekErr == nil && nb != '\n' {
				_ = r.UnreadByte()
			}
			return sb.String(), true, nil
		}
		sb.WriteByte(b)
	}
}

// closableHandle guards a PAL file handle against double-close, since a
// stream's close() may be called explicitly after next() already closed it
// on end-of-stream.
type closableHandle struct {
	handle io.Closer
	closed bool
}

func (h *closableHandle) closeOnce() error {
	if h.closed {
		return nil
	}
	h.closed = true
	return h.handle.Close()
}

func registerStreamIOExterns(rt *runtime.Runtime, types fileIOTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadLinesAsStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			file, err := rt.Platform().FS.OpenReadable(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			handle := &closableHandle{handle: file}
			reader := bufio.NewReader(file)
			next := func() values.BalValue {
				line, ok, readErr := readLineCRLF(reader)
				if readErr != nil {
					_ = handle.closeOnce()
					return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, readErr.Error()))
				}
				if !ok {
					_ = handle.closeOnce()
					return nil
				}
				return values.NewMap(types.lineRecordTy, types.lineRecordAtom, false,
					[]values.MapEntry{{Key: "value", Value: line}})
			}
			closeFn := func() values.BalValue {
				if err := handle.closeOnce(); err != nil {
					return fileIOError(fmt.Sprintf("error while closing file '%s': %s", path, err.Error()))
				}
				return nil
			}
			return values.NewStream(types.lineStreamTy, next, closeFn), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileReadBlocksAsStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			blockSize := int(args[1].(int64))
			if blockSize <= 0 {
				return fileIOError(fmt.Sprintf("invalid block size: %d", blockSize)), nil
			}
			file, err := rt.Platform().FS.OpenReadable(path)
			if err != nil {
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, err.Error())), nil
			}
			handle := &closableHandle{handle: file}
			next := func() values.BalValue {
				buf := make([]byte, blockSize)
				n, readErr := io.ReadFull(file, buf)
				if n > 0 {
					block := bytesToBlockList(types.byteArrTy, types.byteArrAtom, buf[:n])
					return values.NewMap(types.blockRecordTy, types.blockRecordAtom, false,
						[]values.MapEntry{{Key: "value", Value: block}})
				}
				if readErr == io.EOF {
					_ = handle.closeOnce()
					return nil
				}
				_ = handle.closeOnce()
				return fileIOError(fmt.Sprintf("error while reading file '%s': %s", path, readErr.Error()))
			}
			closeFn := func() values.BalValue {
				if err := handle.closeOnce(); err != nil {
					return fileIOError(fmt.Sprintf("error while closing file '%s': %s", path, err.Error()))
				}
				return nil
			}
			return values.NewStream(types.blockStreamTy, next, closeFn), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteLinesFromStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			lineStream, _ := args[1].(*values.Stream)
			option, _ := args[2].(string)

			w, err := rt.Platform().FS.OpenWritable(path, option == "APPEND")
			if err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			for {
				elem := lineStream.Next()
				if elem == nil {
					break
				}
				if errVal, ok := elem.(*values.Error); ok {
					_ = w.Close()
					return errVal, nil
				}
				record, _ := elem.(*values.Map)
				value, _ := record.Get("value")
				line, _ := value.(string)
				if _, writeErr := w.Write([]byte(line + "\n")); writeErr != nil {
					_ = w.Close()
					return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, writeErr.Error())), nil
				}
			}
			if err := w.Close(); err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFileWriteBlocksFromStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			path, _ := args[0].(string)
			byteStream, _ := args[1].(*values.Stream)
			option, _ := args[2].(string)

			w, err := rt.Platform().FS.OpenWritable(path, option == "APPEND")
			if err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			for {
				elem := byteStream.Next()
				if elem == nil {
					break
				}
				if errVal, ok := elem.(*values.Error); ok {
					_ = w.Close()
					return errVal, nil
				}
				record, _ := elem.(*values.Map)
				value, _ := record.Get("value")
				block, _ := value.(*values.List)
				if _, writeErr := w.Write(block.ToByteSlice()); writeErr != nil {
					_ = w.Close()
					return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, writeErr.Error())), nil
				}
			}
			if err := w.Close(); err != nil {
				return fileIOError(fmt.Sprintf("error while writing to file '%s': %s", path, err.Error())), nil
			}
			return nil, nil
		})
}
