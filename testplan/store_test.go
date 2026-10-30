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

package testplan

import "testing"

func TestStoreAndLoad(t *testing.T) {
	key := ModuleKey{Org: "testorg", Module: "store_test_mod_a"}
	plan := &TestPlan{Tests: []TestFunction{{Name: "testA"}}}

	Store(key, plan)
	got, ok := Load(key)
	if !ok {
		t.Fatal("expected Load to find the stored plan")
	}
	if got != plan {
		t.Error("expected Load to return the exact same pointer Store was given")
	}
}

func TestLoad_MissingKeyReturnsFalse(t *testing.T) {
	_, ok := Load(ModuleKey{Org: "testorg", Module: "store_test_never_stored"})
	if ok {
		t.Error("expected Load to report false for a key nothing ever stored")
	}
}

func TestStore_OverwritesPreviousEntry(t *testing.T) {
	key := ModuleKey{Org: "testorg", Module: "store_test_mod_b"}
	first := &TestPlan{Tests: []TestFunction{{Name: "old"}}}
	second := &TestPlan{Tests: []TestFunction{{Name: "new"}}}

	Store(key, first)
	Store(key, second)

	got, ok := Load(key)
	if !ok || got != second {
		t.Errorf("expected the second Store to win, got %+v, ok=%v", got, ok)
	}
}

// TestForget_RemovesTheEntry guards the one concrete fix made for a
// long-lived-host (e.g. a language server) scenario raised in review: Store
// never evicts anything on its own, so a process that opens and closes many
// modules over its lifetime needs an explicit way to bound the store's size.
func TestForget_RemovesTheEntry(t *testing.T) {
	key := ModuleKey{Org: "testorg", Module: "store_test_mod_c"}
	Store(key, &TestPlan{Tests: []TestFunction{{Name: "testA"}}})

	Forget(key)

	if _, ok := Load(key); ok {
		t.Error("expected Load to find nothing after Forget")
	}
}

func TestForget_MissingKeyIsANoOp(t *testing.T) {
	// Must not panic for a key that was never stored.
	Forget(ModuleKey{Org: "testorg", Module: "store_test_never_stored_either"})
}
