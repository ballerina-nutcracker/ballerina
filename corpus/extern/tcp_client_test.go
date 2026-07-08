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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testharness"
)

// tcpPal wraps the default in-memory TestPal but overrides Net.Dial/Listen
// with the real platform implementations, so tcp:Client/tcp:Listener perform
// actual TCP(+TLS) I/O.
type tcpPal struct {
	testharness.TestPal
}

func newTcpPal() *tcpPal {
	return &tcpPal{TestPal: testharness.NewTestPal()}
}

func (p *tcpPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{Dial: palnative.Dial, Listen: palnative.ListenTCP}
	return base
}

// TestTcpClientEcho exercises tcp:Client (init/writeBytes/readBytes/close)
// against a bare Go TCP echo server — no tcp:Listener involved, isolating
// the client half of the port.
func TestTcpClientEcho(t *testing.T) {
	skipIfNoLoopback(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return
			}
		}
	}()

	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("splitting listener addr: %v", err)
	}

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/tcp;

public function main() returns error? {
    tcp:Client c = check new (%q, %s, {});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello

    check c->writeBytes("world".toBytes());
    readonly & byte[] echoed2 = check c->readBytes();
    io:println('string:fromBytes(echoed2)); // @output world

    check c->close();
    tcp:Error? afterClose = c->writeBytes("x".toBytes());
    io:println(afterClose is tcp:Error); // @output true
    return;
}
`, host, port)

	tmpDir := t.TempDir()
	tmpBalFile := filepath.Join(tmpDir, "tcp-client-echo-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "tcp-client-echo-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "tcp-client-echo-v.txtar"),
	}
	runExtern(t, tc, newTcpPal(), nil)
}

// TestTcpClientTLS exercises the TLS path on both tcp:Listener (CertKey)
// and tcp:Client (PEM cert path) — the listener and client both run inside
// the same Ballerina process, mirroring TestLdapClientTLS's use of
// generateTestCerts.
func TestTcpClientTLS(t *testing.T) {
	skipIfNoLoopback(t)
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)

	tmpDir := t.TempDir()
	caCertFile := filepath.Join(tmpDir, "ca.pem")
	serverCertFile := filepath.Join(tmpDir, "server.pem")
	serverKeyFile := filepath.Join(tmpDir, "server-key.pem")
	for _, pair := range []struct {
		path string
		data []byte
	}{
		{caCertFile, caCertPEM},
		{serverCertFile, serverCertPEM},
		{serverKeyFile, serverKeyPEM},
	} {
		if err := os.WriteFile(pair.path, pair.data, 0o600); err != nil {
			t.Fatalf("writing %s: %v", pair.path, err)
		}
	}

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/tcp;

service class EchoService {
    *tcp:ConnectionService;
    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns tcp:Error? {
        check caller->writeBytes(data);
    }
}

service class EchoServer {
    *tcp:Service;
    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new EchoService();
    }
}

listener tcp:Listener tlsListener = new (19393, {
    secureSocket: {key: {certFile: %q, keyFile: %q}}
});

function init() returns error? {
    check tlsListener.attach(new EchoServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19393, {
        secureSocket: {cert: %q}
    });
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello
    check c->close();
}
`, serverCertFile, serverKeyFile, caCertFile)

	tmpBalFile := filepath.Join(tmpDir, "tcp-client-tls-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "tcp-client-tls-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "tcp-client-tls-v.txtar"),
	}
	runExtern(t, tc, newTcpPal(), nil)
}
