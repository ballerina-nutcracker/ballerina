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

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
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

func closerOf(self *values.Object) (io.Closer, bool) {
	v, ok := self.Get("$closer")
	if !ok {
		return nil, false
	}
	c, ok := v.(io.Closer)
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

func initByteChannelModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	typCtx := semtypes.ContextFrom(env)

	bld := semtypes.NewListDefinition()
	types := byteChannelTypes{
		byteArrTy: bld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE),
	}
	types.byteArrAtom = semtypes.ToListAtomicType(typCtx, types.byteArrTy)
	// io:Block is `readonly & byte[]`; a CELL_MUT_NONE list definition is the
	// atom-backed equivalent of that intersection.
	robld := semtypes.NewListDefinition()
	types.roByteArrTy = robld.DefineListTypeWrappedWithEnvSemTypeCellMutability(env, semtypes.BYTE, semtypes.CellMutability_CELL_MUT_NONE)
	types.roByteArrAtom = semtypes.ToListAtomicType(typCtx, types.roByteArrTy)

	streamCompletionTy := semtypes.Union(semtypes.ERROR, semtypes.NIL)
	bsd := semtypes.NewStreamDefinition()
	types.blockRecordTy = closedNextRecordType(env, types.roByteArrTy)
	types.blockRecordAtm = semtypes.ToMappingAtomicType(typCtx, types.blockRecordTy)
	types.blockStreamTy = bsd.Define(env, types.roByteArrTy, streamCompletionTy)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.attachFile",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			path, _ := args[1].(string)
			f, err := rt.Platform().FS.OpenReadable(path)
			if err != nil {
				return fileIOError("error while opening file '" + path + "': " + err.Error()), nil
			}
			self.Put("$reader", f)
			self.Put("$closer", f)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.attachBytes",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			list, _ := args[1].(*values.List)
			self.Put("$reader", bytes.NewReader(list.ToByteSlice()))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.read",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			nBytes, _ := args[1].(int64)

			if isClosed(self) {
				return byteChannelClosedError(), nil
			}
			reader, ok := readerOf(self)
			if !ok {
				return fileIOError("Byte channel is not initialized"), nil
			}

			size := int(nBytes)
			if size <= 0 {
				size = channelBufferSize
			}
			buf := make([]byte, size)
			n, err := reader.Read(buf)
			if n == 0 {
				if err == io.EOF || err == nil {
					return byteChannelEofError(), nil
				}
				return fileIOError("error occurred while reading bytes from the channel. " + err.Error()), nil
			}
			return values.NewList(types.byteArrTy, types.byteArrAtom, false, nil, 0, bytesToItems(buf[:n])), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.readAll",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
			return values.NewList(types.roByteArrTy, types.roByteArrAtom, true, nil, 0, bytesToItems(data)), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.blockStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
			next := func() values.BalValue {
				buf := make([]byte, size)
				n, err := reader.Read(buf)
				if n > 0 {
					block := values.NewList(types.roByteArrTy, types.roByteArrAtom, true, nil, 0, bytesToItems(buf[:n]))
					return values.NewMap(types.blockRecordTy, types.blockRecordAtm, false,
						[]values.MapEntry{{Key: "value", Value: block}})
				}
				if err == io.EOF || err == nil {
					markClosed(self)
					return nil
				}
				markClosed(self)
				return fileIOError("error occurred while reading bytes from the channel. " + err.Error())
			}
			closeFn := func() values.BalValue {
				if isClosed(self) {
					return nil
				}
				markClosed(self)
				return nil
			}
			return values.NewStream(types.blockStreamTy, next, closeFn), nil
		})

	registerBase64Transform := func(name string, transform func(data []byte) ([]byte, error)) {
		runtime.RegisterExternFunction(rt, orgName, moduleName, name,
			func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
				return values.NewList(types.byteArrTy, types.byteArrAtom, false, nil, 0, bytesToItems(out)), nil
			})
	}

	registerBase64Transform("ReadableByteChannel.base64EncodeBytes",
		func(data []byte) ([]byte, error) {
			return base64.StdEncoding.AppendEncode(nil, data), nil
		})

	registerBase64Transform("ReadableByteChannel.base64DecodeBytes",
		func(data []byte) ([]byte, error) {
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

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableByteChannel.close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return byteChannelClosedError(), nil
			}
			markClosed(self)
			if closer, ok := closerOf(self); ok {
				if err := closer.Close(); err != nil {
					return fileIOError("error occurred while closing the channel. " + err.Error()), nil
				}
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableByteChannel.attachFile",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			path, _ := args[1].(string)
			option, _ := args[2].(string)
			w, err := rt.Platform().FS.OpenWritable(path, option == "APPEND")
			if err != nil {
				return fileIOError("error while opening file '" + path + "': " + err.Error()), nil
			}
			self.Put("$writer", w)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableByteChannel.write",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableByteChannel.close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
		})
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
