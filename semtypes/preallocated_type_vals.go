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

type preallocatedTypeVals struct {
	none      basicCellTypeVals
	limited   basicCellTypeVals
	unlimited basicCellTypeVals
}

type basicCellTypeVals struct {
	never          SemType
	nil            SemType
	boolean        SemType
	int            SemType
	float          SemType
	decimal        SemType
	string         SemType
	error          SemType
	list           SemType
	mapping        SemType
	table          SemType
	undef          SemType
	regexp         SemType
	function       SemType
	typedesc       SemType
	handle         SemType
	xml            SemType
	object         SemType
	stream         SemType
	future         SemType
	val            SemType
	inner          SemType
	any            SemType
	simpleOrString SemType
	number         SemType
	simpleBasic    SemType
}

func newPreallocatedTypeVals(env Env) preallocatedTypeVals {
	return preallocatedTypeVals{
		none:      newBasicCellTypeVals(env, CellMutability_CELL_MUT_NONE),
		limited:   newBasicCellTypeVals(env, CellMutability_CELL_MUT_LIMITED),
		unlimited: newBasicCellTypeVals(env, CellMutability_CELL_MUT_UNLIMITED),
	}
}

func newBasicCellTypeVals(env Env, mut CellMutability) basicCellTypeVals {
	return basicCellTypeVals{
		never:          createCellTypeVal(env, neverBits, mut),
		nil:            createCellTypeVal(env, nilBits, mut),
		boolean:        createCellTypeVal(env, booleanBits, mut),
		int:            createCellTypeVal(env, intBits, mut),
		float:          createCellTypeVal(env, floatBits, mut),
		decimal:        createCellTypeVal(env, decimalBits, mut),
		string:         createCellTypeVal(env, stringBits, mut),
		error:          createCellTypeVal(env, errorBits, mut),
		list:           createCellTypeVal(env, listBits, mut),
		mapping:        createCellTypeVal(env, mappingBits, mut),
		table:          createCellTypeVal(env, tableBits, mut),
		undef:          createCellTypeVal(env, undefBits, mut),
		regexp:         createCellTypeVal(env, regexpBits, mut),
		function:       createCellTypeVal(env, functionBits, mut),
		typedesc:       createCellTypeVal(env, typedescBits, mut),
		handle:         createCellTypeVal(env, handleBits, mut),
		xml:            createCellTypeVal(env, xmlBits, mut),
		object:         createCellTypeVal(env, objectBits, mut),
		stream:         createCellTypeVal(env, streamBits, mut),
		future:         createCellTypeVal(env, futureBits, mut),
		val:            createCellTypeVal(env, valBits, mut),
		inner:          createCellTypeVal(env, innerBits, mut),
		any:            createCellTypeVal(env, anyBits, mut),
		simpleOrString: createCellTypeVal(env, simpleOrStringBits, mut),
		number:         createCellTypeVal(env, numberBits, mut),
		simpleBasic:    createCellTypeVal(env, simpleBasicBits, mut),
	}
}

func createCellTypeVal(env Env, ty basicTypeBitSet, mut CellMutability) SemType {
	atomicCell := cellAtomicTypeFrom(ty.semType(), mut)
	atom := env.cellAtom(&atomicCell)
	bdd := bddAtom(atom)
	return getBasicSubtype(BTCell, bdd)
}

func (p *preallocatedTypeVals) basicTypeCell(ty basicTypeBitSet, mut CellMutability) (SemType, bool) {
	switch mut {
	case CellMutability_CELL_MUT_NONE:
		return p.none.basicTypeCell(ty)
	case CellMutability_CELL_MUT_LIMITED:
		return p.limited.basicTypeCell(ty)
	case CellMutability_CELL_MUT_UNLIMITED:
		return p.unlimited.basicTypeCell(ty)
	default:
		return SemType{}, false
	}
}

func (b *basicCellTypeVals) basicTypeCell(ty basicTypeBitSet) (SemType, bool) {
	switch ty {
	case neverBits:
		return b.never, true
	case nilBits:
		return b.nil, true
	case booleanBits:
		return b.boolean, true
	case intBits:
		return b.int, true
	case floatBits:
		return b.float, true
	case decimalBits:
		return b.decimal, true
	case stringBits:
		return b.string, true
	case errorBits:
		return b.error, true
	case listBits:
		return b.list, true
	case mappingBits:
		return b.mapping, true
	case tableBits:
		return b.table, true
	case undefBits:
		return b.undef, true
	case regexpBits:
		return b.regexp, true
	case functionBits:
		return b.function, true
	case typedescBits:
		return b.typedesc, true
	case handleBits:
		return b.handle, true
	case xmlBits:
		return b.xml, true
	case objectBits:
		return b.object, true
	case streamBits:
		return b.stream, true
	case futureBits:
		return b.future, true
	case valBits:
		return b.val, true
	case innerBits:
		return b.inner, true
	case anyBits:
		return b.any, true
	case simpleOrStringBits:
		return b.simpleOrString, true
	case numberBits:
		return b.number, true
	case simpleBasicBits:
		return b.simpleBasic, true
	default:
		return SemType{}, false
	}
}
