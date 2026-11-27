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

package extern_test

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestHttpServicePush drives HTTP/2 server push with a raw HTTP/2 client, since net/http's client
// never accepts pushed streams.
func TestHttpServicePush(t *testing.T) {
	skipIfNoLoopback(t)
	t.Parallel()
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "http-svc-push-v", FuncName: "h2Get",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return h2Get(args[0].(int64), args[1].(string), args[2].(bool))
			}},
	}
	runExtern(t, fileCase("http-service/http-svc-push-v"), newHTTPPal(palnative.NewHTTPClient), externs)
}

// h2stream is what a client received on one stream.
type h2stream struct {
	header http.Header
	body   bytes.Buffer
	ended  bool
	reset  bool
}

// h2Get sends a GET for path over cleartext HTTP/2 and returns, line by line, the promises the
// server made and what each stream carried, the promised streams in the order they were promised.
func h2Get(port int64, path string, enablePush bool) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte(http2.ClientPreface)); err != nil {
		return "", err
	}
	framer := http2.NewFramer(conn, conn)
	decoder := hpack.NewDecoder(4096, nil)
	framer.ReadMetaHeaders = decoder
	push := uint32(0)
	if enablePush {
		push = 1
	}
	if err := framer.WriteSettings(http2.Setting{ID: http2.SettingEnablePush, Val: push}); err != nil {
		return "", err
	}
	var block bytes.Buffer
	encoder := hpack.NewEncoder(&block)
	for _, f := range []hpack.HeaderField{{Name: ":method", Value: "GET"}, {Name: ":scheme", Value: "http"},
		{Name: ":authority", Value: fmt.Sprintf("127.0.0.1:%d", port)}, {Name: ":path", Value: path}} {
		if err := encoder.WriteField(f); err != nil {
			return "", err
		}
	}
	if err := framer.WriteHeaders(http2.HeadersFrameParam{StreamID: 1, BlockFragment: block.Bytes(),
		EndStream: true, EndHeaders: true}); err != nil {
		return "", err
	}

	streams := map[uint32]*h2stream{1: {}}
	var promised []uint32
	var lines []string
	done := func() bool {
		for _, s := range streams {
			if !s.ended && !s.reset {
				return false
			}
		}
		return true
	}
	for !done() {
		frame, err := framer.ReadFrame()
		if err != nil {
			return "", err
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				if err := framer.WriteSettingsAck(); err != nil {
					return "", err
				}
			}
		case *http2.PushPromiseFrame:
			fields, err := decoder.DecodeFull(f.HeaderBlockFragment())
			if err != nil {
				return "", err
			}
			streams[f.PromiseID] = &h2stream{}
			promised = append(promised, f.PromiseID)
			lines = append(lines, "promise "+formatFields(fields))
		case *http2.MetaHeadersFrame:
			s := streams[f.StreamID]
			s.header = http.Header{}
			for _, field := range f.Fields {
				s.header.Add(field.Name, field.Value)
			}
			s.ended = s.ended || f.StreamEnded()
		case *http2.DataFrame:
			s := streams[f.StreamID]
			s.body.Write(f.Data())
			s.ended = s.ended || f.StreamEnded()
		case *http2.RSTStreamFrame:
			streams[f.StreamID].reset = true
		case *http2.GoAwayFrame:
			return "", errors.New("connection closed: " + f.ErrCode.String())
		}
	}
	lines = append(lines, "response "+streams[1].String())
	for _, id := range promised {
		lines = append(lines, "pushed "+streams[id].String())
	}
	return strings.Join(lines, "\n"), nil
}

func (s *h2stream) String() string {
	if s.reset {
		return "reset"
	}
	var fields []string
	for _, name := range []string{":status", "content-type", "x-pushed"} {
		if v := s.header.Get(name); v != "" {
			fields = append(fields, name+"="+v)
		}
	}
	return strings.Join(fields, " ") + " body=" + s.body.String()
}

// formatFields renders the pseudo-headers of a promise in a fixed order, then its other headers.
func formatFields(fields []hpack.HeaderField) string {
	var pseudo, rest []string
	for _, f := range fields {
		if strings.HasPrefix(f.Name, ":") {
			pseudo = append(pseudo, f.Name+"="+f.Value)
		} else {
			rest = append(rest, f.Name+"="+f.Value)
		}
	}
	slices.Sort(pseudo)
	slices.Sort(rest)
	return strings.Join(append(pseudo, rest...), " ")
}
