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
	"compress/zlib"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type compressionMode uint8

const (
	compressAuto compressionMode = iota
	compressAlways
	compressNever
)

type chunkingMode uint8

const (
	chunkAuto chunkingMode = iota
	chunkAlways
	chunkNever
)

// chunkingThreshold is the largest body CHUNKING_AUTO sends with a Content-Length.
const chunkingThreshold = 8192

// responseEncoder applies a service's subtype prefix, content coding and HTTP/1.1 framing,
// holding back up to chunkingThreshold body bytes until it can choose the framing.
type responseEncoder struct {
	http.ResponseWriter
	r      *http.Request
	config serviceConfig

	status      int
	coding      string
	wroteHeader bool
	committed   bool
	aborted     bool
	held        bytes.Buffer
	body        io.Writer
	compressor  io.WriteCloser
}

func newResponseEncoder(w http.ResponseWriter, r *http.Request) *responseEncoder {
	return &responseEncoder{ResponseWriter: w, r: r, config: defaultServiceConfig()}
}

// useService applies the configuration of the service whose resource answers the request.
func (e *responseEncoder) useService(config serviceConfig) {
	e.config = config
}

// started reports whether a final status has been written.
func (e *responseEncoder) started() bool {
	return e.wroteHeader
}

func (e *responseEncoder) WriteHeader(statusCode int) {
	if e.wroteHeader {
		return
	}
	if statusCode < http.StatusOK && statusCode != http.StatusSwitchingProtocols {
		e.ResponseWriter.WriteHeader(statusCode)
		return
	}
	e.wroteHeader = true
	e.status = statusCode
	header := e.Header()
	if statusCode != http.StatusNoContent && negotiatesCoding(e.config.compression, header) {
		addVary(header, "Accept-Encoding")
	}
	// As in jBallerina, compression is decided before the subtype prefix is added.
	if bodyAllowed(e.r, statusCode) {
		e.coding = contentCoding(e.config.compression, header, e.r)
	}
	prefix := e.config.mediaTypeSubtypePrefix
	mediaType, subtype, ok := strings.Cut(header.Get("Content-Type"), "/")
	if prefix != "" && ok && mediaType != "" && !strings.EqualFold(strings.TrimSpace(mediaType), "multipart") &&
		!strings.HasPrefix(strings.ToLower(subtype), strings.ToLower(prefix)+"+") {
		header.Set("Content-Type", mediaType+"/"+prefix+"+"+subtype)
	}
	if !bodyAllowed(e.r, statusCode) {
		_ = e.commit(false)
	}
}

func (e *responseEncoder) Write(b []byte) (int, error) {
	if !e.wroteHeader {
		e.WriteHeader(http.StatusOK)
	}
	if e.committed {
		return e.body.Write(b)
	}
	// The body is held whole only when its length must be measured for a Content-Length.
	measured := e.r.ProtoMajor == 1 && (e.r.ProtoMinor == 0 || e.config.chunking == chunkNever) &&
		(e.coding != "" || e.Header().Get("Content-Length") == "")
	if e.held.Len()+len(b) > chunkingThreshold && !measured {
		if err := e.commit(true); err != nil {
			return 0, err
		}
		return e.body.Write(b)
	}
	e.held.Write(b)
	return len(b), nil
}

// abort marks the body as failed: finish then sends nothing more, not even a held response
// whose measured length would pass the truncated body off as complete.
func (e *responseEncoder) abort() {
	e.aborted = true
}

// finish sends the response held back so far and ends its content coding.
func (e *responseEncoder) finish() {
	if !e.wroteHeader || e.aborted {
		return
	}
	if !e.committed {
		_ = e.commit(false)
	}
	if e.compressor != nil {
		_ = e.compressor.Close()
	}
}

// commit sends the header and the body held so far; streaming means more body follows.
func (e *responseEncoder) commit(streaming bool) error {
	e.committed = true
	header := e.Header()
	hasBody := bodyAllowed(e.r, e.status)
	coding := e.coding
	if !streaming && e.held.Len() == 0 {
		coding = ""
	}
	if strings.EqualFold(strings.TrimSpace(header.Get("Content-Encoding")), "identity") {
		header.Del("Content-Encoding")
	}
	held := e.held.Bytes()
	if coding != "" {
		header.Set("Content-Encoding", coding)
		header.Del("Content-Length")
		if !streaming {
			var encoded bytes.Buffer
			compressor := newCompressor(coding, &encoded)
			_, _ = compressor.Write(held)
			_ = compressor.Close()
			held = encoded.Bytes()
		}
	}
	chunked := hasBody && e.r.ProtoMajor == 1 && e.r.ProtoMinor >= 1 &&
		(e.config.chunking == chunkAlways || (e.config.chunking == chunkAuto && streaming))
	switch {
	case chunked:
		header.Del("Content-Length")
		header.Set("Transfer-Encoding", "chunked")
	case hasBody && !streaming:
		header.Set("Content-Length", strconv.Itoa(len(held)))
	}
	e.ResponseWriter.WriteHeader(e.status)
	e.body = e.ResponseWriter
	if coding != "" && streaming {
		e.compressor = newCompressor(coding, e.ResponseWriter)
		e.body = e.compressor
	}
	if len(held) == 0 || !hasBody {
		return nil
	}
	_, err := e.body.Write(held)
	return err
}

// bodyAllowed reports whether a response with statusCode to r carries a body, as Netty's
// HttpContentEncoder decides which responses pass through unencoded.
func bodyAllowed(r *http.Request, statusCode int) bool {
	return r.Method != http.MethodHead && statusCode != http.StatusNoContent &&
		statusCode != http.StatusNotModified && statusCode >= http.StatusOK
}

// contentCoding returns the coding a response gets, or "": a Content-Encoding the resource set
// selects it, otherwise the compression config and the request's Accept-Encoding decide.
func contentCoding(config compressionConfig, header http.Header, r *http.Request) string {
	if set := strings.ToLower(header.Get("Content-Encoding")); set != "" {
		switch {
		case strings.Contains(set, "gzip"):
			return "gzip"
		case strings.Contains(set, "deflate"):
			return "deflate"
		}
		return ""
	}
	if !negotiatesCoding(config, header) {
		return ""
	}
	values := r.Header.Values("Accept-Encoding")
	acceptEncoding := strings.ToLower(strings.Join(values, ","))
	onlyIdentity, _, _ := strings.Cut(acceptEncoding, ";")
	if config.enable == compressAlways && (len(values) == 0 || strings.TrimSpace(onlyIdentity) == "identity" &&
		!strings.Contains(acceptEncoding, ",")) {
		return "gzip"
	}
	return negotiateCoding(acceptEncoding)
}

// negotiatesCoding reports whether the request's Accept-Encoding chooses the response's coding.
func negotiatesCoding(config compressionConfig, header http.Header) bool {
	return header.Get("Content-Encoding") == "" && config.enable != compressNever &&
		config.matches(header.Get("Content-Type"))
}

// addVary adds field to the Vary header unless it is listed already.
func addVary(header http.Header, field string) {
	for _, value := range header.Values("Vary") {
		for _, listed := range strings.Split(value, ",") {
			if listed = strings.TrimSpace(listed); listed == "*" || strings.EqualFold(listed, field) {
				return
			}
		}
	}
	header.Add("Vary", field)
}

// negotiateCoding picks gzip or deflate from a lower-cased Accept-Encoding: higher q-value,
// gzip on a tie, and "*" for an unlisted one.
func negotiateCoding(acceptEncoding string) string {
	starQ, gzipQ, deflateQ := float32(-1), float32(-1), float32(-1)
	for _, coding := range strings.Split(acceptEncoding, ",") {
		q := float32(1)
		if _, value, ok := strings.Cut(coding, "="); ok {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
			q = float32(parsed)
			if err != nil {
				q = 0
			}
		}
		switch {
		case strings.Contains(coding, "*"):
			starQ = q
		case strings.Contains(coding, "gzip"):
			gzipQ = max(gzipQ, q)
		case strings.Contains(coding, "deflate"):
			deflateQ = max(deflateQ, q)
		}
	}
	switch {
	case gzipQ > 0 || deflateQ > 0:
		if gzipQ >= deflateQ {
			return "gzip"
		}
		return "deflate"
	case starQ > 0 && gzipQ == -1:
		return "gzip"
	case starQ > 0 && deflateQ == -1:
		return "deflate"
	}
	return ""
}

// The deflate coding is zlib-wrapped, as Netty's ZlibWrapper.ZLIB writes it.
func newCompressor(coding string, w io.Writer) io.WriteCloser {
	if coding == "gzip" {
		return gzip.NewWriter(w)
	}
	return zlib.NewWriter(w)
}
