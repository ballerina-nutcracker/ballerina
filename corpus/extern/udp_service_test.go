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
	"net"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
)

// udpPal wraps the in-memory TestPal with the real UDP Net implementations.
type udpPal struct {
	testharness.TestPal
}

func newUdpPal() *udpPal {
	return &udpPal{TestPal: testharness.NewTestPal()}
}

func (p *udpPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{
		DialPacket:        palnative.DialPacket,
		ListenPacket:      palnative.ListenPacket,
		ResolvePacketAddr: palnative.ResolvePacketAddr,
	}
	return base
}

// skipIfNoIPv6Loopback skips tests that need "::1"; some CI environments disable IPv6.
func skipIfNoIPv6Loopback(t *testing.T) {
	t.Helper()
	pc, err := net.ListenPacket("udp6", "[::1]:0")
	if err != nil {
		t.Skip("skipping IPv6-loopback-dependent test: ::1 unavailable")
	}
	_ = pc.Close()
}

func runUdpService(t *testing.T, name string) {
	t.Helper()
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/"+name), newUdpPal(), nil)
}

func TestUdpServiceOnBytesAutoReply(t *testing.T) {
	runUdpService(t, "udp-svc-echo-v")
}

func TestUdpServiceOnDatagramCallerReply(t *testing.T) {
	runUdpService(t, "udp-svc-datagram-v")
}

func TestUdpServiceLifecycle(t *testing.T) {
	runUdpService(t, "udp-svc-lifecycle-v")
}

func TestUdpServiceInline(t *testing.T) {
	runUdpService(t, "udp-svc-inline-v")
}

func TestUdpServiceOnBytesBareForm(t *testing.T) {
	runUdpService(t, "udp-svc-onbytes-bare-v")
}

func TestUdpServiceOnDatagramBareForm(t *testing.T) {
	runUdpService(t, "udp-svc-ondatagram-bare-v")
}

func TestUdpServiceClient(t *testing.T) {
	runUdpService(t, "udp-svc-client-v")
}

func TestUdpServiceHandlerError(t *testing.T) {
	runUdpService(t, "udp-svc-handler-error-v")
}

func TestUdpServiceIPv6(t *testing.T) {
	skipIfNoIPv6Loopback(t)
	runUdpService(t, "udp-svc-ipv6-v")
}
