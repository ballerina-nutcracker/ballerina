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
	"testing"
	"time"

	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/test_util/testharness"
	"ballerina-lang-go/values"
)

// TestTcpServiceOnError exercises the connection-service onError dispatch
// path: dialing the listener directly (bypassing tcp:Client) and forcing a
// TCP reset (RST) via SetLinger(0) surfaces as a non-EOF read error on the
// server's read loop, which routes to onError rather than a normal close.
func TestTcpServiceOnError(t *testing.T) {
	skipIfNoLoopback(t)

	connected := make(chan struct{}, 1)
	signal := make(chan struct{}, 1)
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "tcp-svc-onerror-v", FuncName: "signalConnected",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				select {
				case connected <- struct{}{}:
				default:
				}
				return nil, nil
			}},
		{Org: "$anon", Module: "tcp-svc-onerror-v", FuncName: "resetRawConnection",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				port := args[0].(int64)
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
				if err != nil {
					return values.NewErrorWithMessage("dial: " + err.Error()), nil
				}
				select {
				case <-connected:
				case <-time.After(2 * time.Second):
					_ = conn.Close()
					return values.NewErrorWithMessage("timed out waiting for onConnect"), nil
				}
				if tcpConn, ok := conn.(*net.TCPConn); ok {
					_ = tcpConn.SetLinger(0)
				}
				_ = conn.Close()
				return nil, nil
			}},
		{Org: "$anon", Module: "tcp-svc-onerror-v", FuncName: "signalOnError",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				select {
				case signal <- struct{}{}:
				default:
				}
				return nil, nil
			}},
		{Org: "$anon", Module: "tcp-svc-onerror-v", FuncName: "waitForOnError",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				select {
				case <-signal:
					return nil, nil
				case <-time.After(2 * time.Second):
					return values.NewErrorWithMessage("timed out waiting for onError"), nil
				}
			}},
	}
	runExtern(t, fileCase("tcp-service/tcp-svc-onerror-v"), newTcpPal(), externs)
}
