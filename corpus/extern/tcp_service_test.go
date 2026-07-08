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

// TestTcpServiceEcho starts a tcp:Listener + attached service on a real port
// and drives it from testMain via a real tcp:Client, exercising the full
// listener -> onConnect -> onBytes -> Caller.writeBytes path.
func TestTcpServiceEcho(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-echo-v"), newTcpPal(), nil)
}

// TestTcpServiceOnConnectError verifies that a connection is closed (not
// left stuck forever, as in jBallerina) when onConnect returns an error.
func TestTcpServiceOnConnectError(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-reject-v"), newTcpPal(), nil)
}

// TestTcpServiceLifecycle exercises Listener.detach and Listener.immediateStop.
func TestTcpServiceLifecycle(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("tcp-service/tcp-svc-lifecycle-v"), newTcpPal(), nil)
}
