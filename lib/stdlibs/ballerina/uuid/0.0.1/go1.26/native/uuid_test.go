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

package native

import (
	"strings"
	"testing"

	"ballerina-lang-go/values"
)

// The uuid module's behaviour is exercised end-to-end through the corpus
// tests in corpus/bal/library/subset3/uuid*.bal, which run the full compiler
// -> BIR -> interpreter pipeline. parseHexUintExtern's invalid-hex branch has
// no corpus equivalent: its only caller, toRecord(), always pre-validates the
// hex string (via validate()'s regex for string input, or getUuidFromBytes's
// %x formatting for byte[] input) before parsing, so no Ballerina-reachable
// value can trigger it.
func TestParseHexUintExtern_InvalidHex(t *testing.T) {
	result, err := parseHexUintExtern(nil, []values.BalValue{"not-hex"})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	balErr, ok := result.(*values.Error)
	if !ok {
		t.Fatalf("result = %T, want *values.Error", result)
	}
	if !strings.Contains(balErr.Message, "not-hex") {
		t.Errorf("error message = %q, want it to mention the invalid input", balErr.Message)
	}
}
