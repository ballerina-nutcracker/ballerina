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

package native

import (
	"bufio"
	"cmp"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// The encoder decides the wire framing and content coding, which a Ballerina client hides, so
// these cases read the raw responses of a real server.
func TestResponseEncoder(t *testing.T) {
	t.Parallel()
	small, big := "hi", strings.Repeat("0123456789", 1000)
	cases := []struct {
		name           string
		config         func(*serviceConfig)
		acceptEncoding string
		method         string
		status         int
		contentType    string
		setEncoding    string
		body           string
		wantEncoding   string
		wantChunked    bool
	}{
		{name: "auto without accept-encoding", body: small},
		{name: "auto gzip", acceptEncoding: "gzip", body: small, wantEncoding: "gzip"},
		{name: "auto deflate", acceptEncoding: "deflate", body: big, wantEncoding: "deflate", wantChunked: true},
		{name: "auto unsupported", acceptEncoding: "br", body: small},
		{name: "auto identity", acceptEncoding: "identity", body: small},
		{name: "auto q-values", acceptEncoding: "deflate;q=0.5, gzip;q=0.4", body: small, wantEncoding: "deflate"},
		{name: "auto excluded gzip", acceptEncoding: "gzip;q=0, deflate", body: small, wantEncoding: "deflate"},
		// jBallerina answers * with snappy (ballerina-library#9879).
		{name: "auto star", acceptEncoding: "*", body: small, wantEncoding: "gzip"},
		// jBallerina matches codings case-sensitively (ballerina-library#9880).
		{name: "auto upper case", acceptEncoding: "GZIP", body: small, wantEncoding: "gzip"},
		{name: "auto big", body: big, wantChunked: true},
		{name: "always without accept-encoding", config: compress(compressAlways), body: small, wantEncoding: "gzip"},
		{name: "always identity", config: compress(compressAlways), acceptEncoding: "identity", body: small, wantEncoding: "gzip"},
		// jBallerina does not read Identity as identity under COMPRESSION_ALWAYS (ballerina-library#9880).
		{name: "always identity case", config: compress(compressAlways), acceptEncoding: "Identity;q=1", body: small,
			wantEncoding: "gzip"},
		{name: "always unsupported", config: compress(compressAlways), acceptEncoding: "br", body: small},
		{name: "never", config: compress(compressNever), acceptEncoding: "gzip", body: big, wantChunked: true},
		{name: "content type matched", config: compressTypes("application/JSON"), acceptEncoding: "gzip",
			contentType: "application/json", body: small, wantEncoding: "gzip"},
		{name: "content type unmatched", config: compressTypes("application/json"), acceptEncoding: "gzip",
			contentType: "text/plain", body: small},
		{name: "content type before prefix", config: withPrefix(compressTypes("application/json"), "vnd.x"),
			acceptEncoding: "gzip", contentType: "application/json", body: small, wantEncoding: "gzip"},
		{name: "resource gzip", setEncoding: "gzip", body: small, wantEncoding: "gzip"},
		{name: "resource identity", setEncoding: "identity", acceptEncoding: "gzip", body: small},
		// jBallerina drops a Content-Encoding it cannot apply (ballerina-library#633).
		{name: "resource unsupported", setEncoding: "br", acceptEncoding: "gzip", body: small, wantEncoding: "br"},
		{name: "empty body", acceptEncoding: "gzip"},
		{name: "no content", acceptEncoding: "gzip", status: http.StatusNoContent},
		{name: "not modified", acceptEncoding: "gzip", status: http.StatusNotModified},
		{name: "head", acceptEncoding: "gzip", method: http.MethodHead, body: small},
		{name: "chunking always", config: chunk(chunkAlways), body: small, wantChunked: true},
		{name: "chunking never", config: chunk(chunkNever), acceptEncoding: "gzip", body: big, wantEncoding: "gzip"},
		{name: "chunking threshold", body: big[:chunkingThreshold]},
		{name: "chunking over threshold", body: big[:chunkingThreshold+1], wantChunked: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			config := defaultServiceConfig()
			if c.config != nil {
				c.config(&config)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				encoder := newResponseEncoder(w, r)
				defer encoder.finish()
				encoder.useService(config)
				if c.contentType != "" {
					encoder.Header().Set("Content-Type", c.contentType)
				}
				if c.setEncoding != "" {
					encoder.Header().Set("Content-Encoding", c.setEncoding)
				}
				encoder.WriteHeader(cmp.Or(c.status, http.StatusOK))
				for rest := c.body; rest != ""; {
					n := min(len(rest), 1000)
					_, _ = io.WriteString(encoder, rest[:n])
					rest = rest[n:]
				}
			}))
			defer server.Close()
			method := cmp.Or(c.method, http.MethodGet)
			request, _ := http.NewRequest(method, server.URL, nil)
			if c.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", c.acceptEncoding)
			}
			response, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = response.Body.Close() }()
			if got := response.Header.Get("Content-Encoding"); got != c.wantEncoding {
				t.Errorf("Content-Encoding = %q, want %q", got, c.wantEncoding)
			}
			chunked := len(response.TransferEncoding) > 0
			if chunked != c.wantChunked {
				t.Errorf("chunked = %v (Content-Length %d), want %v", chunked, response.ContentLength, c.wantChunked)
			}
			want := ""
			if c.body != "" && method != http.MethodHead && c.status == 0 {
				want = c.body
			}
			if got := decode(t, response); got != want {
				t.Errorf("body = %d bytes, want %d", len(got), len(want))
			}
		})
	}
}

func TestResponseEncoderHTTP10(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := newResponseEncoder(w, r)
		defer encoder.finish()
		_, _ = io.WriteString(encoder, strings.Repeat("a", 9000))
	}))
	defer server.Close()
	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := fmt.Fprint(conn, "GET / HTTP/1.0\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.ContentLength != 9000 {
		t.Errorf("Content-Length = %d, want 9000", response.ContentLength)
	}
}

// Over HTTP/2 the coding follows the HTTP/1.1 rules; jBallerina sends content-encoding: identity for an
// unencoded response and encodes 204, 304, HEAD and empty responses (ballerina-library#9878).
func TestResponseEncoderHTTP2(t *testing.T) {
	t.Parallel()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := newResponseEncoder(w, r)
		defer encoder.finish()
		status, _ := strconv.Atoi(r.URL.Query().Get("status"))
		encoder.WriteHeader(cmp.Or(status, http.StatusOK))
		_, _ = io.WriteString(encoder, r.URL.Query().Get("body"))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	client := server.Client()
	client.Transport.(*http.Transport).DisableCompression = true
	cases := []struct {
		name, method, query, acceptEncoding, wantEncoding string
	}{
		{"gzip", http.MethodGet, "body=hi", "gzip", "gzip"},
		{"identity", http.MethodGet, "body=hi", "", ""},
		{"empty body", http.MethodGet, "", "gzip", ""},
		{"no content", http.MethodGet, "status=204", "gzip", ""},
		{"not modified", http.MethodGet, "status=304", "gzip", ""},
		{"head", http.MethodHead, "body=hi", "gzip", ""},
	}
	for _, c := range cases {
		request, _ := http.NewRequest(c.method, server.URL+"/?"+c.query, nil)
		if c.acceptEncoding != "" {
			request.Header.Set("Accept-Encoding", c.acceptEncoding)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.ProtoMajor != 2 {
			t.Fatalf("%s: protocol %s, want HTTP/2", c.name, response.Proto)
		}
		if got := response.Header.Get("Content-Encoding"); got != c.wantEncoding {
			t.Errorf("%s: Content-Encoding = %q, want %q", c.name, got, c.wantEncoding)
		}
	}
}

// A body whose length is already known streams under CHUNKING_NEVER instead of being held whole.
func TestResponseEncoderStreamsKnownLength(t *testing.T) {
	t.Parallel()
	recorder := &countingRecorder{ResponseRecorder: httptest.NewRecorder()}
	encoder := newResponseEncoder(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	config := defaultServiceConfig()
	config.chunking = chunkNever
	encoder.useService(config)
	body := strings.Repeat("a", chunkingThreshold+1)
	encoder.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = io.WriteString(encoder, body)
	if recorder.writeHeaderCalls != 1 || recorder.Body.Len() != len(body) {
		t.Errorf("before finish: %d WriteHeader calls, %d body bytes; want the body streamed", recorder.writeHeaderCalls,
			recorder.Body.Len())
	}
	encoder.finish()
}

// A large write past the threshold goes straight through rather than being copied into the hold.
func TestResponseEncoderLargeWriteNotHeld(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	encoder := newResponseEncoder(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	body := strings.Repeat("a", 4*chunkingThreshold)
	_, _ = io.WriteString(encoder, body)
	if encoder.held.Cap() > chunkingThreshold || recorder.Body.Len() != len(body) {
		t.Errorf("held capacity %d, %d body bytes written; want the write passed through", encoder.held.Cap(),
			recorder.Body.Len())
	}
	encoder.finish()
}

// http-svc-service-config-v covers the prefix rules; the module's client does not see a 101 as sent, and the
// status count is not visible to a client, so the encoder is checked directly for both.
func TestResponseEncoderSubtypePrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		contentType string
		status      int
		want        string
	}{
		{"application/json", http.StatusOK, "application/vnd.x+json"},
		{"text/plain", http.StatusSwitchingProtocols, "text/vnd.x+plain"},
	}
	for _, c := range cases {
		recorder := &countingRecorder{ResponseRecorder: httptest.NewRecorder()}
		encoder := newResponseEncoder(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		config := defaultServiceConfig()
		config.mediaTypeSubtypePrefix = "vnd.x"
		encoder.useService(config)
		encoder.Header().Set("Content-Type", c.contentType)
		encoder.WriteHeader(c.status)
		_, _ = encoder.Write([]byte("x"))
		encoder.finish()
		if got := recorder.Header().Get("Content-Type"); got != c.want {
			t.Errorf("Content-Type for %q = %q, want %q", c.contentType, got, c.want)
		}
		if recorder.Code != c.status || recorder.writeHeaderCalls != 1 {
			t.Errorf("status for %q = %d after %d WriteHeader calls, want %d once", c.contentType, recorder.Code,
				recorder.writeHeaderCalls, c.status)
		}
	}
}

// A ResponseWriter sends an implicit 200 on the first Write, so the prefix must be added then too. Every
// module path sets the status before the body, so a corpus test cannot reach this.
func TestResponseEncoderImplicitHeader(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	encoder := newResponseEncoder(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	config := defaultServiceConfig()
	config.mediaTypeSubtypePrefix = "vnd.x"
	encoder.useService(config)
	encoder.Header().Set("Content-Type", "application/json")
	_, _ = encoder.Write([]byte("{}"))
	encoder.finish()
	if got := recorder.Header().Get("Content-Type"); got != "application/vnd.x+json" {
		t.Errorf("Content-Type = %q", got)
	}
}

// countingRecorder counts WriteHeader calls, which httptest.ResponseRecorder drops after the first.
type countingRecorder struct {
	*httptest.ResponseRecorder
	writeHeaderCalls int
}

func (r *countingRecorder) WriteHeader(statusCode int) {
	r.writeHeaderCalls++
	r.ResponseRecorder.WriteHeader(statusCode)
}

func compress(enable compressionMode) func(*serviceConfig) {
	return func(c *serviceConfig) { c.compression.enable = enable }
}

// compressTypes builds the config through readCompressionConfig, which trims and lower-cases.
func compressTypes(contentTypes ...string) func(*serviceConfig) {
	return func(c *serviceConfig) {
		tc := semtypes.ContextFrom(semtypes.CreateTypeEnv())
		items := make([]values.BalValue, len(contentTypes))
		for i, t := range contentTypes {
			items[i] = " " + strings.ToUpper(t) + " "
		}
		compression := newMappingValue()
		compression.Put(tc, "contentTypes", newListValue(items))
		c.compression = readCompressionConfig(compression)
	}
}

func withPrefix(configure func(*serviceConfig), prefix string) func(*serviceConfig) {
	return func(c *serviceConfig) {
		configure(c)
		c.mediaTypeSubtypePrefix = prefix
	}
}

func chunk(chunking chunkingMode) func(*serviceConfig) {
	return func(c *serviceConfig) { c.chunking = chunking }
}

func decode(t *testing.T, response *http.Response) string {
	t.Helper()
	var reader io.Reader = response.Body
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		gz, err := gzip.NewReader(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		reader = gz
	case "deflate":
		z, err := zlib.NewReader(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		reader = z
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A body whose copy fails while it is held to be measured is not sent with the measured length.
func TestResponseEncoderAbortsFailedBody(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	encoder := newResponseEncoder(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	config := defaultServiceConfig()
	config.chunking = chunkNever
	encoder.useService(config)
	stream := io.NopCloser(io.MultiReader(strings.NewReader("partial"), failingReadCloser{}))
	resp := values.NewObject(semtypes.Object, nil, nil, nil, nil)
	writeResponseObject(encoder, resp, http.StatusOK, &responseBodyHolder{stream: stream}, "")
	encoder.finish()
	if !encoder.aborted || recorder.Body.Len() != 0 || recorder.Header().Get("Content-Length") != "" {
		t.Errorf("aborted %v, body %q, Content-Length %q; want nothing sent", encoder.aborted, recorder.Body.String(),
			recorder.Header().Get("Content-Length"))
	}
}
