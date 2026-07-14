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

package semtypes

import (
	"runtime"
	"testing"
)

func defineRuntimeList(env Env, member SemType) SemType {
	ld := NewListDefinition()
	return ld.DefineListTypeWrappedWithEnvSemType(env, member)
}

func TestFreezeRoutesNewAtomsToEphemeralStore(t *testing.T) {
	env := CreateTypeEnv()
	env.Freeze()

	env.atomTableMutex.Lock()
	before := len(env.atomTable)
	env.atomTableMutex.Unlock()

	ty := defineRuntimeList(env, STRING)

	env.atomTableMutex.Lock()
	after := len(env.atomTable)
	env.atomTableMutex.Unlock()

	if after != before {
		t.Fatalf("atomTable grew after freeze: before=%d after=%d", before, after)
	}

	atom := ToListAtomicType(ContextFrom(env), ty)
	if atom == nil {
		t.Fatal("expected a list atomic type for the runtime list")
	}
	if got := len(env.ephemeralSlots); got == 0 {
		t.Fatal("expected ephemeral slots to be populated")
	}
	for _, slot := range env.ephemeralSlots {
		if p := slot.ptr.Value(); p != nil && p.index() < env.frozenAtomCount {
			t.Fatalf("ephemeral atom idx %d must be >= frozenAtomCount %d", p.index(), env.frozenAtomCount)
		}
	}
}

func TestEphemeralAtomIsReclaimedAndSlotReused(t *testing.T) {
	env := CreateTypeEnv()
	env.Freeze()

	// Hold a runtime type, capture its ephemeral slot, then drop it.
	slotsAfterFirst := func() int {
		ty := defineRuntimeList(env, STRING)
		if n := liveEphemeralSlots(env); n == 0 {
			t.Fatal("expected at least one live ephemeral slot")
		}
		slots := len(env.ephemeralSlots)
		runtime.KeepAlive(ty)
		return slots
	}()

	// Force collection; the slot's weak pointer should clear.
	reclaimed := false
	for i := 0; i < 5 && !reclaimed; i++ {
		runtime.GC()
		if liveEphemeralSlots(env) == 0 {
			reclaimed = true
		}
	}
	if !reclaimed {
		t.Fatal("expected the ephemeral atom to be garbage collected")
	}

	// A new runtime type reuses the freed slot instead of growing the array.
	ty2 := defineRuntimeList(env, INT)
	if ToListAtomicType(ContextFrom(env), ty2) == nil {
		t.Fatal("expected a list atomic type for the second runtime list")
	}
	if grew := len(env.ephemeralSlots); grew > slotsAfterFirst {
		t.Fatalf("expected slot reuse, but slots grew from %d to %d", slotsAfterFirst, grew)
	}
	runtime.KeepAlive(ty2)
}

func TestFrozenEnvForbidsRecursiveTypes(t *testing.T) {
	env := CreateTypeEnv()
	env.Freeze()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a panic when allocating a recursive atom after freeze")
		}
	}()
	ld := &ListDefinition{}
	ld.GetSemType(env) // allocates a rec atom -> must panic
}

func liveEphemeralSlots(env Env) int {
	env.ephemeralMu.Lock()
	defer env.ephemeralMu.Unlock()
	n := 0
	for _, slot := range env.ephemeralSlots {
		if slot.ptr.Value() != nil {
			n++
		}
	}
	return n
}
