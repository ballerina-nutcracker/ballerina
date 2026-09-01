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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package corpus

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

// TestNumericIDCancelCorpus drives a real $/cancelRequest against an in-flight
// $pal/blockRequest whose id is an integer. The cancelled request must reply
// with exactly one RequestCancelled (-32800) error; $/cancelRequest itself
// writes no response.
func TestNumericIDCancelCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/numeric-id-cancel.json")
}

// TestStringIDCancelCorpus covers the string-id variant of cancellation.
func TestStringIDCancelCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/string-id-cancel.json")
}

// TestCancelExactlyOnceCorpus verifies a duplicate $/cancelRequest for the
// same in-flight id produces a single -32800 reply, not two.
func TestCancelExactlyOnceCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/cancel-exactly-once.json")
}

// TestCancelNoopCorpus verifies unknown-id and malformed-param
// $/cancelRequest notifications are no-ops: they write no response and the
// blocked request completes normally when released.
func TestCancelNoopCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/cancel-noop.json")
}

// TestDuplicateInflightIDRejectedCorpus verifies a new request reusing an
// in-flight id is rejected with Invalid Request (-32600) and does not replace
// the original registry entry (the original still completes normally).
func TestDuplicateInflightIDRejectedCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/duplicate-inflight-id-rejected.json")
}

// TestNormalCompletionCorpus verifies a tracked request that is released
// (not cancelled) replies with its result.
func TestNormalCompletionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/normal-completion.json")
}

// TestOrderedNotificationsDuringCancelCorpus verifies document notifications
// remain synchronously processed in order while a request is blocked: a
// didClose's publishDiagnostics is written before the cancelled request's
// -32800 reply.
func TestOrderedNotificationsDuringCancelCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "cancellation/testdata/ordered-notifications-during-cancel.json")
}
