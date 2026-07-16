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
	"strconv"
	"testing"

	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testharness"
)

// udpPal wraps the default in-memory TestPal but overrides
// Net.DialPacket/ListenPacket with the real platform implementations, so
// udp:Client/udp:ConnectClient/udp:Listener perform actual UDP I/O.
type udpPal struct {
	testharness.TestPal
}

func newUdpPal() *udpPal {
	return &udpPal{TestPal: testharness.NewTestPal()}
}

func (p *udpPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{DialPacket: palnative.DialPacket, ListenPacket: palnative.ListenPacket}
	return base
}

// skipIfNoIPv6Loopback skips tests that need a working "::1" loopback —
// some CI/container environments disable IPv6 entirely at the kernel level.
func skipIfNoIPv6Loopback(t *testing.T) {
	t.Helper()
	pc, err := net.ListenPacket("udp6", "[::1]:0")
	if err != nil {
		t.Skip("skipping IPv6-loopback-dependent test: ::1 unavailable")
	}
	_ = pc.Close()
}

// goUDPEchoServer starts a bare Go UDP echo server on 127.0.0.1 (not a
// udp:Listener), isolating the client half of the port. Returns the bound
// port and a cleanup func.
func goUDPEchoServer(t *testing.T) int {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	go func() {
		buf := make([]byte, 65507)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if _, err := pc.WriteTo(buf[:n], addr); err != nil {
				return
			}
		}
	}()
	_, portStr, err := net.SplitHostPort(pc.LocalAddr().String())
	if err != nil {
		t.Fatalf("splitting listener addr: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parsing listener port: %v", err)
	}
	return port
}

// TestUdpConnectClientEcho exercises udp:ConnectClient (init/writeBytes/
// readBytes/close) against a bare Go UDP echo server.
func TestUdpConnectClientEcho(t *testing.T) {
	skipIfNoLoopback(t)
	port := goUDPEchoServer(t)

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/udp;

public function main() returns error? {
    udp:ConnectClient c = check new ("127.0.0.1", %d, {timeout: 5});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello

    check c->writeBytes("world".toBytes());
    readonly & byte[] echoed2 = check c->readBytes();
    io:println('string:fromBytes(echoed2)); // @output world

    check c->close();
    udp:Error? afterClose = c->writeBytes("x".toBytes());
    io:println(afterClose is udp:Error); // @output true
    return;
}
`, port)

	tmpDir := t.TempDir()
	tmpBalFile := filepath.Join(tmpDir, "udp-connect-client-echo-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "udp-connect-client-echo-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "udp-connect-client-echo-v.txtar"),
	}
	runExtern(t, tc, newUdpPal(), nil)
}

// TestUdpClientEcho exercises udp:Client (init/sendDatagram/receiveDatagram/
// close) against a bare Go UDP echo server.
func TestUdpClientEcho(t *testing.T) {
	skipIfNoLoopback(t)
	port := goUDPEchoServer(t)

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/udp;

public function main() returns error? {
    udp:Client c = check new ({timeout: 5});
    check c->sendDatagram({remoteHost: "127.0.0.1", remotePort: %d, data: "hello".toBytes()});
    readonly & udp:Datagram echoed = check c->receiveDatagram();
    io:println('string:fromBytes(echoed.data)); // @output hello

    check c->close();
    udp:Error? afterClose = c->sendDatagram({remoteHost: "127.0.0.1", remotePort: %d, data: "x".toBytes()});
    io:println(afterClose is udp:Error); // @output true
    return;
}
`, port, port)

	tmpDir := t.TempDir()
	tmpBalFile := filepath.Join(tmpDir, "udp-client-echo-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "udp-client-echo-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "udp-client-echo-v.txtar"),
	}
	runExtern(t, tc, newUdpPal(), nil)
}
