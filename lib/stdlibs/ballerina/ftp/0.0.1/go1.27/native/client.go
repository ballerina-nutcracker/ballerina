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
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	clientStateField = "$ftp"
	chunkSize        = 8192
)

var errClientClosed = newError(kindError, "FTP Client is already closed, hence further operations are not allowed")

type clientState struct {
	mu           sync.Mutex
	remote       *remoteClient
	retry        *retrySettings
	breaker      *circuitBreaker
	lax          bool
	legacyStream io.ReadCloser
}

func (s *clientState) active() (*remoteClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.remote == nil {
		return nil, errClientClosed
	}
	return s.remote, nil
}

func stateOf(obj *values.Object) *clientState {
	v, _ := obj.Get(clientStateField)
	state, _ := v.(*clientState)
	if state == nil {
		return &clientState{}
	}
	return state
}

func clientArg(args []values.BalValue) *clientState {
	obj, _ := args[0].(*values.Object)
	return stateOf(obj)
}

// fetch reads a whole file through the circuit breaker, as FtpClient.getAllContent does.
func (s *clientState) fetch(p string) ([]byte, error) {
	return guard(s.breaker, func() ([]byte, error) {
		remote, err := s.active()
		if err != nil {
			return nil, err
		}
		return remote.readAll(p)
	})
}

// fetchConverted runs fetch and convert together under the retry policy: jBallerina retries
// conversion failures as well, but only the transfer counts towards the circuit breaker.
func fetchConverted[T any](ctx *extern.Context, s *clientState, name, p string, convert func([]byte) (T, error)) (T, error) {
	return withRetry(ctx, s.retry, name, func() (T, error) {
		data, err := s.fetch(p)
		if err != nil {
			var zero T
			return zero, err
		}
		return convert(data)
	})
}

func (s *clientState) run(op func(remote *remoteClient) error) error {
	_, err := guard(s.breaker, func() (struct{}, error) {
		remote, err := s.active()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, op(remote)
	})
	return err
}

func decodeUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	var b strings.Builder
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		b.WriteRune(r)
		data = data[size:]
	}
	return b.String()
}

func (m *ftpModule) registerClientExterns() {
	register := func(name string, fn extern.NativeFunc) {
		runtime.RegisterExternFunction(m.rt, orgName, moduleName, name, fn)
	}
	register("initEndpoint", m.initEndpoint)
	register("getBytes", m.getBytes)
	register("getText", m.getText)
	register("Client.$remote$getJson", m.getJson)
	register("Client.$remote$getXml", m.getXml)
	register("Client.$remote$getCsv", m.getCsv)
	register("Client.$remote$getBytesAsStream", m.getBytesAsStream)
	register("Client.$remote$putBytes", m.putBytes)
	register("Client.$remote$putText", m.putText)
	register("Client.$remote$putBytesAsStream", m.putBytesAsStream)
	register("Client.$remote$putCsvAsStream", m.putCsvAsStream)
	register("Client.$remote$close", m.closeClient)
	register("putJson", m.putJSON)
	register("putXml", m.putXML)
	register("putCsv", m.putCsv)
	register("put", m.put)
	register("append", m.append)
	register("delete", m.simple(func(r *remoteClient, p string) error { return r.delete(p) }))
	register("mkdir", m.simple(func(r *remoteClient, p string) error { return r.mkdir(p) }))
	register("rmdir", m.simple(func(r *remoteClient, p string) error { return r.rmdir(p) }))
	register("rename", m.twoPath(func(r *remoteClient, a, b string) error { return r.rename(a, b) }))
	register("move", m.twoPath(func(r *remoteClient, a, b string) error { return r.rename(a, b) }))
	register("copy", m.twoPath(func(r *remoteClient, a, b string) error { return r.copy(a, b) }))
	register("exists", m.exists)
	register("size", m.size)
	register("list", m.list)
	register("isDirectory", m.isDirectory)
	register("externInitialGetStreamEntryRecord", m.getFirst)
	register("externGetStreamEntryRecord", m.getNext)
	register("externCloseInputStream", m.closeLegacyStream)
	register("toJsonString", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return toJSONString(args[0]), nil
	})
}

func (m *ftpModule) initEndpoint(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	obj := args[0].(*values.Object)
	config := args[1].(*values.Map)
	state := &clientState{lax: getBool(config, "laxDataBinding")}
	retry, err := validateRetryConfig(config)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	state.retry = retry
	platform := m.rt.Platform()
	ep, err := parseEndpoint(config, platform)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	if cbConfig := getMap(config, "circuitBreaker"); cbConfig != nil {
		breaker, err := newCircuitBreaker(cbConfig, platform.Time.Now)
		if err != nil {
			return m.errorResult(ctx, err)
		}
		state.breaker = breaker
	}
	remote, err := connect(platform, ep)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	state.remote = remote
	obj.Put(clientStateField, state)
	return nil, nil
}

func (m *ftpModule) getBytes(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	p := args[1].(string)
	data, err := fetchConverted(ctx, clientArg(args), "getBytes", p, func(d []byte) ([]byte, error) { return d, nil })
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return m.types.bytes(data), nil
}

func (m *ftpModule) getText(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	p := args[1].(string)
	text, err := fetchConverted(ctx, clientArg(args), "getText", p, func(d []byte) (string, error) {
		return decodeUTF8(d), nil
	})
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return text, nil
}

func targetType(arg values.BalValue) semtypes.SemType {
	return arg.(*values.TypeDesc).Constraint()
}

func (m *ftpModule) getJson(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state, p, target := clientArg(args), args[1].(string), targetType(args[2])
	value, err := fetchConverted(ctx, state, "getJson", p, func(d []byte) (values.BalValue, error) {
		return bindJSON(ctx.TypeCtx(), m.types, d, target, p, state.lax)
	})
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return value, nil
}

func (m *ftpModule) getXml(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	p, target := args[1].(string), targetType(args[2])
	value, err := fetchConverted(ctx, clientArg(args), "getXml", p, func(d []byte) (values.BalValue, error) {
		return bindXML(ctx.TypeCtx(), d, target, p)
	})
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return value, nil
}

func (m *ftpModule) getCsv(ctx *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return m.errorResult(ctx, newError(kindError, "Reading CSV content is not yet supported"))
}

// contentStream exposes a reader as a stream<byte[], error?> of chunks of up to 8192 bytes.
func (m *ftpModule) contentStream(r io.ReadCloser) *values.Stream {
	var once sync.Once
	closeReader := func() error {
		var err error
		once.Do(func() { err = r.Close() })
		return err
	}
	next := func() values.BalValue {
		buf := make([]byte, chunkSize)
		n, err := r.Read(buf)
		for n == 0 && err == nil {
			n, err = r.Read(buf)
		}
		if n > 0 {
			return values.NewMap(m.types.byteRecordTy, m.types.byteRecordAtom, false,
				[]values.MapEntry{{Key: "value", Value: m.types.bytes(buf[:n])}})
		}
		if err == io.EOF {
			if closeErr := closeReader(); closeErr != nil {
				return values.NewErrorWithMessage("Unable to clean input stream value")
			}
			return nil
		}
		_ = closeReader()
		return values.NewErrorWithMessage("Unable to parse value")
	}
	closeFn := func() values.BalValue {
		if err := closeReader(); err != nil {
			return values.NewErrorWithMessage("Unable to clean input stream value")
		}
		return nil
	}
	return values.NewStream(m.types.byteStreamTy, next, closeFn)
}

func (m *ftpModule) getBytesAsStream(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state, p := clientArg(args), args[1].(string)
	r, err := guard(state.breaker, func() (io.ReadCloser, error) {
		remote, err := state.active()
		if err != nil {
			return nil, err
		}
		return remote.openFile(p)
	})
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return m.contentStream(r), nil
}

func writeOption(arg values.BalValue) bool {
	option, _ := arg.(string)
	return option == "APPEND"
}

func (m *ftpModule) writeContent(ctx *extern.Context, args []values.BalValue, data []byte, appendMode bool) (values.BalValue, error) {
	p := args[1].(string)
	return m.errorResult(ctx, clientArg(args).run(func(remote *remoteClient) error {
		return remote.writeBytes(p, data, appendMode)
	}))
}

func (m *ftpModule) putBytes(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.writeContent(ctx, args, args[2].(*values.List).ToByteSlice(), writeOption(args[3]))
}

func (m *ftpModule) putText(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.writeContent(ctx, args, []byte(args[2].(string)), writeOption(args[3]))
}

func (m *ftpModule) putJSON(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.writeContent(ctx, args, []byte(args[2].(string)), writeOption(args[3]))
}

func (m *ftpModule) putXML(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.writeContent(ctx, args, []byte(args[2].(values.XMLValue).XMLString()), writeOption(args[3]))
}

func (m *ftpModule) putCsv(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	appendMode := writeOption(args[3])
	content := csvContent(args[2].(*values.List), !appendMode, m.rt.Platform().OS.LineSeparator())
	return m.writeContent(ctx, args, []byte(content), appendMode)
}

// iteratorError is a failure of the source stream itself, reported as is rather than as a
// transfer failure.
type iteratorError struct {
	message string
}

func (e *iteratorError) Error() string { return e.message }

// streamReader pulls a Ballerina stream lazily into an io.Reader. A zero-length chunk ends the
// upload, as SequenceInputStream does in jBallerina.
type streamReader struct {
	stream *values.Stream
	encode func(values.BalValue) []byte
	buf    []byte
	done   bool
}

func (r *streamReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		if r.done {
			return 0, io.EOF
		}
		elem := r.stream.Next()
		switch v := elem.(type) {
		case nil:
			r.done = true
			return 0, io.EOF
		case *values.Error:
			r.done = true
			return 0, &iteratorError{message: "Iterator error: " + v.Message}
		case *values.Map:
			value, _ := v.Get("value")
			r.buf = r.encode(value)
			if len(r.buf) == 0 {
				r.done = true
				return 0, io.EOF
			}
		}
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func (m *ftpModule) writeStream(ctx *extern.Context, args []values.BalValue, encode func(values.BalValue) []byte) (values.BalValue, error) {
	p := args[1].(string)
	reader := &streamReader{stream: args[2].(*values.Stream), encode: encode}
	return m.errorResult(ctx, clientArg(args).run(func(remote *remoteClient) error {
		err := remote.write(p, reader, writeOption(args[3]))
		var iterErr *iteratorError
		if errors.As(err, &iterErr) {
			return newError(kindError, iterErr.message)
		}
		return err
	}))
}

func (m *ftpModule) putBytesAsStream(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.writeStream(ctx, args, func(v values.BalValue) []byte {
		return v.(*values.List).ToByteSlice()
	})
}

func (m *ftpModule) putCsvAsStream(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	first := !writeOption(args[3])
	lineSeparator := m.rt.Platform().OS.LineSeparator()
	return m.writeStream(ctx, args, func(v values.BalValue) []byte {
		var row string
		switch t := v.(type) {
		case *values.List:
			row = csvListRow(t)
		case *values.Map:
			keys := t.Keys()
			row = csvRecordRow(t, keys)
			if first {
				row = csvHeaderRow(keys) + lineSeparator + row
			}
		}
		first = false
		return []byte(row + lineSeparator)
	})
}

func (m *ftpModule) closeClient(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state := clientArg(args)
	state.mu.Lock()
	remote := state.remote
	state.remote = nil
	state.mu.Unlock()
	if remote == nil {
		return nil, nil
	}
	if err := remote.close(); err != nil {
		return m.errorResult(ctx, newError(kindError, "Error occurred while closing the FTP client: "+err.Error()))
	}
	return nil, nil
}

func (m *ftpModule) simple(op func(remote *remoteClient, p string) error) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		p := args[1].(string)
		return m.errorResult(ctx, clientArg(args).run(func(remote *remoteClient) error { return op(remote, p) }))
	}
}

func (m *ftpModule) twoPath(op func(remote *remoteClient, src, dest string) error) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		src, dest := args[1].(string), args[2].(string)
		return m.errorResult(ctx, clientArg(args).run(func(remote *remoteClient) error { return op(remote, src, dest) }))
	}
}

func query[T any](state *clientState, op func(remote *remoteClient) (T, error)) (T, error) {
	return guard(state.breaker, func() (T, error) {
		remote, err := state.active()
		if err != nil {
			var zero T
			return zero, err
		}
		return op(remote)
	})
}

func (m *ftpModule) exists(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	p := args[1].(string)
	ok, err := query(clientArg(args), func(r *remoteClient) (bool, error) { return r.exists(p) })
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return ok, nil
}

func (m *ftpModule) size(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	p := args[1].(string)
	n, err := query(clientArg(args), func(r *remoteClient) (int64, error) { return r.size(p) })
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return n, nil
}

func (m *ftpModule) isDirectory(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	p := args[1].(string)
	ok, err := query(clientArg(args), func(r *remoteClient) (bool, error) { return r.isDirectory(p) })
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return ok, nil
}

func (m *ftpModule) list(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state, p := clientArg(args), args[1].(string)
	var ep *endpoint
	entries, err := query(state, func(r *remoteClient) ([]remoteEntry, error) {
		ep = r.ep
		return r.list(p)
	})
	if err != nil {
		return m.errorResult(ctx, err)
	}
	m.types.resolveRecordTypes(ctx)
	dir := cleanPath(p)
	items := make([]values.BalValue, len(entries))
	for i, entry := range entries {
		items[i] = clientFileInfo(m.types, ctx.TypeCtx(), ep, path.Join(dir, entry.name), entry)
	}
	return m.types.fileInfoList(items), nil
}

func (m *ftpModule) readonlyChunk(data []byte) *values.Map {
	return values.NewMap(m.types.roRecordTy, m.types.roRecordAtom, false,
		[]values.MapEntry{{Key: "value", Value: m.types.readonlyBytes(data)}})
}

func readChunk(r io.Reader) ([]byte, error) {
	buf := make([]byte, chunkSize)
	n, err := r.Read(buf)
	for n == 0 && err == nil {
		n, err = r.Read(buf)
	}
	return buf[:n], err
}

func (m *ftpModule) getFirst(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state, p := clientArg(args), args[1].(string)
	remote, err := state.active()
	if err != nil {
		return m.errorResult(ctx, err)
	}
	r, err := remote.openFile(p)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	state.mu.Lock()
	previous := state.legacyStream
	state.legacyStream = r
	state.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return m.nextLegacyChunk(state)
}

func (m *ftpModule) nextLegacyChunk(state *clientState) (values.BalValue, error) {
	state.mu.Lock()
	r := state.legacyStream
	state.mu.Unlock()
	if r == nil {
		return nil, nil
	}
	data, err := readChunk(r)
	if len(data) > 0 {
		return m.readonlyChunk(data), nil
	}
	state.mu.Lock()
	if state.legacyStream == r {
		state.legacyStream = nil
	}
	state.mu.Unlock()
	closeErr := r.Close()
	if err == io.EOF {
		err = closeErr
	}
	if err != nil {
		return values.NewErrorWithMessage("Error occurred while reading the file: " + err.Error()), nil
	}
	return nil, nil
}

func (m *ftpModule) getNext(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.nextLegacyChunk(clientArg(args))
}

func (m *ftpModule) closeLegacyStream(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	state := clientArg(args)
	state.mu.Lock()
	r := state.legacyStream
	state.legacyStream = nil
	state.mu.Unlock()
	if r != nil {
		if err := r.Close(); err != nil {
			return values.NewErrorWithMessage("Error occurred while closing the stream: " + err.Error()), nil
		}
	}
	return nil, nil
}

// compressedName mirrors FtpUtil.getCompressedFileName, which cuts the whole path at its last
// dot and panics when there is none.
func compressedName(p string) (string, error) {
	i := strings.LastIndexByte(p, '.')
	if i < 0 {
		return "", fmt.Errorf("begin 0, end -1, length %d", len(p))
	}
	return p[:i] + ".zip", nil
}

func zipContent(name string, data []byte) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create(name)
	_, _ = f.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

func inputBytes(input *values.Map) ([]byte, error) {
	if !getBool(input, "isFile") {
		text, _ := getString(input, "textContent")
		return []byte(text), nil
	}
	v, _ := input.Get("fileContent")
	s, ok := v.(*values.Stream)
	if !ok {
		return nil, newError(kindError, "Error while reading a file")
	}
	var buf bytes.Buffer
	for {
		elem := s.Next()
		switch e := elem.(type) {
		case nil:
			return buf.Bytes(), nil
		case *values.Error:
			return nil, newError(kindError, "Error while reading a file: "+e.Message)
		case *values.Map:
			value, _ := e.Get("value")
			buf.Write(value.(*values.List).ToByteSlice())
		}
	}
}

func (m *ftpModule) legacyWrite(ctx *extern.Context, args []values.BalValue, appendMode bool) (values.BalValue, error) {
	state, input := clientArg(args), args[1].(*values.Map)
	p, _ := getString(input, "filePath")
	data, err := inputBytes(input)
	if err != nil {
		return m.errorResult(ctx, err)
	}
	if getBool(input, "compressInput") {
		name, err := compressedName(p)
		if err != nil {
			return nil, err
		}
		data = zipContent(path.Base(p), data)
		p = name
	}
	remote, err := state.active()
	if err != nil {
		return m.errorResult(ctx, err)
	}
	return m.errorResult(ctx, remote.writeBytes(p, data, appendMode))
}

func (m *ftpModule) put(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.legacyWrite(ctx, args, false)
}

func (m *ftpModule) append(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return m.legacyWrite(ctx, args, true)
}
