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
	"errors"
	"net"
	"sync/atomic"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

// TestTcpServiceEcho starts a tcp:Listener + attached service on a real port
// and drives it from testMain via a real tcp:Client, exercising the full
// listener -> onConnect -> onBytes -> Caller.writeBytes path.
func TestTcpServiceEcho(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-echo-v"), newTcpPal(), nil)
}

// TestTcpServiceOnConnectError verifies that a connection is closed (not
// left stuck forever, as in jBallerina) when onConnect returns an error.
func TestTcpServiceOnConnectError(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-reject-v"), newTcpPal(), nil)
}

// TestTcpServiceLifecycle exercises Listener.detach, Listener.immediateStop and
// starting a listener with no service attached.
func TestTcpServiceLifecycle(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-lifecycle-v"), newTcpPal(), nil)
}

// TestTcpServiceInline exercises the anonymous
// `service on new tcp:Listener(...) { ... }` declaration style.
func TestTcpServiceInline(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-inline-v"), newTcpPal(), nil)
}

// TestTcpServiceTimeouts exercises Client.readBytes' read-timeout error path
// against a listener that never writes back.
func TestTcpServiceTimeouts(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-timeouts-v"), newTcpPal(), nil)
}

// TestTcpServiceOnBytesBareForm exercises the bare onBytes(readonly & byte[])
// signature (no Caller parameter) — see onBytesArgs in native/dispatch.go.
func TestTcpServiceOnBytesBareForm(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-onbytes-bare-v"), newTcpPal(), nil)
}

func TestTcpServiceOnCloseOnly(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-onclose-only-v"), newTcpPal(), nil)
}

// TestTcpServiceLocalClose verifies that closing the Caller from onBytes
// dispatches onClose without a spurious onError.
func TestTcpServiceLocalClose(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-local-close-v"), newTcpPal(), nil)
}

// TestTcpServicePanic verifies that a panic in onBytes or onConnect, or an
// error returned from onBytes, is logged instead of crashing the interpreter.
func TestTcpServicePanic(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-panic-v"), newTcpPal(), nil)
}

// TestTcpServiceInvalid verifies that services jBallerina's compiler plugin
// rejects are rejected at attach or when onConnect returns them.
func TestTcpServiceInvalid(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-invalid-v"), newTcpPal(), nil)
}

// TestTcpServiceWriteFail verifies that Caller writes and data returned from
// onBytes fail once the peer has closed and reset the connection.
func TestTcpServiceWriteFail(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-write-fail-v"), newTcpPal(), nil)
}

// TestTcpServiceReturnFail verifies that a failed write of the data returned
// from onBytes routes one onError and closes the connection.
func TestTcpServiceReturnFail(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-return-fail-v"), newTcpPal(), nil)
}

// TestTcpClientWriteTimeout verifies that Client.writeBytes fails with a
// write timeout when the peer stops reading.
func TestTcpClientWriteTimeout(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-client-write-timeout-v"), newTcpPal(), nil)
}

// TestTcpServiceAcceptRetry verifies that a transient Accept error does not
// stop the listener's accept loop.
func TestTcpServiceAcceptRetry(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-accept-retry-v"), &flakyAcceptPal{tcpPal: newTcpPal()}, nil)
}

// TestTcpServiceNoConnectionService verifies that a connection whose
// onConnect returns () stays tracked, so immediateStop still closes it.
func TestTcpServiceNoConnectionService(t *testing.T) {
	t.Parallel()
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-no-conn-svc-v"), newTcpPal(), nil)
}

type flakyAcceptPal struct {
	*tcpPal
}

func (p *flakyAcceptPal) Platform() pal.Platform {
	base := p.tcpPal.Platform()
	base.Net.Listen = func(network, address string, tlsCfg *pal.ServerTLSConfig) (net.Listener, error) {
		ln, err := palnative.ListenTCP(network, address, tlsCfg)
		if err != nil {
			return nil, err
		}
		return &flakyListener{Listener: ln}, nil
	}
	return base
}

// flakyListener fails its first Accept with a transient error.
type flakyListener struct {
	net.Listener
	failed atomic.Bool
}

func (l *flakyListener) Accept() (net.Conn, error) {
	if l.failed.CompareAndSwap(false, true) {
		return nil, errors.New("transient accept failure")
	}
	return l.Listener.Accept()
}
