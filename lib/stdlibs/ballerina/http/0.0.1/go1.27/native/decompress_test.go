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
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// A zlib header with FDICT set needs a preset dictionary, which zlib.NewReader rejects.
func TestDeflateReadCloserKeepsInitError(t *testing.T) {
	t.Parallel()
	reader := &deflateReadCloser{underlying: io.NopCloser(bytes.NewReader([]byte{0x78, 0xbb, 0, 0, 0, 0}))}
	buf := make([]byte, 8)
	_, first := reader.Read(buf)
	_, second := reader.Read(buf)
	if first == nil || second != first {
		t.Errorf("Read errors = %v, %v; want the same init error twice", first, second)
	}
}

// The encoded Content-Length decides eager buffering although decoding removes it.
func TestBuildResponseBuffersSmallCompressedBody(t *testing.T) {
	t.Parallel()
	var body bytes.Buffer
	gz := gzip.NewWriter(&body)
	_, _ = gz.Write([]byte("hello"))
	_ = gz.Close()
	closed := false
	stream := &closeRecorder{Reader: bytes.NewReader(body.Bytes()), closed: &closed}
	headers := map[string][]string{"Content-Encoding": {"gzip"}, "Content-Length": {"23"}}
	response := buildResponse(semtypes.ContextFrom(semtypes.CreateTypeEnv()), semtypes.Object, 200, headers, stream)
	holderValue, _ := response.Get("body")
	holder := holderValue.(*responseBodyHolder)
	if string(holder.buf) != "hello" || holder.stream != nil || !closed {
		t.Errorf("holder = %q (stream %v, closed %v); want the decoded body buffered", holder.buf, holder.stream != nil, closed)
	}
	if len(headers) != 0 {
		t.Errorf("headers = %v; want Content-Encoding and Content-Length removed", headers)
	}
}

type closeRecorder struct {
	io.Reader
	closed *bool
}

func (c *closeRecorder) Close() error {
	*c.closed = true
	return nil
}
