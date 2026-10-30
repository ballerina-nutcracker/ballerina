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

import "sync"

// ModuleKey identifies the module a TestPlan was discovered for.
type ModuleKey struct {
	Org    string
	Module string
}

// store is process-local, in-memory state shared between a compiler plugin
// (the writer, running once per module during compilation) and whatever
// reads it afterward in the same process (a frontend, or a test asserting
// parity against another producer). Plugins run concurrently, one goroutine
// per module (confirmed against PR #921's actual dispatch code), so this
// needs real synchronization, not just documentation saying so.
//
// Known limitation, by design so far: a key carries no revision or
// compilation-instance identity, and Store always overwrites unconditionally
// — there is no way for a reader to tell which compile produced the plan it
// gets back, or to detect that a newer compile is already in flight. A
// one-shot CLI process never notices, since exactly one compile of any given
// module is ever in flight at a time. A long-lived host that recompiles the
// same module repeatedly (a language server, reacting to edits) would need
// to add that itself — e.g. by never calling Store/Load concurrently for the
// same key, or by layering its own generation check on top — since nothing
// here provides one. This was raised in review and deliberately left
// unaddressed rather than guessed at: no such host exists yet to validate
// the right shape against (see the implementation plan's notes on this).
var store sync.Map // ModuleKey -> *TestPlan

// Store records plan as the TestPlan discovered for key, overwriting any
// previous entry for the same key.
func Store(key ModuleKey, plan *TestPlan) {
	store.Store(key, plan)
}

// Load returns the TestPlan previously stored for key, if any.
func Load(key ModuleKey) (*TestPlan, bool) {
	v, ok := store.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*TestPlan), true
}

// Forget removes the stored TestPlan for key, if any — a no-op if nothing
// was stored. A one-shot CLI process never needs this (the whole store is
// freed when the process exits); a long-lived host that opens and closes
// many modules over its lifetime should call this when a module is removed
// from the project, so the store doesn't grow for as long as the process
// runs. There is no bulk/clear-all operation — evict what you opened.
func Forget(key ModuleKey) {
	store.Delete(key)
}
