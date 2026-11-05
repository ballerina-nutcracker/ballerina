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

// Byte-stream and Server-Sent Event bodies, mirroring jBallerina's getByteStream/setByteStream,
// bytes_to_sse_stream_generator.bal and sse_to_bytes_stream_generator.bal, with the behaviour of
// module-ballerina-http#2752.

package native

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// eventStreamReadSize matches #2752's AVAILABLE_BYTES_ARRAY_SIZE: each SSE read takes whatever
// has arrived, up to 8 KB.
const eventStreamReadSize = 8192

// streamBody is the streaming state shared by request and response bodies.
type streamBody struct {
	source     *values.Stream // a body set with setByteStream
	fromWire   bool           // received from the network, so getByteStream can read it
	reader     io.ReadCloser  // the wire body once getByteStream or getSseEventStream has taken it
	framer     *sseFramer
	noCache    bool  // set by setSseEventStream; Cache-Control is added when the response is sent
	pendingErr error // a read failure that arrived with the last block, returned by the next read
}

func (b *streamBody) closeReader() error {
	if b.reader == nil {
		return nil
	}
	r := b.reader
	b.reader = nil
	return r.Close()
}

// takeWire hands the unread body over to a byte stream. The second result is false when a
// payload getter has already read the body.
func (h *requestBodyHolder) takeWire() (io.ReadCloser, bool) {
	var r io.ReadCloser
	taken := false
	h.once.Do(func() {
		taken = true
		r = wireReader(h.stream, h.buf, nil)
		h.stream = nil
		h.buf = []byte{}
	})
	return r, taken
}

func (h *responseBodyHolder) takeWire() (io.ReadCloser, bool) {
	var r io.ReadCloser
	taken := false
	h.once.Do(func() {
		taken = true
		r = wireReader(h.stream, h.buf, h.readErr)
		h.stream = nil
		h.buf = []byte{}
	})
	return r, taken
}

// wireReader returns nil for an empty body, which jBallerina has no byte channel for. A body
// buffered eagerly ends with the error that cut it short, if any.
func wireReader(stream io.ReadCloser, buf []byte, readErr error) io.ReadCloser {
	if stream != nil {
		return stream
	}
	if readErr != nil {
		return io.NopCloser(io.MultiReader(bytes.NewReader(buf), errorReader{readErr}))
	}
	if len(buf) > 0 {
		return io.NopCloser(bytes.NewReader(buf))
	}
	return nil
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

// readBlock reads up to size bytes of the wire body, or nil at its end. A read error that
// arrives with data is kept and returned by the next call, so a truncated body never ends
// as a clean stream.
func (b *streamBody) readBlock(size int64) ([]byte, error) {
	if b.reader == nil {
		return nil, nil
	}
	if b.pendingErr != nil {
		err := b.pendingErr
		b.pendingErr = nil
		_ = b.closeReader()
		return nil, err
	}
	buf := make([]byte, size)
	n := 0
	var err error
	for n < len(buf) && err == nil {
		var m int
		m, err = b.reader.Read(buf[n:])
		n += m
	}
	if n > 0 {
		if err != nil && !errors.Is(err, io.EOF) {
			b.pendingErr = err
		}
		return buf[:n], nil
	}
	_ = b.closeReader()
	if errors.Is(err, io.EOF) {
		return nil, nil
	}
	return nil, err
}

type messageBody interface {
	takeWire() (io.ReadCloser, bool)
	state() *streamBody
	hasData() bool
	hasParts() bool
}

func (h *requestBodyHolder) state() *streamBody  { return &h.streamBody }
func (h *responseBodyHolder) state() *streamBody { return &h.streamBody }
func (h *requestBodyHolder) hasData() bool       { return len(h.buf) > 0 }
func (h *responseBodyHolder) hasData() bool      { return len(h.buf) > 0 }
func (b *bodyPartsBase) hasParts() bool          { return b.parts != nil }

// bodyOf returns the body of an http:Request or http:Response, and which of the two it is for
// error messages.
func bodyOf(msg *values.Object) (messageBody, string) {
	if v, ok := msg.Get("$body"); ok {
		if h, ok := v.(*requestBodyHolder); ok {
			return h, "request"
		}
	}
	if v, ok := msg.Get("body"); ok {
		if h, ok := v.(*responseBodyHolder); ok {
			return h, "response"
		}
	}
	return nil, ""
}

// balStreamReader reads a Ballerina stream<byte[], io:Error?>. Read calls the stream's next(),
// so it must only be used on a strand's goroutine.
type balStreamReader struct {
	stream  *values.Stream
	pending []byte
	err     error
}

func newBalStreamReader(s *values.Stream) *balStreamReader {
	return &balStreamReader{stream: s}
}

func (r *balStreamReader) Read(p []byte) (int, error) {
	for len(r.pending) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		switch elem := r.stream.Next().(type) {
		case nil:
			r.err = io.EOF
		case *values.Error:
			r.err = &streamElementError{elem}
		case *values.Map:
			value, _ := elem.Get("value")
			if list, ok := value.(*values.List); ok {
				r.pending = list.ToByteSlice()
			}
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

// streamElementError carries an error value the Ballerina stream returned.
type streamElementError struct{ value *values.Error }

func (e *streamElementError) Error() string { return e.value.Message }

// outboundBody is a request body: in memory or a network stream (reader), or a Ballerina byte
// stream (source).
type outboundBody struct {
	reader io.Reader
	length int64
	source *values.Stream
}

func sendRequest(ctx *extern.Context, client pal.HTTPClient, method, url string, body outboundBody,
	contentType string, headers map[string][]string) (int, map[string][]string, io.ReadCloser, error) {
	if body.source != nil {
		return executeStreamed(goCtxOrBackground(ctx), client, method, url, body.source, contentType, headers)
	}
	return client.Execute(goCtxOrBackground(ctx), method, url, body.reader, body.length, contentType, headers)
}

var errStreamPanicked = errors.New("request body stream panicked")

var errUploadRejected = errors.New("request body not sent after an error response")

// executeStreamed drains a Ballerina byte stream on the calling strand into a pipe the transport
// reads from, so the stream's next() never runs on a transport goroutine. The body is sent
// chunked, and keeps flowing after a success response arrives, until the stream ends or the
// transport stops reading it; an error response ends it, as the server will not read the rest.
func executeStreamed(goCtx context.Context, client pal.HTTPClient, method, url string, source *values.Stream,
	contentType string, headers map[string][]string) (int, map[string][]string, io.ReadCloser, error) {
	type result struct {
		status  int
		headers map[string][]string
		body    io.ReadCloser
		err     error
	}
	pr, pw := io.Pipe()
	done := make(chan result, 1)
	go func() {
		status, respHeaders, respBody, err := client.Execute(goCtx, method, url, pr, -1, contentType, headers)
		switch {
		case err != nil:
			_ = pr.CloseWithError(err)
		case status >= http.StatusMultipleChoices:
			_ = pr.CloseWithError(errUploadRejected)
		}
		done <- result{status, respHeaders, respBody, err}
	}()
	defer func() {
		if rec := recover(); rec != nil {
			_ = pw.CloseWithError(errStreamPanicked)
			if res := <-done; res.body != nil {
				_ = res.body.Close()
			}
			panic(rec)
		}
	}()
	reader := newBalStreamReader(source)
	buf := make([]byte, 32*1024)
	var streamErr error
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if _, writeErr := pw.Write(buf[:n]); writeErr != nil {
				break
			}
		}
		if errors.Is(err, io.EOF) {
			_ = pw.Close()
			break
		}
		if err != nil {
			streamErr = err
			_ = pw.CloseWithError(err)
			break
		}
	}
	res := <-done
	if streamErr != nil {
		if res.body != nil {
			_ = res.body.Close()
		}
		return 0, nil, nil, errors.New("Error occurred while streaming content: " + streamErr.Error())
	}
	return res.status, res.headers, res.body, res.err
}

// streamBodyPanic is a panic raised by a response body stream's next() after the headers were
// sent, so the handler logs it and aborts the response instead of writing an error body.
type streamBodyPanic struct{ value any }

// writeStreamedBody writes a byte-stream response body. An event stream is flushed after each
// event and closed at the end, as jBallerina's EventStreamWriter does; any other byte stream is
// left open, as jBallerina never closes a byte stream it sent as a body. Each write gets the
// listener timeout afresh, so a long-lived stream is only cut off when a write stalls. The
// deadline is cleared while the stream produces the next block, since on HTTP/2 an expired
// deadline resets the stream even with no write in progress.
func writeStreamedBody(w http.ResponseWriter, source *values.Stream, contentType string, timeout time.Duration) {
	defer func() {
		if rec := recover(); rec != nil {
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			panic(streamBodyPanic{rec})
		}
	}()
	eventStream := isEventStreamType(contentType)
	rc := http.NewResponseController(w)
	setDeadline := func(deadline time.Time) {
		if timeout > 0 {
			_ = rc.SetWriteDeadline(deadline)
		}
	}
	// The handler's final flush runs under the last deadline set here.
	defer func() { setDeadline(time.Now().Add(timeout)) }()
	reader := newBalStreamReader(source)
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			setDeadline(time.Now().Add(timeout))
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				break
			}
			if eventStream {
				_ = rc.Flush()
			}
			setDeadline(time.Time{})
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// Aborting tells the client the body is incomplete instead of ending it cleanly.
			panic(http.ErrAbortHandler)
		}
	}
	if eventStream && source.Close != nil {
		source.Close()
	}
}

// sseFramer splits a text/event-stream body into events, following #2752's
// BytesToEventStreamGenerator: an event ends at a blank line, that is a line break ('\r\n', '\n'
// or '\r') followed by another, in any mix. Bytes of an unfinished event carry over to the next
// read, so a multi-byte character split across reads stays intact. A UTF-8 byte order mark
// at the start of the body is dropped, as the WHATWG event-stream format requires.
type sseFramer struct {
	reader     io.Reader
	readBuf    []byte
	pending    []byte
	eventStart int
	scanFrom   int
	// The last event ended at a '\r', so a '\n' right after it completes that line break.
	skipLineFeed bool
	bomChecked   bool
	eof          bool
	err          error
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func newSSEFramer(r io.Reader) *sseFramer {
	return &sseFramer{reader: r, readBuf: make([]byte, eventStreamReadSize)}
}

// next returns the next event's bytes including its terminating line breaks, or nil at the end
// of the body, discarding an unfinished last event and, as #2752 does, blocks of only line breaks.
// A read error is returned once the events completed before it have been handed on.
func (f *sseFramer) next() ([]byte, error) {
	for {
		if !f.skipBOM() {
			if err := f.read(); err != nil {
				return nil, err
			}
			continue
		}
		if end, ok := f.findEventEnd(); ok {
			event := f.pending[f.eventStart:end]
			f.eventStart = end
			if len(bytes.Trim(event, "\r\n")) == 0 {
				continue
			}
			return bytes.Clone(event), nil
		}
		if f.err != nil {
			return nil, f.err
		}
		if f.eof {
			return nil, nil
		}
		if f.eventStart > 0 {
			f.pending = append(f.pending[:0], f.pending[f.eventStart:]...)
			f.scanFrom -= f.eventStart
			f.eventStart = 0
		}
		if err := f.read(); err != nil {
			return nil, err
		}
	}
}

// read appends the next read to pending, returning an error only when nothing came with it.
func (f *sseFramer) read() error {
	if f.err != nil {
		return f.err
	}
	if f.eof {
		return nil
	}
	n, err := f.reader.Read(f.readBuf)
	f.pending = append(f.pending, f.readBuf[:n]...)
	if errors.Is(err, io.EOF) {
		f.eof = true
	} else if err != nil {
		f.err = err
		if n == 0 {
			return err
		}
	}
	return nil
}

// skipBOM drops a leading byte order mark, reporting false while too few bytes have arrived
// to tell whether the body starts with one.
func (f *sseFramer) skipBOM() bool {
	if f.bomChecked {
		return true
	}
	if len(f.pending) < len(utf8BOM) && bytes.HasPrefix(utf8BOM, f.pending) && !f.eof && f.err == nil {
		return false
	}
	if bytes.HasPrefix(f.pending, utf8BOM) {
		f.eventStart = len(utf8BOM)
		f.scanFrom = len(utf8BOM)
	}
	f.bomChecked = true
	return true
}

func (f *sseFramer) findEventEnd() (int, bool) {
	if f.skipLineFeed && f.eventStart < len(f.pending) {
		if f.pending[f.eventStart] == '\n' {
			f.eventStart++
		}
		f.skipLineFeed = false
	}
	start := f.eventStart
	for i := max(f.scanFrom, start); i < len(f.pending); i++ {
		current := f.pending[i]
		if i > start && (current == '\n' || current == '\r') {
			previous := f.pending[i-1]
			if previous == '\n' || (previous == '\r' && current == '\r') {
				f.scanFrom = i + 1
				f.skipLineFeed = current == '\r'
				return i + 1, true
			}
		}
	}
	f.scanFrom = len(f.pending)
	return 0, false
}

// splitLines splits on '\r\n', '\n' or '\r', keeping empty lines, like Ballerina's
// re `\r\n|\n|\r`.split.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n':
			lines = append(lines, s[start:i])
			start = i + 1
		case '\r':
			lines = append(lines, s[start:i])
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			start = i + 1
		}
	}
	return append(lines, s[start:])
}

// parseSSEEvent fills target with the fields of one event, following #2752's parseSseEvent:
// field names are trimmed, one leading space is removed from a value, data lines are joined
// with '\n', a non-integer retry leaves retry absent, and unknown fields are ignored.
func parseSSEEvent(tc semtypes.Context, event string, target *values.Map) {
	var id, comment, data, eventName *string
	var retry *int64
	for _, line := range splitLines(event) {
		if line == "" {
			continue
		}
		name, value := line, ""
		if colon := strings.IndexByte(line, ':'); colon >= 0 {
			name = strings.Trim(line[:colon], " \t\n\v\f\r")
			value = strings.TrimPrefix(line[colon+1:], " ")
		}
		switch name {
		case "id":
			id = &value
		case "":
			comment = &value
		case "retry":
			retry = nil
			if n, err := strconv.ParseInt(value, 10, 64); err == nil {
				retry = &n
			}
		case "event":
			eventName = &value
		case "data":
			if data == nil {
				data = &value
			} else {
				joined := *data + "\n" + value
				data = &joined
			}
		}
	}
	putIfSet := func(key string, v *string) {
		if v != nil {
			target.Put(tc, key, *v)
		}
	}
	putIfSet("data", data)
	putIfSet("id", id)
	putIfSet("comment", comment)
	if retry != nil {
		target.Put(tc, "retry", *retry)
	}
	putIfSet("event", eventName)
}

// sseEventBytes serialises one event as jBallerina's getEventText does: comment, id, event,
// retry, then one data line per line of data, followed by a blank line.
func sseEventBytes(event *values.Map) []byte {
	var b strings.Builder
	field := func(name string) (string, bool) {
		v, ok := event.Get(name)
		if !ok {
			return "", false
		}
		s, ok := v.(string)
		return s, ok
	}
	if comment, ok := field("comment"); ok {
		for _, line := range splitLines(comment) {
			b.WriteString(": " + line + "\n")
		}
	}
	if id, ok := field("id"); ok {
		b.WriteString("id: " + id + "\n")
	}
	if name, ok := field("event"); ok {
		b.WriteString("event: " + name + "\n")
	}
	if v, ok := event.Get("retry"); ok {
		if n, ok := v.(int64); ok {
			b.WriteString("retry: " + strconv.FormatInt(n, 10) + "\n")
		}
	}
	if data, ok := field("data"); ok {
		for _, line := range splitLines(data) {
			b.WriteString("data: " + line + "\n")
		}
	}
	b.WriteString("\n")
	return []byte(b.String())
}

// isEventStreamType compares the media type case-insensitively and without its parameters,
// as RFC 9110 section 8.3.1 requires.
func isEventStreamType(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.EqualFold(strings.TrimSpace(mediaType), "text/event-stream")
}

func byteStreamError(kind string, cause *values.Error) *values.Error {
	return values.NewError(semtypes.Error, "Error occurred while retrieving the byte stream from the "+kind,
		cause, "GenericClientError", nil)
}

func mimeError(typeName, message string) *values.Error {
	return values.NewError(semtypes.Error, message, nil, typeName, nil)
}

// takeByteStream readies a body for reading as a stream, returning a stream set earlier with
// setByteStream, an error when the body cannot be streamed, or (nil, nil) once the wire body is
// ready to read. The checks follow mime's MimeEntityBody.getByteStream.
func takeByteStream(body messageBody, kind string) (*values.Stream, *values.Error) {
	st := body.state()
	if st.source != nil {
		return st.source, nil
	}
	if body.hasParts() {
		return nil, byteStreamError(kind, mimeError("ParserError",
			"Byte stream is not available since payload contains a set of body parts"))
	}
	if st.reader != nil {
		return nil, nil
	}
	if st.fromWire {
		r, taken := body.takeWire()
		if r != nil {
			st.reader = r
			return nil, nil
		}
		if !taken {
			return nil, byteStreamError(kind, dataSourceError())
		}
	} else if body.hasData() {
		return nil, byteStreamError(kind, dataSourceError())
	}
	return nil, byteStreamError(kind, mimeError("ParserError", "Byte stream is not available as payload"))
}

// streamedBodyNoContent mirrors jBallerina, whose text, json and xml payload getters see a body
// set with setByteStream as empty content.
func streamedBodyNoContent(bodyVal values.BalValue, dataKind string) *values.Error {
	var source *values.Stream
	switch h := bodyVal.(type) {
	case *requestBodyHolder:
		source = h.source
	case *responseBodyHolder:
		source = h.source
	}
	if source == nil {
		return nil
	}
	cause := values.NewError(semtypes.Error, "Error occurred while extracting "+dataKind+" data from entity",
		mimeError("NoContentError", "Empty content"), "ParserError", nil)
	return values.NewError(semtypes.Error, "No content", cause, "NoContentError", nil)
}

func dataSourceError() *values.Error {
	return mimeError("ParserError",
		"Byte stream is not available but payload can be obtain either as xml, json, string or byte[] type")
}

func registerStreamExterns(rt *runtime.Runtime, types *httpTypes) {
	setByteStream := func(isRequest bool) extern.NativeFunc {
		return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			source := args[1].(*values.Stream)
			tc := ctx.TypeCtx()
			if isRequest {
				holder := &requestBodyHolder{buf: []byte{}, contentLength: -1}
				holder.source = source
				self.Put("$body", holder)
				ct := payloadContentType(requestContentType(tc, self), optionalArg(args, 2), "application/octet-stream")
				setRequestHeader(self, "content-type", ct, tc)
				return nil, nil
			}
			holder := &responseBodyHolder{buf: []byte{}}
			holder.source = source
			self.Put("body", holder)
			ct := payloadContentType(responseContentType(self), optionalArg(args, 2), "application/octet-stream")
			responseHeaders(self).Put(tc, "content-type", newListValue([]values.BalValue{ct}))
			return nil, nil
		}
	}
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.setByteStream", setByteStream(true))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setByteStream", setByteStream(false))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetNoCache",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			if body, _ := bodyOf(args[0].(*values.Object)); body != nil {
				body.state().noCache = true
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetByteStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			body, kind := bodyOf(args[0].(*values.Object))
			arraySize := args[1].(int64)
			if arraySize <= 0 {
				return byteStreamError(kind, mimeError("GenericMimeError",
					"The array size of a byte stream must be positive, but was "+strconv.FormatInt(arraySize, 10))), nil
			}
			source, err := takeByteStream(body, kind)
			if err != nil {
				return err, nil
			}
			if source != nil {
				return source, nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externReadBodyBytes",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			body, _ := bodyOf(args[0].(*values.Object))
			block, err := body.state().readBlock(args[1].(int64))
			if err != nil {
				return mimeError("GenericError", "Error occurred while reading stream:"+err.Error()), nil
			}
			if block == nil {
				return nil, nil
			}
			return byteArrayValue(ctx, types, block), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCloseBody",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			body, _ := bodyOf(args[0].(*values.Object))
			st := body.state()
			st.framer = nil
			if err := st.closeReader(); err != nil {
				return mimeError("GenericError", "Error occurred while closing the stream: "+err.Error()), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPrepareSseEventStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			resp := args[0].(*values.Object)
			if !isEventStreamType(responseContentType(resp)) {
				return values.NewError(semtypes.Error,
					"invalid payload target type. The response is not of text/event-stream content type.",
					nil, "PayloadBindingClientError", nil), nil
			}
			body, kind := bodyOf(resp)
			st := body.state()
			if st.framer != nil {
				// A second event stream carries on from the events the first has not handed on.
				return nil, nil
			}
			source, err := takeByteStream(body, kind)
			if err != nil {
				return err, nil
			}
			if source != nil {
				st.framer = newSSEFramer(newBalStreamReader(source))
			} else {
				st.framer = newSSEFramer(st.reader)
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externReadSseEvent",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			body, _ := bodyOf(args[0].(*values.Object))
			st := body.state()
			if st.framer == nil {
				return nil, nil
			}
			event, err := st.framer.next()
			if err != nil {
				st.framer = nil
				_ = st.closeReader()
				// An error from a stream set with setByteStream is returned as it is.
				var elemErr *streamElementError
				if errors.As(err, &elemErr) {
					return elemErr.value, nil
				}
				return mimeError("GenericError", "Error occurred while reading stream: "+err.Error()), nil
			}
			if event == nil {
				st.framer = nil
				_ = st.closeReader()
				return nil, nil
			}
			return byteArrayValue(ctx, types, event), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseSseEvent",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			parseSSEEvent(ctx.TypeCtx(), args[0].(string), args[1].(*values.Map))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetSseEventBytes",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return byteArrayValue(ctx, types, sseEventBytes(args[0].(*values.Map))), nil
		})
}

// createSSEResponse turns a resource function's stream return value into an http:Response
// through the .bal createSseResponse helper.
func createSSEResponse(ctx *extern.Context, result values.BalValue) (*values.Object, *values.Error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, "createSseResponse")
	if !ok {
		return nil, values.NewErrorWithMessage("http: internal helper function createSseResponse not found")
	}
	resp, err := ctx.InvokeFunction(handle, []values.BalValue{result})
	if err != nil {
		return nil, values.NewErrorWithMessage(err.Error())
	}
	switch v := resp.(type) {
	case *values.Object:
		return v, nil
	case *values.Error:
		return nil, v
	}
	return nil, values.NewErrorWithMessage("http: createSseResponse returned an unexpected value")
}

// bindEventStream binds a response to a target that is neither http:Response nor anydata,
// following jBallerina's processResponse.
func bindEventStream(ctx *extern.Context, resp *values.Object, targetArg values.BalValue) values.BalValue {
	result := openEventStream(ctx, resp, targetArg)
	if _, failed := result.(*values.Error); failed {
		closeUnreadBody(resp)
	}
	return result
}

// closeUnreadBody closes the body of a response that is not handed to the caller, without
// reading it, so a peer that keeps sending cannot hold the call.
func closeUnreadBody(resp *values.Object) {
	body, _ := bodyOf(resp)
	if body == nil {
		return
	}
	st := body.state()
	st.framer = nil
	_ = st.closeReader()
	if r, _ := body.takeWire(); r != nil {
		_ = r.Close()
	}
}

func openEventStream(ctx *extern.Context, resp *values.Object, targetArg values.BalValue) values.BalValue {
	isTarget, ok := ctx.LookupFunction(orgName, moduleName, "isEventStreamTarget")
	if !ok {
		return values.NewErrorWithMessage("http: internal helper function isEventStreamTarget not found")
	}
	matches, err := ctx.InvokeFunction(isTarget, []values.BalValue{targetArg})
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	if matches != true {
		return values.NewError(semtypes.Error, "payload binding failed: "+
			"Target return type must not be a union of stream<http:SseEvent, error?> and anydata",
			nil, "PayloadBindingClientError", nil)
	}
	getStream, ok := ctx.LookupFunction(orgName, moduleName, "getSseEventStream")
	if !ok {
		return values.NewErrorWithMessage("http: internal helper function getSseEventStream not found")
	}
	eventStream, err := ctx.InvokeFunction(getStream, []values.BalValue{resp})
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	// A narrower element type would hold events, such as comment-only ones, outside it.
	target := targetArg.(*values.TypeDesc).Constraint()
	if s, ok := eventStream.(*values.Stream); ok && !semtypes.IsSubtype(ctx.TypeCtx(), s.Type, target) {
		return values.NewError(semtypes.Error, "payload binding failed: "+
			"Target return type must accept stream<http:SseEvent, error?>", nil, "PayloadBindingClientError", nil)
	}
	return eventStream
}
