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
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
)

// TestMessagingListenerRetrievePanic checks that a store whose retrieve panics doesn't block the listener's stop.
// It asserts on stdout directly, since the task module's panic log line embeds a timestamp and a job id.
func TestMessagingListenerRetrievePanic(t *testing.T) {
	t.Parallel()
	pal := testharness.NewTestPal()
	done := make(chan struct{})
	go func() {
		defer close(done)
		testharness.Run(t, fileCase("messaging/messaging-listener-retrieve-panic-v"), pal, nil)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("immediateStop hung after the store's retrieve panicked")
	}
	stdout := pal.Stdout()
	if !strings.HasSuffix(stdout, "stopped\n") {
		t.Errorf("stdout = %q, want it to end with %q", stdout, "stopped\n")
	}
	if !strings.Contains(stdout, "]. broken store\n") {
		t.Errorf("stdout missing the job's panic log line, got: %q", stdout)
	}
}
