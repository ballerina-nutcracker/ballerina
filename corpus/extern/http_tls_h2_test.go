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
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestHttpServiceTLSHTTP2 checks which protocol a TLS listener negotiates with a client that
// offers HTTP/2 and HTTP/1.1.
func TestHttpServiceTLSHTTP2(t *testing.T) {
	skipIfNoLoopback(t)
	t.Parallel()
	module := "http-svc-tls-h2-v"
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: module, FuncName: "httpsProtocol",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				return httpsProtocol(args[0].(int64))
			}},
	}
	runExtern(t, fileCase("http-service/"+module), newHTTPPal(palnative.NewHTTPClient).withRealFS(), externs)
}

// httpsProtocol sends a GET over TLS, offering HTTP/2 and HTTP/1.1, and returns the protocol and
// status of the response.
func httpsProtocol(port int64) (string, error) {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test server certificate
		ForceAttemptHTTP2: true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/hello", port))
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()
	return fmt.Sprintf("%s %d", resp.Proto, resp.StatusCode), nil
}
