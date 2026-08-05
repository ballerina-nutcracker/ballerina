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

import "testing"

// TestUdpServiceOnBytesAutoReply starts a udp:Listener + attached service on
// a real port and drives it from testMain via a real udp:ConnectClient,
// exercising the listener -> onBytes -> auto-reply path.
func TestUdpServiceOnBytesAutoReply(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-echo-v"), newUdpPal(), nil)
}

// TestUdpServiceOnDatagramCallerReply exercises the onDatagram dispatch path
// with an explicit Caller.sendDatagram reply, driven via a connectionless
// udp:Client.
func TestUdpServiceOnDatagramCallerReply(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-datagram-v"), newUdpPal(), nil)
}

// TestUdpServiceLifecycle exercises Listener.detach and Listener.immediateStop.
func TestUdpServiceLifecycle(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-lifecycle-v"), newUdpPal(), nil)
}

// TestUdpServiceInline exercises the anonymous
// `service on new udp:Listener(...) { ... }` declaration style.
func TestUdpServiceInline(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-inline-v"), newUdpPal(), nil)
}

// TestUdpServiceOnBytesBareForm exercises the bare onBytes(readonly & byte[])
// signature (no Caller parameter) — see remoteMethodArgs in native/dispatch.go.
func TestUdpServiceOnBytesBareForm(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-onbytes-bare-v"), newUdpPal(), nil)
}

// TestUdpServiceOnDatagramBareForm exercises the bare
// onDatagram(readonly & udp:Datagram) signature (no Caller parameter) — the
// exact shape used by this module's own official examples. See
// remoteMethodArgs in native/dispatch.go.
func TestUdpServiceOnDatagramBareForm(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-ondatagram-bare-v"), newUdpPal(), nil)
}

// TestUdpServiceIPv6 exercises a dual-stack-bound listener replying to a
// ConnectClient dialing an IPv6 literal ("::1") — see resolveUDPAddr in
// native/udp.go.
func TestUdpServiceIPv6(t *testing.T) {
	skipIfNoLoopback(t)
	skipIfNoIPv6Loopback(t)
	runExtern(t, fileCase("udp-service/udp-svc-ipv6-v"), newUdpPal(), nil)
}
