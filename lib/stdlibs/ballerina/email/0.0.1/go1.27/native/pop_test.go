// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

// TestPopTimeouts checks that a stalled POP3 server fails the connection instead of blocking it.
func TestPopTimeouts(t *testing.T) {
	// Not parallel: overrides the package-level receiveTimeout.
	saved := receiveTimeout
	receiveTimeout = 200 * time.Millisecond
	t.Cleanup(func() { receiveTimeout = saved })
	platform := pal.Platform{
		Net:  pal.Net{Dial: palnative.Dial, UpgradeTLS: palnative.UpgradeTLS},
		Time: pal.Time{Now: time.Now},
	}
	for _, greeting := range []string{"", "+OK ready\r\n"} {
		port, _ := stalledServer(t, greeting)
		client := popClient{&mailbox{mailServer: mailServer{host: "127.0.0.1", port: port, security: "START_TLS_NEVER"}}}
		start := time.Now()
		_, err := client.connect(platform)
		if err == nil || err.Message != "Connect failed" {
			t.Fatalf("greeting %q: connect error = %v, want Connect failed", greeting, errorMessage(err))
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("greeting %q: connect took %v, want it bounded by the timeout", greeting, elapsed)
		}
	}
}
