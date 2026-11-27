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
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestHttpServiceListenerConfig checks the connection settings of a listener on the wire.
func TestHttpServiceListenerConfig(t *testing.T) {
	skipIfNoLoopback(t)
	t.Parallel()
	module := "http-svc-listener-config-v"
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: module, FuncName: "h1Connection",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return h1Connection(args[0].(int64), args[1].(string))
			}},
		{Org: "$anon", Module: module, FuncName: "h2Settings",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return h2Settings(args[0].(int64), args[1].(bool))
			}},
	}
	runExtern(t, fileCase("http-service/"+module), newHTTPPal(palnative.NewHTTPClient).withRealFS(), externs)
}

// h1Connection sends two GET requests with the given HTTP version on one connection, asking for
// keep-alive over HTTP/1.0, and reports the Connection header of each response, or that the server
// closed the connection.
func h1Connection(port int64, version string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", err
	}
	reader := bufio.NewReader(conn)
	var results []string
	for range 2 {
		keepAlive := ""
		if version == "HTTP/1.0" {
			keepAlive = "Connection: keep-alive\r\n"
		}
		if _, err := fmt.Fprintf(conn, "GET /hello %s\r\nHost: localhost\r\n%s\r\n", version, keepAlive); err != nil {
			results = append(results, "closed")
			break
		}
		resp, err := http.ReadResponse(reader, nil)
		if err != nil {
			results = append(results, "closed")
			break
		}
		_ = resp.Body.Close()
		// ReadResponse drops a "close" token from the Connection header and sets Close instead.
		connection := resp.Header.Get("Connection")
		if resp.Close {
			connection = "close"
		}
		results = append(results, "connection="+connection)
	}
	return strings.Join(results, " "), nil
}

// h2Settings opens an HTTP/2 connection, over TLS when secure, and returns the stream limit and
// initial window size of the server's SETTINGS frame.
func h2Settings(port int64, secure bool) (string, error) {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	var conn net.Conn
	var err error
	if secure {
		conn, err = tls.Dial("tcp", address, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2"}}) //nolint:gosec // test server certificate
	} else {
		conn, err = net.Dial("tcp", address)
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte(http2.ClientPreface)); err != nil {
		return "", err
	}
	framer := http2.NewFramer(conn, conn)
	if err := framer.WriteSettings(); err != nil {
		return "", err
	}
	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return "", err
		}
		settings, ok := frame.(*http2.SettingsFrame)
		if !ok || settings.IsAck() {
			continue
		}
		streams, _ := settings.Value(http2.SettingMaxConcurrentStreams)
		window, _ := settings.Value(http2.SettingInitialWindowSize)
		return fmt.Sprintf("maxConcurrentStreams=%d initialWindowSize=%d", streams, window), nil
	}
}
