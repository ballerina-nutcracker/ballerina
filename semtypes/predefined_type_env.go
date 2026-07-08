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
	"ballerina-lang-go/common"
	"sync"
)

// initializedTypeAtom is a generic record holding an atomic type and its index
type initializedTypeAtom[E interface {
	atomicType
	comparable
}] struct {
	atomicType E
	index      int
}

// predefinedTypeEnv is a utility class used to create various type atoms that need to be initialized
// without an environment and common to all environments.
type predefinedTypeEnv struct {
	// Storage lists
	initializedCellAtoms       []initializedTypeAtom[*cellAtomicType]
	initializedListAtoms       []initializedTypeAtom[*ListAtomicType]
	initializedMappingAtoms    []initializedTypeAtom[*MappingAtomicType]
	initializedRecListAtoms    []*ListAtomicType
	initializedRecMappingAtoms []*MappingAtomicType
	nextAtomIndex              int

	// cellAtomicType fields
	_cellAtomicVal                    *cellAtomicType
	_cellAtomicNever                  *cellAtomicType
	_callAtomicInner                  *cellAtomicType
	_cellAtomicInnerMapping           *cellAtomicType
	_cellAtomicInnerMappingRO         *cellAtomicType
	_cellAtomicInnerRO                *cellAtomicType
	_cellAtomicUndef                  *cellAtomicType
	_cellAtomicValRO                  *cellAtomicType
	_cellAtomicObjectMember           *cellAtomicType
	_cellAtomicObjectMemberKind       *cellAtomicType
	_cellAtomicObjectMemberRO         *cellAtomicType
	_cellAtomicObjectMemberVisibility *cellAtomicType
	_cellAtomicMappingArray           *cellAtomicType
	_cellAtomicMappingArrayRO         *cellAtomicType

	// ListAtomicType fields
	_listAtomicMapping        *ListAtomicType
	_listAtomicMappingRO      *ListAtomicType
	_listAtomicThreeElementRO *ListAtomicType
	_listAtomicTwoElement     *ListAtomicType
	_listAtomicThreeElement   *ListAtomicType
	_listAtomicRO             *ListAtomicType

	// MappingAtomicType fields
	_mappingAtomicObject         *MappingAtomicType
	_mappingAtomicObjectMember   *MappingAtomicType
	_mappingAtomicObjectMemberRO *MappingAtomicType
	_mappingAtomicObjectRO       *MappingAtomicType
	_mappingAtomicRO             *MappingAtomicType

	// typeAtom fields
	_atomCellInner                  *typeAtom
	_atomCellInnerMapping           *typeAtom
	_atomCellInnerMappingRO         *typeAtom
	_atomCellInnerRO                *typeAtom
	_atomCellNever                  *typeAtom
	_atomCellObjectMember           *typeAtom
	_atomCellObjectMemberKind       *typeAtom
	_atomCellObjectMemberRO         *typeAtom
	_atomCellObjectMemberVisibility *typeAtom
	_atomCellUndef                  *typeAtom
	_atomCellVal                    *typeAtom
	_atomCellValRO                  *typeAtom
	_atomListMapping                *typeAtom
	_atomListMappingRO              *typeAtom
	_atomListTwoElement             *typeAtom
	_atomMappingObject              *typeAtom
	_atomMappingObjectMember        *typeAtom
	_atomMappingObjectMemberRO      *typeAtom
	_atomCellMappingArray           *typeAtom
	_atomCellMappingArrayRO         *typeAtom
	_atomListThreeElement           *typeAtom
	_atomListThreeElementRO         *typeAtom
}

// Package-level singleton instance
var predefinedTypeEnvInstance *predefinedTypeEnv
var predefinedTypeEnvInitializer sync.Once

// predefinedTypeEnvGetInstance returns the singleton instance
func predefinedTypeEnvGetInstance() *predefinedTypeEnv {
	predefinedTypeEnvInitializer.Do(func() {
		predefinedTypeEnvInstance = &predefinedTypeEnv{
			initializedCellAtoms:       make([]initializedTypeAtom[*cellAtomicType], 0),
			initializedListAtoms:       make([]initializedTypeAtom[*ListAtomicType], 0),
			initializedMappingAtoms:    make([]initializedTypeAtom[*MappingAtomicType], 0),
			initializedRecListAtoms:    make([]*ListAtomicType, 0),
			initializedRecMappingAtoms: make([]*MappingAtomicType, 0),
			nextAtomIndex:              0,
		}
	})
	return predefinedTypeEnvInstance
}

// Helper methods

// addInitializedCellAtom adds a cellAtomicType to the initialized atoms list
func (p *predefinedTypeEnv) addInitializedCellAtom(atom *cellAtomicType) {
	addInitializedAtom(p, &p.initializedCellAtoms, atom)
}

// addInitializedListAtom adds a ListAtomicType to the initialized atoms list
func (p *predefinedTypeEnv) addInitializedListAtom(atom *ListAtomicType) {
	addInitializedAtom(p, &p.initializedListAtoms, atom)
}

// addInitializedMapAtom adds a MappingAtomicType to the initialized atoms list
func (p *predefinedTypeEnv) addInitializedMapAtom(atom *MappingAtomicType) {
	addInitializedAtom(p, &p.initializedMappingAtoms, atom)
}

// addInitializedAtom is a generic function to add an atom to the atoms list with an index
func addInitializedAtom[E interface {
	atomicType
	comparable
}](env *predefinedTypeEnv, atoms *[]initializedTypeAtom[E], atom E) {
	*atoms = append(*atoms, initializedTypeAtom[E]{atomicType: atom, index: env.nextAtomIndex})
	env.nextAtomIndex++
}

// cellAtomIndex returns the index of a cellAtomicType in the initialized atoms list
func (p *predefinedTypeEnv) cellAtomIndex(atom *cellAtomicType) int {
	return atomIndex(p.initializedCellAtoms, atom)
}

// listAtomIndex returns the index of a ListAtomicType in the initialized atoms list
func (p *predefinedTypeEnv) listAtomIndex(atom *ListAtomicType) int {
	return atomIndex(p.initializedListAtoms, atom)
}

// mappingAtomIndex returns the index of a MappingAtomicType in the initialized atoms list
func (p *predefinedTypeEnv) mappingAtomIndex(atom *MappingAtomicType) int {
	return atomIndex(p.initializedMappingAtoms, atom)
}

// atomIndex is a generic function to find the index of an atom in the atoms list.
func atomIndex[E interface {
	atomicType
	comparable
}](initializedAtoms []initializedTypeAtom[E], atom E) int {
	for _, initializedAtom := range initializedAtoms {
		if initializedAtom.atomicType == atom {
			return initializedAtom.index
		}
	}
	panic("IndexOutOfBoundsException")
}

// Getter methods

// cellAtomicVal returns the cellAtomicType for VAL with limited mutability
func (p *predefinedTypeEnv) cellAtomicVal() *cellAtomicType {
	if p._cellAtomicVal == nil {
		val := cellAtomicTypeFrom(VAL, CellMutability_CELL_MUT_LIMITED)
		p._cellAtomicVal = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicVal
}

// atomCellVal returns the typeAtom for cell val
func (p *predefinedTypeEnv) atomCellVal() *typeAtom {
	if p._atomCellVal == nil {
		cellAtomicVal := p.cellAtomicVal()
		atomCellVal := createTypeAtom(p.cellAtomIndex(cellAtomicVal), cellAtomicVal)
		p._atomCellVal = &atomCellVal
	}
	return p._atomCellVal
}

// cellAtomicNever returns the cellAtomicType for NEVER with limited mutability
func (p *predefinedTypeEnv) cellAtomicNever() *cellAtomicType {
	if p._cellAtomicNever == nil {
		val := cellAtomicTypeFrom(NEVER, CellMutability_CELL_MUT_LIMITED)
		p._cellAtomicNever = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicNever
}

// atomCellNever returns the typeAtom for cell never
func (p *predefinedTypeEnv) atomCellNever() *typeAtom {
	if p._atomCellNever == nil {
		cellAtomicNever := p.cellAtomicNever()
		atomCellNever := createTypeAtom(p.cellAtomIndex(cellAtomicNever), cellAtomicNever)
		p._atomCellNever = &atomCellNever
	}
	return p._atomCellNever
}

// cellAtomicInner returns the cellAtomicType for INNER with limited mutability
func (p *predefinedTypeEnv) cellAtomicInner() *cellAtomicType {
	if p._callAtomicInner == nil {
		val := cellAtomicTypeFrom(INNER, CellMutability_CELL_MUT_LIMITED)
		p._callAtomicInner = &val
		p.addInitializedCellAtom(&val)
	}
	return p._callAtomicInner
}

// atomCellInner returns the typeAtom for cell inner
func (p *predefinedTypeEnv) atomCellInner() *typeAtom {
	if p._atomCellInner == nil {
		cellAtomicInner := p.cellAtomicInner()
		atomCellInner := createTypeAtom(p.cellAtomIndex(cellAtomicInner), cellAtomicInner)
		p._atomCellInner = &atomCellInner
	}
	return p._atomCellInner
}

// cellAtomicInnerMapping returns the cellAtomicType for union(MAPPING, UNDEF) with limited mutability
func (p *predefinedTypeEnv) cellAtomicInnerMapping() *cellAtomicType {
	if p._cellAtomicInnerMapping == nil {
		val := cellAtomicTypeFrom(Union(MAPPING, UNDEF), CellMutability_CELL_MUT_LIMITED)
		p._cellAtomicInnerMapping = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicInnerMapping
}

// atomCellInnerMapping returns the typeAtom for cell inner mapping
func (p *predefinedTypeEnv) atomCellInnerMapping() *typeAtom {
	if p._atomCellInnerMapping == nil {
		cellAtomicInnerMapping := p.cellAtomicInnerMapping()
		atomCellInnerMapping := createTypeAtom(p.cellAtomIndex(cellAtomicInnerMapping), cellAtomicInnerMapping)
		p._atomCellInnerMapping = &atomCellInnerMapping
	}
	return p._atomCellInnerMapping
}

// cellAtomicInnerMappingRO returns the cellAtomicType for union(MAPPING_RO, UNDEF) with limited mutability
func (p *predefinedTypeEnv) cellAtomicInnerMappingRO() *cellAtomicType {
	if p._cellAtomicInnerMappingRO == nil {
		val := cellAtomicTypeFrom(Union(MAPPING_RO, UNDEF), CellMutability_CELL_MUT_LIMITED)
		p._cellAtomicInnerMappingRO = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicInnerMappingRO
}

// atomCellInnerMappingRO returns the typeAtom for cell inner mapping RO
func (p *predefinedTypeEnv) atomCellInnerMappingRO() *typeAtom {
	if p._atomCellInnerMappingRO == nil {
		cellAtomicInnerMappingRO := p.cellAtomicInnerMappingRO()
		atomCellInnerMappingRO := createTypeAtom(p.cellAtomIndex(cellAtomicInnerMappingRO), cellAtomicInnerMappingRO)
		p._atomCellInnerMappingRO = &atomCellInnerMappingRO
	}
	return p._atomCellInnerMappingRO
}

// listAtomicMapping returns the ListAtomicType for empty fixed length array with CELL_SEMTYPE_INNER_MAPPING
func (p *predefinedTypeEnv) listAtomicMapping() *ListAtomicType {
	if p._listAtomicMapping == nil {
		val := listAtomicTypeFrom(fixedLengthArrayEmpty(), CELL_SEMTYPE_INNER_MAPPING)
		p._listAtomicMapping = &val
		p.addInitializedListAtom(&val)
	}
	return p._listAtomicMapping
}

// atomListMapping returns the typeAtom for list mapping
func (p *predefinedTypeEnv) atomListMapping() *typeAtom {
	if p._atomListMapping == nil {
		listAtomicMapping := p.listAtomicMapping()
		atomListMapping := createTypeAtom(p.listAtomIndex(listAtomicMapping), listAtomicMapping)
		p._atomListMapping = &atomListMapping
	}
	return p._atomListMapping
}

// listAtomicMappingRO returns the ListAtomicType for empty fixed length array with CELL_SEMTYPE_INNER_MAPPING_RO
func (p *predefinedTypeEnv) listAtomicMappingRO() *ListAtomicType {
	if p._listAtomicMappingRO == nil {
		val := listAtomicTypeFrom(fixedLengthArrayEmpty(), CELL_SEMTYPE_INNER_MAPPING_RO)
		p._listAtomicMappingRO = &val
		p.addInitializedListAtom(&val)
	}
	return p._listAtomicMappingRO
}

// atomListMappingRO returns the typeAtom for list mapping RO
func (p *predefinedTypeEnv) atomListMappingRO() *typeAtom {
	if p._atomListMappingRO == nil {
		listAtomicMappingRO := p.listAtomicMappingRO()
		atomListMappingRO := createTypeAtom(p.listAtomIndex(listAtomicMappingRO), listAtomicMappingRO)
		p._atomListMappingRO = &atomListMappingRO
	}
	return p._atomListMappingRO
}

// cellAtomicInnerRO returns the cellAtomicType for INNER_READONLY with no mutability
func (p *predefinedTypeEnv) cellAtomicInnerRO() *cellAtomicType {
	if p._cellAtomicInnerRO == nil {
		val := cellAtomicTypeFrom(INNER_READONLY, CellMutability_CELL_MUT_NONE)
		p._cellAtomicInnerRO = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicInnerRO
}

// atomCellInnerRO returns the typeAtom for cell inner RO
func (p *predefinedTypeEnv) atomCellInnerRO() *typeAtom {
	if p._atomCellInnerRO == nil {
		cellAtomicInnerRO := p.cellAtomicInnerRO()
		atomCellInnerRO := createTypeAtom(p.cellAtomIndex(cellAtomicInnerRO), cellAtomicInnerRO)
		p._atomCellInnerRO = &atomCellInnerRO
	}
	return p._atomCellInnerRO
}

// cellAtomicUndef returns the cellAtomicType for UNDEF with no mutability
func (p *predefinedTypeEnv) cellAtomicUndef() *cellAtomicType {
	if p._cellAtomicUndef == nil {
		val := cellAtomicTypeFrom(UNDEF, CellMutability_CELL_MUT_NONE)
		p._cellAtomicUndef = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicUndef
}

// atomCellUndef returns the typeAtom for cell undef
func (p *predefinedTypeEnv) atomCellUndef() *typeAtom {
	if p._atomCellUndef == nil {
		cellAtomicUndef := p.cellAtomicUndef()
		atomCellUndef := createTypeAtom(p.cellAtomIndex(cellAtomicUndef), cellAtomicUndef)
		p._atomCellUndef = &atomCellUndef
	}
	return p._atomCellUndef
}

// listAtomicTwoElement returns the ListAtomicType for two-element list with CELL_SEMTYPE_VAL and CELL_SEMTYPE_UNDEF
func (p *predefinedTypeEnv) listAtomicTwoElement() *ListAtomicType {
	if p._listAtomicTwoElement == nil {
		val := listAtomicTypeFrom(fixedLengthArrayFrom([]SemType{CELL_SEMTYPE_VAL}, 2), CELL_SEMTYPE_UNDEF)
		p._listAtomicTwoElement = &val
		p.addInitializedListAtom(&val)
	}
	return p._listAtomicTwoElement
}

// atomListTwoElement returns the typeAtom for list two element
func (p *predefinedTypeEnv) atomListTwoElement() *typeAtom {
	if p._atomListTwoElement == nil {
		listAtomicTwoElement := p.listAtomicTwoElement()
		atomListTwoElement := createTypeAtom(p.listAtomIndex(listAtomicTwoElement), listAtomicTwoElement)
		p._atomListTwoElement = &atomListTwoElement
	}
	return p._atomListTwoElement
}

// cellAtomicValRO returns the cellAtomicType for VAL_READONLY with no mutability
func (p *predefinedTypeEnv) cellAtomicValRO() *cellAtomicType {
	if p._cellAtomicValRO == nil {
		val := cellAtomicTypeFrom(VAL_READONLY, CellMutability_CELL_MUT_NONE)
		p._cellAtomicValRO = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicValRO
}

// atomCellValRO returns the typeAtom for cell val RO
func (p *predefinedTypeEnv) atomCellValRO() *typeAtom {
	if p._atomCellValRO == nil {
		cellAtomicValRO := p.cellAtomicValRO()
		atomCellValRO := createTypeAtom(p.cellAtomIndex(cellAtomicValRO), cellAtomicValRO)
		p._atomCellValRO = &atomCellValRO
	}
	return p._atomCellValRO
}

// mappingAtomicObjectMemberRO returns the MappingAtomicType for object member RO
func (p *predefinedTypeEnv) mappingAtomicObjectMemberRO() *MappingAtomicType {
	if p._mappingAtomicObjectMemberRO == nil {
		val := mappingAtomicTypeFrom(
			[]string{"kind", "value", "visibility"},
			[]SemType{CELL_SEMTYPE_OBJECT_MEMBER_KIND, CELL_SEMTYPE_VAL_RO, CELL_SEMTYPE_OBJECT_MEMBER_VISIBILITY},
			CELL_SEMTYPE_UNDEF)
		p._mappingAtomicObjectMemberRO = &val
		p.addInitializedMapAtom(&val)
	}
	return p._mappingAtomicObjectMemberRO
}

// atomMappingObjectMemberRO returns the typeAtom for mapping object member RO
func (p *predefinedTypeEnv) atomMappingObjectMemberRO() *typeAtom {
	if p._atomMappingObjectMemberRO == nil {
		mappingAtomicObjectMemberRO := p.mappingAtomicObjectMemberRO()
		atomMappingObjectMemberRO := createTypeAtom(p.mappingAtomIndex(mappingAtomicObjectMemberRO), mappingAtomicObjectMemberRO)
		p._atomMappingObjectMemberRO = &atomMappingObjectMemberRO
	}
	return p._atomMappingObjectMemberRO
}

// cellAtomicObjectMemberRO returns the cellAtomicType for object member RO
func (p *predefinedTypeEnv) cellAtomicObjectMemberRO() *cellAtomicType {
	if p._cellAtomicObjectMemberRO == nil {
		val := cellAtomicTypeFrom(MAPPING_SEMTYPE_OBJECT_MEMBER_RO, CellMutability_CELL_MUT_NONE)
		p._cellAtomicObjectMemberRO = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicObjectMemberRO
}

// atomCellObjectMemberRO returns the typeAtom for cell object member RO
func (p *predefinedTypeEnv) atomCellObjectMemberRO() *typeAtom {
	if p._atomCellObjectMemberRO == nil {
		cellAtomicObjectMemberRO := p.cellAtomicObjectMemberRO()
		atomCellObjectMemberRO := createTypeAtom(p.cellAtomIndex(cellAtomicObjectMemberRO), cellAtomicObjectMemberRO)
		p._atomCellObjectMemberRO = &atomCellObjectMemberRO
	}
	return p._atomCellObjectMemberRO
}

// cellAtomicObjectMemberKind returns the cellAtomicType for object member kind
func (p *predefinedTypeEnv) cellAtomicObjectMemberKind() *cellAtomicType {
	if p._cellAtomicObjectMemberKind == nil {
		val := cellAtomicTypeFrom(Union(StringConst("field"), StringConst("method")), CellMutability_CELL_MUT_NONE)
		p._cellAtomicObjectMemberKind = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicObjectMemberKind
}

// atomCellObjectMemberKind returns the typeAtom for cell object member kind
func (p *predefinedTypeEnv) atomCellObjectMemberKind() *typeAtom {
	if p._atomCellObjectMemberKind == nil {
		cellAtomicObjectMemberKind := p.cellAtomicObjectMemberKind()
		atomCellObjectMemberKind := createTypeAtom(p.cellAtomIndex(cellAtomicObjectMemberKind), cellAtomicObjectMemberKind)
		p._atomCellObjectMemberKind = &atomCellObjectMemberKind
	}
	return p._atomCellObjectMemberKind
}

// cellAtomicObjectMemberVisibility returns the cellAtomicType for object member visibility
func (p *predefinedTypeEnv) cellAtomicObjectMemberVisibility() *cellAtomicType {
	if p._cellAtomicObjectMemberVisibility == nil {
		val := cellAtomicTypeFrom(Union(StringConst("public"), StringConst("private")), CellMutability_CELL_MUT_NONE)
		p._cellAtomicObjectMemberVisibility = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicObjectMemberVisibility
}

// atomCellObjectMemberVisibility returns the typeAtom for cell object member visibility
func (p *predefinedTypeEnv) atomCellObjectMemberVisibility() *typeAtom {
	if p._atomCellObjectMemberVisibility == nil {
		cellAtomicObjectMemberVisibility := p.cellAtomicObjectMemberVisibility()
		atomCellObjectMemberVisibility := createTypeAtom(p.cellAtomIndex(cellAtomicObjectMemberVisibility), cellAtomicObjectMemberVisibility)
		p._atomCellObjectMemberVisibility = &atomCellObjectMemberVisibility
	}
	return p._atomCellObjectMemberVisibility
}

// mappingAtomicObjectMember returns the MappingAtomicType for object member
func (p *predefinedTypeEnv) mappingAtomicObjectMember() *MappingAtomicType {
	if p._mappingAtomicObjectMember == nil {
		val := mappingAtomicTypeFrom(
			[]string{"kind", "value", "visibility"},
			[]SemType{CELL_SEMTYPE_OBJECT_MEMBER_KIND, CELL_SEMTYPE_VAL, CELL_SEMTYPE_OBJECT_MEMBER_VISIBILITY},
			CELL_SEMTYPE_UNDEF)
		p._mappingAtomicObjectMember = &val
		p.addInitializedMapAtom(&val)
	}
	return p._mappingAtomicObjectMember
}

// atomMappingObjectMember returns the typeAtom for mapping object member
func (p *predefinedTypeEnv) atomMappingObjectMember() *typeAtom {
	if p._atomMappingObjectMember == nil {
		mappingAtomicObjectMember := p.mappingAtomicObjectMember()
		atomMappingObjectMember := createTypeAtom(p.mappingAtomIndex(mappingAtomicObjectMember), mappingAtomicObjectMember)
		p._atomMappingObjectMember = &atomMappingObjectMember
	}
	return p._atomMappingObjectMember
}

// cellAtomicObjectMember returns the cellAtomicType for object member
func (p *predefinedTypeEnv) cellAtomicObjectMember() *cellAtomicType {
	if p._cellAtomicObjectMember == nil {
		val := cellAtomicTypeFrom(MAPPING_SEMTYPE_OBJECT_MEMBER, CellMutability_CELL_MUT_UNLIMITED)
		p._cellAtomicObjectMember = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicObjectMember
}

// atomCellObjectMember returns the typeAtom for cell object member
func (p *predefinedTypeEnv) atomCellObjectMember() *typeAtom {
	if p._atomCellObjectMember == nil {
		cellAtomicObjectMember := p.cellAtomicObjectMember()
		atomCellObjectMember := createTypeAtom(p.cellAtomIndex(cellAtomicObjectMember), cellAtomicObjectMember)
		p._atomCellObjectMember = &atomCellObjectMember
	}
	return p._atomCellObjectMember
}

// mappingAtomicObject returns the MappingAtomicType for object
func (p *predefinedTypeEnv) mappingAtomicObject() *MappingAtomicType {
	if p._mappingAtomicObject == nil {
		val := mappingAtomicTypeFrom(
			[]string{"$qualifiers"},
			[]SemType{CELL_SEMTYPE_OBJECT_QUALIFIER},
			CELL_SEMTYPE_OBJECT_MEMBER)
		p._mappingAtomicObject = &val
		p.addInitializedMapAtom(&val)
	}
	return p._mappingAtomicObject
}

// atomMappingObject returns the typeAtom for mapping object
func (p *predefinedTypeEnv) atomMappingObject() *typeAtom {
	if p._atomMappingObject == nil {
		mappingAtomicObject := p.mappingAtomicObject()
		atomMappingObject := createTypeAtom(p.mappingAtomIndex(mappingAtomicObject), mappingAtomicObject)
		p._atomMappingObject = &atomMappingObject
	}
	return p._atomMappingObject
}

// listAtomicRO returns the ListAtomicType for read-only list
func (p *predefinedTypeEnv) listAtomicRO() *ListAtomicType {
	if p._listAtomicRO == nil {
		val := listAtomicTypeFrom(fixedLengthArrayEmpty(), CELL_SEMTYPE_INNER_RO)
		p._listAtomicRO = &val
		p.initializedRecListAtoms = append(p.initializedRecListAtoms, &val)
	}
	return p._listAtomicRO
}

// mappingAtomicRO returns the MappingAtomicType for read-only mapping
func (p *predefinedTypeEnv) mappingAtomicRO() *MappingAtomicType {
	if p._mappingAtomicRO == nil {
		val := mappingAtomicTypeFrom([]string{}, []SemType{}, CELL_SEMTYPE_INNER_RO)
		p._mappingAtomicRO = &val
		p.initializedRecMappingAtoms = append(p.initializedRecMappingAtoms, &val)
	}
	return p._mappingAtomicRO
}

// getMappingAtomicObjectRO returns the MappingAtomicType for read-only object
func (p *predefinedTypeEnv) getMappingAtomicObjectRO() *MappingAtomicType {
	if p._mappingAtomicObjectRO == nil {
		val := mappingAtomicTypeFrom(
			[]string{"$qualifiers"},
			[]SemType{CELL_SEMTYPE_OBJECT_QUALIFIER},
			CELL_SEMTYPE_OBJECT_MEMBER_RO)
		p._mappingAtomicObjectRO = &val
		p.initializedRecMappingAtoms = append(p.initializedRecMappingAtoms, &val)
	}
	return p._mappingAtomicObjectRO
}

// cellAtomicMappingArray returns the cellAtomicType for mapping array
func (p *predefinedTypeEnv) cellAtomicMappingArray() *cellAtomicType {
	if p._cellAtomicMappingArray == nil {
		val := cellAtomicTypeFrom(MAPPING_ARRAY, CellMutability_CELL_MUT_LIMITED)
		p._cellAtomicMappingArray = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicMappingArray
}

// atomCellMappingArray returns the typeAtom for cell mapping array
func (p *predefinedTypeEnv) atomCellMappingArray() *typeAtom {
	if p._atomCellMappingArray == nil {
		cellAtomicMappingArray := p.cellAtomicMappingArray()
		atomCellMappingArray := createTypeAtom(p.cellAtomIndex(cellAtomicMappingArray), cellAtomicMappingArray)
		p._atomCellMappingArray = &atomCellMappingArray
	}
	return p._atomCellMappingArray
}

// cellAtomicMappingArrayRO returns the cellAtomicType for read-only mapping array
func (p *predefinedTypeEnv) cellAtomicMappingArrayRO() *cellAtomicType {
	if p._cellAtomicMappingArrayRO == nil {
		val := cellAtomicTypeFrom(MAPPING_ARRAY_RO, CellMutability_CELL_MUT_LIMITED)
		p._cellAtomicMappingArrayRO = &val
		p.addInitializedCellAtom(&val)
	}
	return p._cellAtomicMappingArrayRO
}

// atomCellMappingArrayRO returns the typeAtom for cell mapping array RO
func (p *predefinedTypeEnv) atomCellMappingArrayRO() *typeAtom {
	if p._atomCellMappingArrayRO == nil {
		cellAtomicMappingArrayRO := p.cellAtomicMappingArrayRO()
		atomCellMappingArrayRO := createTypeAtom(p.cellAtomIndex(cellAtomicMappingArrayRO), cellAtomicMappingArrayRO)
		p._atomCellMappingArrayRO = &atomCellMappingArrayRO
	}
	return p._atomCellMappingArrayRO
}

// listAtomicThreeElement returns the ListAtomicType for three-element list
func (p *predefinedTypeEnv) listAtomicThreeElement() *ListAtomicType {
	if p._listAtomicThreeElement == nil {
		val := listAtomicTypeFrom(
			fixedLengthArrayFrom([]SemType{CELL_SEMTYPE_LIST_SUBTYPE_MAPPING, CELL_SEMTYPE_VAL}, 3),
			CELL_SEMTYPE_UNDEF)
		p._listAtomicThreeElement = &val
		p.addInitializedListAtom(&val)
	}
	return p._listAtomicThreeElement
}

// atomListThreeElement returns the typeAtom for list three element
func (p *predefinedTypeEnv) atomListThreeElement() *typeAtom {
	if p._atomListThreeElement == nil {
		listAtomicThreeElement := p.listAtomicThreeElement()
		atomListThreeElement := createTypeAtom(p.listAtomIndex(listAtomicThreeElement), listAtomicThreeElement)
		p._atomListThreeElement = &atomListThreeElement
	}
	return p._atomListThreeElement
}

// listAtomicThreeElementRO returns the ListAtomicType for read-only three-element list
func (p *predefinedTypeEnv) listAtomicThreeElementRO() *ListAtomicType {
	if p._listAtomicThreeElementRO == nil {
		val := listAtomicTypeFrom(
			fixedLengthArrayFrom([]SemType{CELL_SEMTYPE_LIST_SUBTYPE_MAPPING_RO, CELL_SEMTYPE_VAL}, 3),
			CELL_SEMTYPE_UNDEF)
		p._listAtomicThreeElementRO = &val
		p.addInitializedListAtom(&val)
	}
	return p._listAtomicThreeElementRO
}

// atomListThreeElementRO returns the typeAtom for list three element RO
func (p *predefinedTypeEnv) atomListThreeElementRO() *typeAtom {
	if p._atomListThreeElementRO == nil {
		listAtomicThreeElementRO := p.listAtomicThreeElementRO()
		atomListThreeElementRO := createTypeAtom(p.listAtomIndex(listAtomicThreeElementRO), listAtomicThreeElementRO)
		p._atomListThreeElementRO = &atomListThreeElementRO
	}
	return p._atomListThreeElementRO
}

// ReservedRecAtomCount returns the maximum count of reserved rec atoms
func (p *predefinedTypeEnv) ReservedRecAtomCount() int {
	if len(p.initializedRecListAtoms) > len(p.initializedRecMappingAtoms) {
		return len(p.initializedRecListAtoms)
	}
	return len(p.initializedRecMappingAtoms)
}

// GetPredefinedRecAtom returns a predefined recAtom for the given index
func (p *predefinedTypeEnv) GetPredefinedRecAtom(index int) common.Optional[*recAtom] {
	if p.IsPredefinedRecAtom(index) {
		recAtom := createRecAtom(index)
		return common.OptionalOf(&recAtom)
	}
	return common.OptionalEmpty[*recAtom]()
}

// IsPredefinedRecAtom checks if the given index is a predefined rec atom
func (p *predefinedTypeEnv) IsPredefinedRecAtom(index int) bool {
	return index >= 0 && index < p.ReservedRecAtomCount()
}
