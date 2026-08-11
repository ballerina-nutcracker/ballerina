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
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// channelBufferSize matches jBallerina's IOConstants.CHANNEL_BUFFER_SIZE,
// used when read() is called with a non-positive nBytes.
const channelBufferSize = 16384

func readerOf(self *values.Object) (io.Reader, bool) {
	v, ok := self.Get("$reader")
	if !ok {
		return nil, false
	}
	r, ok := v.(io.Reader)
	return r, ok
}

func closerOf(self *values.Object) (*closableHandle, bool) {
	v, ok := self.Get("$closer")
	if !ok {
		return nil, false
	}
	c, ok := v.(*closableHandle)
	return c, ok
}

func writerOf(self *values.Object) (io.WriteCloser, bool) {
	v, ok := self.Get("$writer")
	if !ok {
		return nil, false
	}
	w, ok := v.(io.WriteCloser)
	return w, ok
}

func isClosed(self *values.Object) bool {
	v, ok := self.Get("$closed")
	if !ok {
		return false
	}
	closed, _ := v.(bool)
	return closed
}

func markClosed(self *values.Object) {
	self.Put("$closed", true)
}

func byteChannelClosedError() values.BalValue {
	return fileIOError("Byte channel is already closed.")
}

func byteChannelEofError() values.BalValue {
	return fileIOError("EoF when reading from the channel")
}

type byteChannelTypes struct {
	byteArrTy     semtypes.SemType
	byteArrAtom   *semtypes.ListAtomicType
	roByteArrTy   semtypes.SemType
	roByteArrAtom *semtypes.ListAtomicType

	blockStreamTy  semtypes.SemType
	blockRecordTy  semtypes.SemType
	blockRecordAtm *semtypes.MappingAtomicType
}

// byteChannelExterns implements the extern functions of both
// ReadableByteChannel and WritableByteChannel; each is registered as a
// named method rather than an inline closure, so the registration list
// below stays a plain map from Ballerina method name to Go implementation.
type byteChannelExterns struct {
	rt    *runtime.Runtime
	types byteChannelTypes
}

func (e *byteChannelExterns) readableAttachFile(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	path, _ := args[1].(string)
	f, err := e.rt.Platform().FS.OpenReadable(path)
	if err != nil {
		return fileIOError("error while opening file '" + path + "': " + err.Error()), nil
	}
	self.Put("$reader", f)
	self.Put("$closer", &closableHandle{handle: f})
	return nil, nil
}

func (e *byteChannelExterns) readableAttachBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	list, _ := args[1].(*values.List)
	self.Put("$reader", bytes.NewReader(list.ToByteSlice()))
	return nil, nil
}

func (e *byteChannelExterns) readableRead(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	nBytes, _ := args[1].(int64)

	if isClosed(self) {
		return byteChannelClosedError(), nil
	}
	// jBallerina's read() returns an empty byte[] the first time the
	// channel is exhausted, and only errors on a subsequent call.
	if eofReached(self) {
		return byteChannelEofError(), nil
	}
	reader, ok := readerOf(self)
	if !ok {
		return fileIOError("Byte channel is not initialized"), nil
	}

	size := int(nBytes)
	if size <= 0 {
		size = channelBufferSize
	}
	result := make([]byte, 0, min(size, channelBufferSize))
	var readErr error
	for len(result) < size {
		chunk := make([]byte, min(size-len(result), channelBufferSize))
		n, err := reader.Read(chunk)
		if n > 0 {
			result = append(result, chunk[:n]...)
		}
		if err != nil {
			readErr = err
			break
		}
		if n == 0 {
			break
		}
	}
	if len(result) == 0 {
		if readErr != nil && readErr != io.EOF {
			return fileIOError("error occurred while reading bytes from the channel. " + readErr.Error()), nil
		}
		self.Put("$eof", true)
	}
	return values.NewList(e.types.byteArrTy, e.types.byteArrAtom, false, nil, 0, bytesToItems(result)), nil
}

func (e *byteChannelExterns) readableReadAll(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)

	if isClosed(self) {
		return byteChannelClosedError(), nil
	}
	reader, ok := readerOf(self)
	if !ok {
		return fileIOError("Byte channel is not initialized"), nil
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return fileIOError("error occurred while reading bytes from the channel. " + err.Error()), nil
	}
	return values.NewList(e.types.roByteArrTy, e.types.roByteArrAtom, true, nil, 0, bytesToItems(data)), nil
}

func (e *byteChannelExterns) readableBlockStream(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	blockSize, _ := args[1].(int64)

	if isClosed(self) {
		return byteChannelClosedError(), nil
	}
	reader, ok := readerOf(self)
	if !ok {
		return fileIOError("Byte channel is not initialized"), nil
	}
	if blockSize <= 0 {
		return fileIOError("invalid block size"), nil
	}
	size := int(blockSize)
	// Reaching the end of the stream (or closing it explicitly) only
	// releases the underlying OS handle; it must not mark the parent
	// ReadableByteChannel itself closed, since jBallerina still allows
	// a later channel.close() to succeed after the stream is done.
	closeUnderlyingHandle := func() error {
		if closer, ok := closerOf(self); ok {
			return closer.closeOnce()
		}
		return nil
	}
	buf := make([]byte, min(size, channelBufferSize))
	next := func() values.BalValue {
		result := make([]byte, 0, len(buf))
		var readErr error
		for len(result) < size {
			chunk := buf
			if remaining := size - len(result); remaining < len(chunk) {
				chunk = buf[:remaining]
			}
			n, err := reader.Read(chunk)
			if n > 0 {
				result = append(result, chunk[:n]...)
			}
			if err != nil {
				readErr = err
				break
			}
			if n == 0 {
				break
			}
		}
		if readErr != nil && readErr != io.EOF {
			_ = closeUnderlyingHandle()
			return fileIOError("error occurred while reading bytes from the channel. " + readErr.Error())
		}
		if len(result) == 0 {
			if closeErr := closeUnderlyingHandle(); closeErr != nil {
				return fileIOError("error occurred while closing the channel. " + closeErr.Error())
			}
			return nil
		}
		block := values.NewList(e.types.roByteArrTy, e.types.roByteArrAtom, true, nil, 0, bytesToItems(result))
		return values.NewMap(e.types.blockRecordTy, e.types.blockRecordAtm, false,
			[]values.MapEntry{{Key: "value", Value: block}})
	}
	closeFn := func() values.BalValue {
		if err := closeUnderlyingHandle(); err != nil {
			return fileIOError("error occurred while closing the channel. " + err.Error())
		}
		return nil
	}
	return values.NewStream(e.types.blockStreamTy, next, closeFn), nil
}

// base64Transform reads the whole channel, applies transform, and wraps the
// result as a byte[]; base64EncodeBytes/base64DecodeBytes supply transform.
func (e *byteChannelExterns) base64Transform(args []values.BalValue, transform func(data []byte) ([]byte, error)) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return fileIOError("Channel is already closed."), nil
	}
	reader, ok := readerOf(self)
	if !ok {
		return fileIOError("Byte channel is not initialized"), nil
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return fileIOError(err.Error()), nil
	}
	out, err := transform(data)
	if err != nil {
		// A Go error return panics in the interpreter, matching
		// jBallerina, where the Base64 IllegalArgumentException
		// escapes uncaught.
		return nil, err
	}
	return values.NewList(e.types.byteArrTy, e.types.byteArrAtom, false, nil, 0, bytesToItems(out)), nil
}

func (e *byteChannelExterns) base64EncodeBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return e.base64Transform(args, func(data []byte) ([]byte, error) {
		return base64.StdEncoding.AppendEncode(nil, data), nil
	})
}

func (e *byteChannelExterns) base64DecodeBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return e.base64Transform(args, func(data []byte) ([]byte, error) {
		// jBallerina's decoder treats the final padding as optional.
		enc := base64.StdEncoding
		if len(data)%4 != 0 {
			enc = base64.RawStdEncoding
		}
		decoded, err := enc.AppendDecode(nil, data)
		if err != nil {
			return nil, fmt.Errorf("illegal Base64 input: %s", err.Error())
		}
		return decoded, nil
	})
}

func (e *byteChannelExterns) readableClose(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return byteChannelClosedError(), nil
	}
	markClosed(self)
	if closer, ok := closerOf(self); ok {
		if err := closer.closeOnce(); err != nil {
			return fileIOError("error occurred while closing the channel. " + err.Error()), nil
		}
	}
	return nil, nil
}

func (e *byteChannelExterns) writableAttachFile(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	path, _ := args[1].(string)
	option, _ := args[2].(string)
	w, err := e.rt.Platform().FS.OpenWritable(path, option == "APPEND")
	if err != nil {
		return fileIOError("error while opening file '" + path + "': " + err.Error()), nil
	}
	self.Put("$writer", w)
	return nil, nil
}

func (e *byteChannelExterns) writableWrite(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	list, _ := args[1].(*values.List)
	offset, _ := args[2].(int64)

	if isClosed(self) {
		return byteChannelClosedError(), nil
	}
	writer, ok := writerOf(self)
	if !ok {
		return fileIOError("Byte channel is not initialized"), nil
	}
	content := list.ToByteSlice()
	if offset < 0 || int(offset) > len(content) {
		return fileIOError("invalid offset for the given content"), nil
	}
	n, err := writer.Write(content[offset:])
	if err != nil {
		return fileIOError("error occurred while writing bytes to the channel. " + err.Error()), nil
	}
	return int64(n), nil
}

func (e *byteChannelExterns) writableClose(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return byteChannelClosedError(), nil
	}
	markClosed(self)
	if writer, ok := writerOf(self); ok {
		if err := writer.Close(); err != nil {
			return fileIOError("error occurred while closing the channel. " + err.Error()), nil
		}
	}
	return nil, nil
}

func initByteChannelModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	typCtx := semtypes.ContextFrom(env)

	bld := semtypes.NewListDefinition()
	types := byteChannelTypes{
		byteArrTy: bld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE),
	}
	types.byteArrAtom = semtypes.ToListAtomicType(env, types.byteArrTy)
	// io:Block is `readonly & byte[]`; a CELL_MUT_NONE list definition is the
	// atom-backed equivalent of that intersection.
	robld := semtypes.NewListDefinition()
	types.roByteArrTy = robld.DefineListTypeWrappedWithEnvSemTypeCellMutability(env, semtypes.BYTE, semtypes.CellMutability_CELL_MUT_NONE)
	types.roByteArrAtom = semtypes.ToListAtomicType(env, types.roByteArrTy)

	streamCompletionTy := semtypes.Union(semtypes.ERROR, semtypes.NIL)
	bsd := semtypes.NewStreamDefinition()
	types.blockRecordTy = closedNextRecordType(env, types.roByteArrTy)
	types.blockRecordAtm = semtypes.ToMappingAtomicType(typCtx, types.blockRecordTy)
	types.blockStreamTy = bsd.Define(env, types.roByteArrTy, streamCompletionTy)

	e := &byteChannelExterns{rt: rt, types: types}
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.attachFile", e.readableAttachFile)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.attachBytes", e.readableAttachBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.read", e.readableRead)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.readAll", e.readableReadAll)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.blockStream", e.readableBlockStream)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.base64EncodeBytes", e.base64EncodeBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.base64DecodeBytes", e.base64DecodeBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.close", e.readableClose)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableByteChannel.attachFile", e.writableAttachFile)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableByteChannel.write", e.writableWrite)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableByteChannel.close", e.writableClose)
}

func bytesToItems(data []byte) []values.BalValue {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return items
}

func init() {
	runtime.RegisterModuleInitializer(initByteChannelModule)
}
