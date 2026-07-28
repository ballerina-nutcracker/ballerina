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
	"ballerina/decimal"
)

// Filler is a marker interface for all filler values.
type Filler interface {
	isFiller()
}

// SingleValueFiller is the filler for a type whose filler is an immediate
// value (nil, primitive default, or singleton shape).
type SingleValueFiller Value

// MappingFiller represents the filler of an empty mapping atomic type.
type MappingFiller struct {
	Atomic *MappingAtomicType
	Type   SemType
}

// ListFiller represents the filler of a list atomic type. Members holds
// the filler for each fixed-length slot. The runtime growth-filler (used
// when an out-of-range index forces the list to grow) is derived on demand
// from Atomic.Rest().
type ListFiller struct {
	Atomic  *ListAtomicType
	Type    SemType
	Members []Filler
}

// TableFiller represents empty table of type Type
type TableFiller struct {
	Type SemType
}

// ObjectFiller represents object of type Type
type ObjectFiller struct {
	Type SemType
}

// StreamFiller represents empty stream of type Type
type StreamFiller struct {
	Type SemType
}

// XMLFiller represents empty XML (xml “) of type Type
type XMLFiller struct {
	Type SemType
}

func (SingleValueFiller) isFiller() {}
func (MappingFiller) isFiller()     {}
func (ListFiller) isFiller()        {}
func (TableFiller) isFiller()       {}
func (ObjectFiller) isFiller()      {}
func (StreamFiller) isFiller()      {}
func (XMLFiller) isFiller()         {}

// FillerValue returns the filler value according to https://ballerina.io/spec/lang/master/#FillMember
// return nil if there is no filler value
func FillerValue(cx Context, t SemType) (Filler, bool) {
	if ContainsBasicType(t, NIL) {
		return SingleValueFiller(valueFrom(nil)), true
	}
	if shape := SingleShape(t); !shape.IsEmpty() {
		return SingleValueFiller(shape.Get()), true
	}
	bitset := widenToBasicTypeBits(t)
	if bitCount(bitset) != 1 {
		// Only nil containing unions can have filler values
		return nil, false
	}
	switch bitset {
	case booleanBits:
		return SingleValueFiller(valueFrom(false)), true
	case intBits:
		if IsSubtype(cx, IntConst(0), t) {
			return SingleValueFiller(valueFrom(int64(0))), true
		}
		return nil, false
	case floatBits:
		if IsSubtype(cx, FloatConst(0), t) {
			return SingleValueFiller(valueFrom(float64(0))), true
		}
		return nil, false
	case decimalBits:
		zero := decimal.FromInt64(0)
		if IsSubtype(cx, DecimalConst(*zero), t) {
			return SingleValueFiller(valueFrom(zero)), true
		}
		return nil, false
	case stringBits:
		if IsSubtype(cx, StringConst(""), t) {
			return SingleValueFiller(valueFrom("")), true
		}
		return nil, false
	case listBits:
		return listFiller(cx, t)
	case mappingBits:
		return mappingFiller(cx, t)
	case tableBits:
		return TableFiller{Type: t}, true
	case objectBits:
		return objectFiller(cx, t)
	case streamBits:
		return StreamFiller{Type: t}, true
	case xmlBits:
		return xmlFiller(cx, t)
	default:
		return nil, false
	}
}

func xmlFiller(cx Context, t SemType) (Filler, bool) {
	if IsSubtype(cx, XML_TEXT, t) {
		return XMLFiller{Type: t}, true
	}
	return nil, false
}

func objectFiller(cx Context, t SemType) (Filler, bool) {
	alts := ObjectAlternatives(cx, t)
	if len(alts) != 1 {
		return nil, false
	}
	if !initFnFillerCompatible(cx, alts[0].InitFnType) {
		return nil, false
	}
	return ObjectFiller{Type: alts[0].ObjectType}, true
}

func initFnFillerCompatible(cx Context, initFnTy SemType) bool {
	ld := NewListDefinition()
	emptyArgList := ld.TupleTypeWrapped(cx.Env())
	paramListTy := FunctionParamListType(cx, initFnTy)
	if IsZero(paramListTy) || !IsSubtype(cx, emptyArgList, paramListTy) {
		return false
	}
	retTy := FunctionReturnType(cx, initFnTy, emptyArgList)
	if IsZero(retTy) {
		return false
	}
	return !ContainsBasicType(retTy, ERROR)
}

func mappingFiller(cx Context, t SemType) (Filler, bool) {
	mat := ToMappingAtomicType(cx, t)
	// NOTE: this don't take into account default fields (Which is not a part of type)
	if mat == nil || len(mat.Names) != 0 {
		return nil, false
	}
	if filler, memoized := cx._fillerMemo[mat]; memoized {
		return filler, filler != nil
	}
	filler := MappingFiller{Atomic: mat, Type: t}
	cx._fillerMemo[mat] = filler
	return filler, true
}

func listFiller(cx Context, t SemType) (Filler, bool) {
	lat := ToListAtomicType(cx, t)
	if lat == nil {
		return nil, false
	}
	if filler, memoized := cx._fillerMemo[lat]; memoized {
		return filler, filler != nil
	}
	cx._fillerMemo[lat] = nil

	memberFillers := make([]Filler, lat.Members.FixedLength)
	for i := 0; i < lat.Members.FixedLength; i++ {
		filler, ok := FillerValue(cx, lat.MemberAtInnerVal(i))
		if !ok {
			return nil, false
		}
		memberFillers[i] = filler
	}
	filler := ListFiller{Atomic: lat, Type: t, Members: memberFillers}
	cx._fillerMemo[lat] = filler
	return filler, true
}
