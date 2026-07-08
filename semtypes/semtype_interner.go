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

// SemtypeInterner is meant to be used to intern copies of the same semtype. IMPORTANT: this does not
// actually check if two given types are equal under IsEqual. SemtypeInterner is thread compatible,
// leaving users to apply suitable synchronization logic where needed.
type SemtypeInterner struct {
	basicTypeHandles      map[basicTypeBitSet]InternHandle
	complexSemtypeHandles map[complexSemtypeInternKey]complexSemtypeInternValues
}

type InternHandle int64

type complexSemtypeInternKey struct {
	all  basicTypeBitSet
	some basicTypeBitSet
}

type complexSemtypeInternValues struct {
	base      int32
	dataLists [][]ProperSubtypeData
}

func NewSemtypeInterner() *SemtypeInterner {
	return &SemtypeInterner{
		basicTypeHandles:      make(map[basicTypeBitSet]InternHandle),
		complexSemtypeHandles: make(map[complexSemtypeInternKey]complexSemtypeInternValues),
	}
}

func (i *SemtypeInterner) Intern(ty SemType) InternHandle {
	if ty.some() == 0 {
		bits := ty.all()
		if handle, ok := i.basicTypeHandles[bits]; ok {
			return handle
		}
		handle := basicInternHandle(bits)
		i.basicTypeHandles[bits] = handle
		return handle
	}
	key := complexSemtypeInternKey{all: ty.all(), some: ty.some()}
	values := i.complexSemtypeHandles[key]
	dataList := ty.subtypeDataList()
	for idx, existing := range values.dataLists {
		if sameSubtypeDataList(existing, dataList) {
			return complexInternHandle(values.base, idx)
		}
	}
	if values.base == 0 {
		values.base = int32(len(i.complexSemtypeHandles) + 1)
	}
	idx := len(values.dataLists)
	values.dataLists = append(values.dataLists, dataList)
	i.complexSemtypeHandles[key] = values
	return complexInternHandle(values.base, idx)
}

func (i *SemtypeInterner) Lookup(ty SemType) (InternHandle, bool) {
	if ty.some() == 0 {
		return basicInternHandle(ty.all()), true
	}
	key := complexSemtypeInternKey{all: ty.all(), some: ty.some()}
	values, ok := i.complexSemtypeHandles[key]
	if !ok {
		return 0, false
	}
	dataList := ty.subtypeDataList()
	for idx, existing := range values.dataLists {
		if sameSubtypeDataList(existing, dataList) {
			return complexInternHandle(values.base, idx), true
		}
	}
	return 0, false
}

func basicInternHandle(bits basicTypeBitSet) InternHandle {
	return InternHandle(-int(bits) - 1)
}

func complexInternHandle(base int32, index int) InternHandle {
	return InternHandle(index)<<32 | InternHandle(uint32(base))
}

func sameComplexSemType(a, b SemType) bool {
	return a.all() == b.all() && a.some() == b.some() &&
		sameSubtypeDataList(a.subtypeDataList(), b.subtypeDataList())
}

func sameSubtypeDataList(a, b []ProperSubtypeData) bool {
	if len(a) != len(b) {
		return false
	}
	for idx := range a {
		if !sameSubtypeData(a[idx], b[idx]) {
			return false
		}
	}
	return true
}

func sameSubtypeData(a, b ProperSubtypeData) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch a.(type) {
	case intSubtype, floatSubtype, decimalSubtype, stringSubtype:
		return false
	}
	return a == b
}
