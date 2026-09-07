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
	"fmt"
	"io"
	"math"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func dataChannelClosedError() values.BalValue {
	return fileIOError("Data channel is already closed.")
}

func littleEndianOf(self *values.Object) bool {
	v, ok := self.Get("$order")
	if !ok {
		return false
	}
	order, _ := v.(string)
	return order == "LE"
}

// readDataBytes reads up to width bytes from the wrapped byte channel,
// marking the channel's EOF flag when the input ends. Zero available bytes is
// an error, which panics in the interpreter, matching jBallerina's
// BufferUnderflowException for fixed-width reads on an exhausted channel.
func readDataBytes(self *values.Object, width int) ([]byte, error) {
	byteCh, _ := byteChannelOf(self)
	reader, ok := readerOf(byteCh)
	if !ok {
		return nil, fmt.Errorf("byte channel is not initialized")
	}
	buf := make([]byte, width)
	n, err := io.ReadFull(reader, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		self.Put("$eof", true)
		err = nil
	}
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("no bytes available to read from the channel")
	}
	if littleEndianOf(self) {
		// jBallerina reverses the whole fixed-width buffer before truncating
		// to the bytes actually read, so a short little-endian read sees the
		// zero padding first.
		for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
			buf[i], buf[j] = buf[j], buf[i]
		}
	}
	return buf[:n], nil
}

// decodeDataInt decodes data as a signed big-endian integer using the same
// arithmetic widths as jBallerina's DataChannel.deriveLong, whose behaviour
// for reads shorter than the representation width depends on them.
func decodeDataInt(width int, data []byte) int64 {
	firstByte := int64(int8(data[0])) & 0xFFFF
	if width == 2 {
		firstByte = int64(int16(firstByte))
	}
	totalBits := uint(len(data)-1) * 8
	var value int64
	if width == 4 {
		value = int64(int32(firstByte << totalBits))
	} else {
		value = firstByte << totalBits
	}
	for i := 1; i < len(data); i++ {
		value += int64(data[i]) << (uint(len(data)-1-i) * 8)
	}
	return value
}

func encodeDataInt(value int64, width int, littleEndian bool) []byte {
	buf := make([]byte, width)
	for i := 0; i < width; i++ {
		buf[i] = byte(value >> (uint(width-1-i) * 8))
	}
	if littleEndian {
		for i, j := 0, width-1; i < j; i, j = i+1, j-1 {
			buf[i], buf[j] = buf[j], buf[i]
		}
	}
	return buf
}

// varIntWidth returns the minimal number of 7-bit groups that hold value with
// its sign bit, capped at 10 groups for the full int64 range.
func varIntWidth(value int64) int {
	for n := 1; n < 10; n++ {
		limit := int64(1) << (uint(7*n) - 1)
		if value >= -limit && value < limit {
			return n
		}
	}
	return 10
}

// encodeVarInt renders value in jBallerina's variable-length format: 7-bit
// groups in big-endian order with the continuation bit set on every byte but
// the last, and the group order reversed for little-endian channels.
func encodeVarInt(value int64, littleEndian bool) []byte {
	n := varIntWidth(value)
	groups := make([]byte, n)
	for i := 0; i < n; i++ {
		groups[i] = byte(value>>(uint(7*(n-1-i)))) & 0x7F
	}
	if littleEndian {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			groups[i], groups[j] = groups[j], groups[i]
		}
	}
	for i := 0; i < n-1; i++ {
		groups[i] |= 0x80
	}
	return groups
}

// writeDataBytes writes data to self's wrapped writable byte channel.
func writeDataBytes(self *values.Object, data []byte) values.BalValue {
	byteCh, _ := byteChannelOf(self)
	writer, ok := writerOf(byteCh)
	if !ok {
		return fileIOError("Byte channel is not initialized")
	}
	if _, err := writer.Write(data); err != nil {
		return fileIOError("error occurred while writing to the channel. " + err.Error())
	}
	return nil
}

func dataChannelInitChannel(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	byteCh, _ := args[1].(*values.Object)
	order, _ := args[2].(string)
	self.Put("$byteChannel", byteCh)
	self.Put("$order", order)
	return nil, nil
}

func dataChannelClose(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	markClosed(self)
	if err := closeUnderlyingByteChannel(self); err != nil {
		return fileIOError("error occurred while closing the channel. " + err.Error()), nil
	}
	return nil, nil
}

// readFixedInt reads a width-byte signed integer, shared by
// readableDataChannelReadInt16/32/64.
func readFixedInt(args []values.BalValue, width int) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	data, err := readDataBytes(self, width)
	if err != nil {
		return nil, err
	}
	return decodeDataInt(width, data), nil
}

func readableDataChannelReadInt16(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return readFixedInt(args, 2)
}

func readableDataChannelReadInt32(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return readFixedInt(args, 4)
}

func readableDataChannelReadInt64(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return readFixedInt(args, 8)
}

func readableDataChannelReadFloat32(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	data, err := readDataBytes(self, 4)
	if err != nil {
		return nil, err
	}
	return float64(math.Float32frombits(uint32(decodeDataInt(4, data)))), nil
}

func readableDataChannelReadFloat64(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	data, err := readDataBytes(self, 8)
	if err != nil {
		return nil, err
	}
	return math.Float64frombits(uint64(decodeDataInt(8, data))), nil
}

func readableDataChannelReadBool(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	data, err := readDataBytes(self, 1)
	if err != nil {
		return nil, err
	}
	return data[0] == 1, nil
}

func readableDataChannelReadString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	nBytes, _ := args[1].(int64)
	charset, _ := args[2].(string)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	if eofReached(self) {
		return byteChannelEofError(), nil
	}
	if nBytes < 0 {
		return fileIOError(fmt.Sprintf("invalid number of bytes: %d", nBytes)), nil
	}
	enc, err := lookupCharset(charset)
	if err != nil {
		return fileIOError("Error occurred while reading string: " + err.Error()), nil
	}
	byteCh, _ := byteChannelOf(self)
	reader, ok := readerOf(byteCh)
	if !ok {
		return nil, fmt.Errorf("byte channel is not initialized")
	}
	result := make([]byte, 0, min(int(nBytes), channelBufferSize))
	remaining := int(nBytes)
	for remaining > 0 {
		chunk := make([]byte, min(remaining, channelBufferSize))
		n, readErr := io.ReadFull(reader, chunk)
		result = append(result, chunk[:n]...)
		remaining -= n
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			self.Put("$eof", true)
			break
		}
		if readErr != nil {
			return fileIOError("Error occurred while reading string: " + readErr.Error()), nil
		}
	}
	decoded, decErr := enc.NewDecoder().Bytes(result)
	if decErr != nil {
		return fileIOError("Error occurred while reading string: " + decErr.Error()), nil
	}
	return string(decoded), nil
}

func readableDataChannelReadVarInt(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	byteCh, _ := byteChannelOf(self)
	reader, ok := readerOf(byteCh)
	if !ok {
		return nil, fmt.Errorf("byte channel is not initialized")
	}
	var groups []byte
	one := make([]byte, 1)
	for {
		if len(groups) == 10 {
			return nil, fmt.Errorf("variable-length integer is longer than 10 bytes")
		}
		if _, err := io.ReadFull(reader, one); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				self.Put("$eof", true)
				return nil, fmt.Errorf("no bytes available to read from the channel")
			}
			return nil, fmt.Errorf("error occurred while reading from the channel: %s", err.Error())
		}
		groups = append(groups, one[0]&0x7F)
		if one[0]&0x80 == 0 {
			break
		}
	}
	if littleEndianOf(self) {
		for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
			groups[i], groups[j] = groups[j], groups[i]
		}
	}
	var value int64
	for _, g := range groups {
		value = value<<7 | int64(g)
	}
	signBit := uint(7*len(groups)) - 1
	if signBit < 63 && value>>signBit&1 == 1 {
		value |= int64(-1) << signBit
	}
	return value, nil
}

// writeFixedInt writes value as a width-byte signed integer, shared by
// writableDataChannelWriteInt16/32/64.
func writeFixedInt(args []values.BalValue, width int) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	value, _ := args[1].(int64)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	return writeDataBytes(self, encodeDataInt(value, width, littleEndianOf(self))), nil
}

func writableDataChannelWriteInt16(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return writeFixedInt(args, 2)
}

func writableDataChannelWriteInt32(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return writeFixedInt(args, 4)
}

func writableDataChannelWriteInt64(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return writeFixedInt(args, 8)
}

func writableDataChannelWriteFloat32(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	value, _ := args[1].(float64)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	bits := int64(math.Float32bits(float32(value)))
	return writeDataBytes(self, encodeDataInt(bits, 4, littleEndianOf(self))), nil
}

func writableDataChannelWriteFloat64(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	value, _ := args[1].(float64)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	bits := int64(math.Float64bits(value))
	return writeDataBytes(self, encodeDataInt(bits, 8, littleEndianOf(self))), nil
}

func writableDataChannelWriteBool(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	value, _ := args[1].(bool)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	b := byte(0)
	if value {
		b = 1
	}
	return writeDataBytes(self, []byte{b}), nil
}

func writableDataChannelWriteString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	value, _ := args[1].(string)
	charset, _ := args[2].(string)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	enc, err := lookupCharset(charset)
	if err != nil {
		// A Go error return panics in the interpreter, matching
		// jBallerina, where the unchecked charset exception escapes.
		return nil, err
	}
	data, encErr := enc.NewEncoder().Bytes([]byte(value))
	if encErr != nil {
		return fileIOError("error occurred while writing to the channel. " + encErr.Error()), nil
	}
	return writeDataBytes(self, data), nil
}

func writableDataChannelWriteVarInt(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	value, _ := args[1].(int64)
	if isClosed(self) {
		return dataChannelClosedError(), nil
	}
	return writeDataBytes(self, encodeVarInt(value, littleEndianOf(self))), nil
}

func initDataChannelModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.initChannel", dataChannelInitChannel)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.initChannel", dataChannelInitChannel)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readInt16", readableDataChannelReadInt16)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readInt32", readableDataChannelReadInt32)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readInt64", readableDataChannelReadInt64)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readFloat32", readableDataChannelReadFloat32)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readFloat64", readableDataChannelReadFloat64)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readBool", readableDataChannelReadBool)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readString", readableDataChannelReadString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.readVarInt", readableDataChannelReadVarInt)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableDataChannel.close", dataChannelClose)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeInt16", writableDataChannelWriteInt16)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeInt32", writableDataChannelWriteInt32)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeInt64", writableDataChannelWriteInt64)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeFloat32", writableDataChannelWriteFloat32)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeFloat64", writableDataChannelWriteFloat64)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeBool", writableDataChannelWriteBool)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeString", writableDataChannelWriteString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.writeVarInt", writableDataChannelWriteVarInt)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableDataChannel.close", dataChannelClose)
}

func init() {
	runtime.RegisterModuleInitializer(initDataChannelModule)
}
