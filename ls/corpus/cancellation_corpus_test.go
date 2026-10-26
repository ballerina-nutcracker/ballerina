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
)

func TestNumericIDCancelCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/numeric-id-cancel.json")
}

func TestStringIDCancelCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/string-id-cancel.json")
}

func TestCancelExactlyOnceCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/cancel-exactly-once.json")
}

func TestCancelNoopCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/cancel-noop.json")
}

func TestDuplicateInflightIDRejectedCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/duplicate-inflight-id-rejected.json")
}

func TestNormalCompletionCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/normal-completion.json")
}

// didClose must clear diagnostics before the blocked request's cancellation reply.
func TestOrderedNotificationsDuringCancelCorpus(t *testing.T) {
	t.Parallel()
	runTranscript(t, "cancellation/testdata/ordered-notifications-during-cancel.json")
}
