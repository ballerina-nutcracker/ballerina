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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// coalescingConn holds writes made after startBuffering and sends them in a
// single write on Close, so the last TLS data record and the close_notify
// alert arrive together and the peer's tls.Conn.Read returns (n, io.EOF).
// That only happens on TLS 1.2, where the alert's record type is not
// encrypted, so the Go side of these tests pins TLS 1.2.
type coalescingConn struct {
	net.Conn
	mu        sync.Mutex
	buffering bool
	pending   bytes.Buffer
}

func (c *coalescingConn) startBuffering() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffering = true
}

func (c *coalescingConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.buffering {
		return c.pending.Write(p)
	}
	return c.Conn.Write(p)
}

func (c *coalescingConn) Close() error {
	c.mu.Lock()
	// tls.Conn.Close expires the write deadline before closing the transport.
	_ = c.Conn.SetWriteDeadline(time.Time{})
	_, _ = c.Conn.Write(c.pending.Bytes())
	c.mu.Unlock()
	return c.Conn.Close()
}

// sendFinalBytesAndClose writes "bye" and closes the TLS connection so the
// data and the close_notify alert reach the peer in one segment.
func sendFinalBytesAndClose(tlsConn *tls.Conn, raw *coalescingConn) error {
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	raw.startBuffering()
	if _, err := tlsConn.Write([]byte("bye")); err != nil {
		return err
	}
	return tlsConn.Close()
}

func writeTLSCertFiles(t *testing.T) (caCertPEM []byte, caFile, certFile, keyFile string) {
	t.Helper()
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)
	dir := t.TempDir()
	caFile = filepath.Join(dir, "ca.pem")
	certFile = filepath.Join(dir, "server.pem")
	keyFile = filepath.Join(dir, "server-key.pem")
	for path, data := range map[string][]byte{caFile: caCertPEM, certFile: serverCertPEM, keyFile: serverKeyPEM} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return caCertPEM, caFile, certFile, keyFile
}

func runInlineCase(t *testing.T, name, balContent string, externs []testharness.ExternRegistration) {
	t.Helper()
	balFile := filepath.Join(t.TempDir(), name+".bal")
	if err := os.WriteFile(balFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}
	tc := test_util.TestCase{
		Name:         name,
		InputPath:    balFile,
		ExpectedPath: filepath.Join(expectedDir, name+".txtar"),
	}
	runExtern(t, tc, newTcpPal(), externs)
}

// TestTcpClientTLSFinalBytes verifies that Client.readBytes returns the
// bytes a TLS read delivers together with the server's close_notify.
func TestTcpClientTLSFinalBytes(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	_, caFile, certFile, keyFile := writeTLSCertFiles(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("loading server cert: %v", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		raw := &coalescingConn{Conn: conn}
		_ = sendFinalBytesAndClose(tls.Server(raw, &tls.Config{Certificates: []tls.Certificate{cert}, MaxVersion: tls.VersionTLS12}), raw)
	}()
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("splitting listener addr: %v", err)
	}

	runInlineCase(t, "tcp-client-tls-final-bytes-v", fmt.Sprintf(`
import ballerina/io;
import ballerina/tcp;

public function main() returns error? {
    tcp:Client c = check new (%q, %s, timeout = 5, secureSocket = {cert: %q});
    readonly & byte[] last = check c->readBytes();
    io:println('string:fromBytes(last)); // @output bye
    (readonly & byte[])|tcp:Error next = c->readBytes();
    io:println(next is tcp:Error ? next.message() : "data"); // @output Connection closed by the server.
    check c->close();
}
`, host, port, caFile), nil)
}

// TestTcpServiceTLSFinalBytes verifies that a TLS listener dispatches the
// bytes a read delivers together with the client's close_notify to onBytes
// before onClose.
func TestTcpServiceTLSFinalBytes(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	caCertPEM, _, certFile, keyFile := writeTLSCertFiles(t)
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(caCertPEM)

	const module = "tcp-svc-tls-final-bytes-v"
	closed := make(chan struct{}, 1)
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: module, FuncName: "sendFinalBytes",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", args[0].(int64)))
				if err != nil {
					return values.NewErrorWithMessage("dial: " + err.Error()), nil
				}
				raw := &coalescingConn{Conn: conn}
				tlsConn := tls.Client(raw, &tls.Config{RootCAs: roots, ServerName: "127.0.0.1", MaxVersion: tls.VersionTLS12})
				if err := sendFinalBytesAndClose(tlsConn, raw); err != nil {
					return values.NewErrorWithMessage("send: " + err.Error()), nil
				}
				return nil, nil
			}},
		{Org: "$anon", Module: module, FuncName: "signalClosed",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				select {
				case closed <- struct{}{}:
				default:
				}
				return nil, nil
			}},
		{Org: "$anon", Module: module, FuncName: "waitForClose",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				select {
				case <-closed:
					return nil, nil
				case <-time.After(5 * time.Second):
					return values.NewErrorWithMessage("timed out waiting for onClose"), nil
				}
			}},
	}

	runInlineCase(t, module, fmt.Sprintf(`
import ballerina/io;
import ballerina/tcp;

service class PrintService {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) {
        io:println('string:fromBytes(data)); // @output bye
    }

    remote function onClose() {
        signalClosed();
    }
}

service class PrintServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new PrintService();
    }
}

listener tcp:Listener tlsListener = new (19412, localHost = "127.0.0.1",
    secureSocket = {key: <tcp:CertKey>{certFile: %q, keyFile: %q}}
);

function init() returns error? {
    check tlsListener.attach(new PrintServer());
}

isolated function sendFinalBytes(int port) returns error? = external;
isolated function signalClosed() = external;
isolated function waitForClose() returns error? = external;

public function testMain() returns error? {
    check sendFinalBytes(19412);
    error? closeResult = waitForClose();
    io:println(closeResult is ()); // @output true
}
`, certFile, keyFile), externs)
}

// TestTcpClientTLSHandshakeTimeout verifies that secureSocket.handshakeTimeout
// bounds the client handshake against a server that never answers the ClientHello.
func TestTcpClientTLSHandshakeTimeout(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	_, caFile, _, _ := writeTLSCertFiles(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		time.Sleep(10 * time.Second)
	}()
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("splitting listener addr: %v", err)
	}

	start := time.Now()
	runInlineCase(t, "tcp-client-tls-handshake-timeout-v", fmt.Sprintf(`
import ballerina/io;
import ballerina/tcp;

public function main() {
    tcp:Client|tcp:Error c = new (%q, %s, secureSocket = {cert: %q, handshakeTimeout: 0.5});
    io:println(c is tcp:Error); // @output true
}
`, host, port, caFile), nil)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("handshake took %v, want it bounded by handshakeTimeout", elapsed)
	}
}
