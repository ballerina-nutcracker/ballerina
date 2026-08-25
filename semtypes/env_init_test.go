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
// software distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package semtypes

import (
	"testing"
)

// TestEnvInitAtomTable tests environment initialization with atom table
// Ported from EnvInitTest.java:testEnvInitAtomTable()
func TestEnvInitAtomTable(t *testing.T) {
	env := CreateTypeEnv()

	// Ensure atoms are in the table by calling Env methods
	cellAtomicVal := cellAtomicTypeFrom(Val, CellMutabilityLimited)
	typeAtom0 := env.cellAtom(&cellAtomicVal)

	cellAtomicNever := cellAtomicTypeFrom(Never, CellMutabilityLimited)
	typeAtom1 := env.cellAtom(&cellAtomicNever)

	cellAtomicInner := cellAtomicTypeFrom(Inner, CellMutabilityLimited)
	typeAtom2 := env.cellAtom(&cellAtomicInner)

	cellAtomicInnerMapping := cellAtomicTypeFrom(Union(Mapping, Undef), CellMutabilityLimited)
	typeAtom3 := env.cellAtom(&cellAtomicInnerMapping)

	listAtomicMapping := listAtomicTypeFrom(fixedLengthArrayEmpty(), cellSemtypeInnerMapping)
	typeAtom4 := env.listAtom(&listAtomicMapping)

	typeAtom5 := env.cellAtom(cellAtomicInnerMappingRo)

	listAtomicMappingRo := listAtomicTypeFrom(fixedLengthArrayEmpty(), cellSemtypeInnerMappingRo)
	typeAtom6 := env.listAtom(&listAtomicMappingRo)

	cellAtomicInnerRo := cellAtomicTypeFrom(ReadonlyInner, CellMutabilityNone)
	typeAtom7 := env.cellAtom(&cellAtomicInnerRo)

	cellAtomicUndef := cellAtomicTypeFrom(Undef, CellMutabilityNone)
	typeAtom8 := env.cellAtom(&cellAtomicUndef)

	listAtomicTwoElement := listAtomicTypeFrom(
		fixedLengthArrayFrom([]SemType{cellSemtypeVal}, 2),
		cellSemtypeUndef,
	)
	typeAtom9 := env.listAtom(&listAtomicTwoElement)

	// Now check the atomTable
	env.atomTableMutex.Lock()
	atomTable := env.atomTable
	env.atomTableMutex.Unlock()

	// Check that the atomTable contains at least the expected entries
	// Note: The Go implementation may have more atoms than the Java version
	assertTrue(t, len(atomTable) >= 19, "atomTable should have at least 19 entries, got %d", len(atomTable))

	// Verify the atoms are in the table and match
	ta0, ok := atomTable[&cellAtomicVal]
	assertTrue(t, ok, "cellAtomicVal should be in atomTable")
	assertEqual(t, ta0.AtomicType, &cellAtomicVal)
	assertEqual(t, ta0, *typeAtom0)

	ta1, ok := atomTable[&cellAtomicNever]
	assertTrue(t, ok, "cellAtomicNever should be in atomTable")
	assertEqual(t, ta1.AtomicType, &cellAtomicNever)
	assertEqual(t, ta1, *typeAtom1)

	ta2, ok := atomTable[&cellAtomicInner]
	assertTrue(t, ok, "cellAtomicInner should be in atomTable")
	assertEqual(t, ta2.AtomicType, &cellAtomicInner)
	assertEqual(t, ta2, *typeAtom2)

	ta3, ok := atomTable[&cellAtomicInnerMapping]
	assertTrue(t, ok, "cellAtomicInnerMapping should be in atomTable")
	assertEqual(t, ta3.AtomicType, &cellAtomicInnerMapping)
	assertEqual(t, ta3, *typeAtom3)

	ta4, ok := atomTable[&listAtomicMapping]
	assertTrue(t, ok, "listAtomicMapping should be in atomTable")
	assertEqual(t, ta4.AtomicType, &listAtomicMapping)
	assertEqual(t, ta4, *typeAtom4)

	ta5, ok := atomTable[cellAtomicInnerMappingRo]
	assertTrue(t, ok, "cellAtomicInnerMappingRo should be in atomTable")
	assertEqual(t, ta5.AtomicType, cellAtomicInnerMappingRo)
	assertEqual(t, ta5, *typeAtom5)

	ta6, ok := atomTable[&listAtomicMappingRo]
	assertTrue(t, ok, "listAtomicMappingRo should be in atomTable")
	assertEqual(t, ta6.AtomicType, &listAtomicMappingRo)
	assertEqual(t, ta6, *typeAtom6)

	ta7, ok := atomTable[&cellAtomicInnerRo]
	assertTrue(t, ok, "cellAtomicInnerRo should be in atomTable")
	assertEqual(t, ta7.AtomicType, &cellAtomicInnerRo)
	assertEqual(t, ta7, *typeAtom7)

	ta8, ok := atomTable[&cellAtomicUndef]
	assertTrue(t, ok, "cellAtomicUndef should be in atomTable")
	assertEqual(t, ta8.AtomicType, &cellAtomicUndef)
	assertEqual(t, ta8, *typeAtom8)

	ta9, ok := atomTable[&listAtomicTwoElement]
	assertTrue(t, ok, "listAtomicTwoElement should be in atomTable")
	assertEqual(t, ta9.AtomicType, &listAtomicTwoElement)
	assertEqual(t, ta9, *typeAtom9)
}

// TestTypeAtomIndices tests type atom indices uniqueness
// Ported from EnvInitTest.java:testTypeAtomIndices()
func TestTypeAtomIndices(t *testing.T) {
	env := CreateTypeEnv()

	env.atomTableMutex.Lock()
	atomTable := env.atomTable
	env.atomTableMutex.Unlock()

	indices := make(map[int]bool)
	for _, typeAtom := range atomTable {
		index := typeAtom.index()
		if indices[index] {
			t.Errorf("Duplicate index found: %d", index)
		}
		indices[index] = true
	}
}

// TestEnvInitRecAtoms tests recursive atoms initialization
// Ported from EnvInitTest.java:testEnvInitRecAtoms()
func TestEnvInitRecAtoms(t *testing.T) {
	env := CreateTypeEnv()

	// Test recListAtoms
	env.recListAtomsMutex.Lock()
	recListAtoms := env.recListAtoms
	env.recListAtomsMutex.Unlock()

	// 2 predefined + 2 preallocated (json, anydata)
	assertEqual(t, len(recListAtoms), 4)
	if recListAtoms[0] == nil {
		t.Error("recListAtoms[0] should not be nil")
	} else if recListAtoms[0] != listAtomicRo {
		t.Errorf("recListAtoms[0] does not match expected ListAtomicType")
	}
	if recListAtoms[1] != nil {
		t.Error("recListAtoms[1] should be nil")
	}
	if recListAtoms[2] == nil {
		t.Error("recListAtoms[2] (json) should not be nil")
	}
	if recListAtoms[3] == nil {
		t.Error("recListAtoms[3] (anydata) should not be nil")
	}

	// Test recMappingAtoms
	env.recMappingAtomsMutex.Lock()
	recMappingAtoms := env.recMappingAtoms
	env.recMappingAtomsMutex.Unlock()

	// 2 predefined + 2 preallocated (json, anydata)
	assertEqual(t, len(recMappingAtoms), 4)
	if recMappingAtoms[0] == nil {
		t.Error("recMappingAtoms[0] should not be nil")
	} else if recMappingAtoms[0] != mappingAtomicRo {
		t.Errorf("recMappingAtoms[0] does not match mappingAtomicRo")
	}
	if recMappingAtoms[1] == nil {
		t.Error("recMappingAtoms[1] should not be nil")
	} else if recMappingAtoms[1] != mappingAtomicObjectRo {
		t.Errorf("recMappingAtoms[1] does not match mappingAtomicObjectRo")
	}
	if recMappingAtoms[2] == nil {
		t.Error("recMappingAtoms[2] (json) should not be nil")
	}
	if recMappingAtoms[3] == nil {
		t.Error("recMappingAtoms[3] (anydata) should not be nil")
	}

	// Test recFunctionAtoms
	env.recFunctionAtomsMutex.Lock()
	recFunctionAtoms := env.recFunctionAtoms
	env.recFunctionAtomsMutex.Unlock()

	assertEqual(t, len(recFunctionAtoms), 0)
}
