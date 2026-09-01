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

package values

import (
	"fmt"
	"math"
	"strings"
	"unsafe"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type List struct {
	Type       semtypes.SemType
	atomic     *semtypes.ListAtomicType
	isReadonly bool
	elems      []BalValue
	filler     FillerFactory
}

// NewList constructs a fully initialized list without applying any inherent
// type or readonly checks. The resulting length is max(size, len(initial));
// initial values occupy the low indices and any remaining slots are populated
// by the filler factory.
func NewList(ty semtypes.SemType, atomic *semtypes.ListAtomicType, isReadonly bool,
	filler FillerFactory, size int, initial []BalValue,
) *List {
	if atomic == nil {
		panic("values.NewList: atomic type must not be nil")
	}
	length := max(len(initial), size)
	elems := make([]BalValue, length)
	copy(elems, initial)
	if len(initial) < length {
		panic("values.NewList: missing values")
	}
	return &List{
		Type:       ty,
		atomic:     atomic,
		isReadonly: isReadonly,
		elems:      elems,
		filler:     filler,
	}
}

func (l *List) Len() int {
	return len(l.elems)
}

func (l *List) IsReadonly() bool {
	return l.isReadonly
}

func (l *List) Get(idx int) BalValue {
	return l.elems[idx]
}

// FillingSet stores value at idx, resizing the list if necessary.
// Panics if the list is readonly or value does not belong to the inherent
// member type at idx. Filler-grown intermediate slots are not type-checked.
func (l *List) FillingSet(tc semtypes.Context, idx int, value BalValue) {
	l.checkMutable()
	l.checkMemberType(tc, idx, value)
	currentLen := len(l.elems)
	if idx < currentLen {
		l.elems[idx] = value
		return
	}
	if l.filler == nil {
		panic(NewErrorWithMessage("no filler value"))
	}
	if idx >= math.MaxInt32 {
		panic(NewErrorWithMessage("list too long"))
	}
	newLen := idx + 1
	if newLen <= cap(l.elems) {
		l.elems = l.elems[:newLen]
		for i := currentLen; i < idx; i++ {
			l.elems[i] = l.filler()
		}
		l.elems[idx] = value
		return
	}
	for len(l.elems) < idx {
		l.elems = append(l.elems, l.filler())
	}
	l.elems = append(l.elems, value)
}

// FillingGet returns the value at idx, growing the list with filler values when idx is beyond the current length.
// Panics if growth is required and the list is readonly.
func (l *List) FillingGet(idx int) BalValue {
	if idx < len(l.elems) {
		return l.elems[idx]
	}
	l.checkMutable()
	if l.filler == nil {
		panic(NewErrorWithMessage("no filler value"))
	}
	if idx >= math.MaxInt32 {
		panic(NewErrorWithMessage("list too long"))
	}
	for len(l.elems) <= idx {
		l.elems = append(l.elems, l.filler())
	}
	return l.elems[idx]
}

// Append adds values at the tail, checking each against the inherent member
// type at its eventual index.
func (l *List) Append(tc semtypes.Context, vs ...BalValue) {
	l.checkMutable()
	base := len(l.elems)
	for i, v := range vs {
		l.checkMemberType(tc, base+i, v)
	}
	l.elems = append(l.elems, vs...)
}

// RemoveAt removes and returns the element at idx, shifting subsequent elements down.
// Panics if the removal would leave a value outside its own inherent type, either
// by dropping a mandatory member or by sliding a member into a slot it does not
// belong to.
func (l *List) RemoveAt(tc semtypes.Context, idx int) BalValue {
	l.checkMutable()
	l.checkCanShrinkTo(len(l.elems) - 1)
	l.checkShiftedMemberTypes(tc, idx)
	val := l.elems[idx]
	copy(l.elems[idx:], l.elems[idx+1:])
	last := len(l.elems) - 1
	l.elems[last] = nil // release the vacated slot so the GC can reclaim it
	l.elems = l.elems[:last]
	return val
}

// Clear removes all elements from the list.
// Panics if the inherent type mandates any member.
func (l *List) Clear() {
	l.checkMutable()
	l.checkCanShrinkTo(0)
	clear(l.elems) // release references before truncating so the GC can reclaim them
	l.elems = l.elems[:0]
}

// checkCanShrinkTo panics if newLen is below the number of members the inherent
// type mandates. Both `int[2]` and `[int, string, int...]` fix their leading
// members; only members past that prefix are backed by the rest type and may be
// removed. Testing the mandatory count rather than the rest type covers the
// fixed-length case too, since there every member is mandatory.
func (l *List) checkCanShrinkTo(newLen int) {
	mandatory := l.atomic.FixedLength()
	if newLen >= mandatory {
		return
	}
	if semtypes.IsNever(l.atomic.Rest()) {
		panic(NewErrorWithMessage("inherent type violation: cannot change the length of a fixed-length list"))
	}
	panic(NewErrorWithMessage(fmt.Sprintf(
		"inherent type violation: cannot reduce the length of a tuple below its %d mandatory member(s)", mandatory)))
}

// checkShiftedMemberTypes panics if removing idx would slide a member into a
// mandatory slot whose type it does not belong to — removing index 0 of
// `[int, string, int...]` would otherwise leave the string in the int slot.
func (l *List) checkShiftedMemberTypes(tc semtypes.Context, idx int) {
	for pos := idx; pos < l.atomic.FixedLength(); pos++ {
		l.checkMemberType(tc, pos, l.elems[pos+1])
	}
}

func (l *List) checkMutable() {
	if l.isReadonly {
		panic(NewErrorWithMessage("inherent type violation: cannot mutate readonly value"))
	}
}

func (l *List) checkMemberType(tc semtypes.Context, idx int, value BalValue) {
	memberTy := l.atomic.MemberAtInnerVal(idx)
	valueTy := SemTypeForValue(value)
	if !semtypes.IsSubtype(tc, valueTy, memberTy) {
		panic(NewErrorWithMessage("inherent type violation"))
	}
}

func (l *List) String(visited map[uintptr]bool) string {
	ptr := uintptr(unsafe.Pointer(l))
	if visited[ptr] {
		return "[...]"
	}
	if l.Len() > 0 {
		if inner, ok := l.Get(0).(*List); ok && inner == l {
			return "[...]"
		}
	}
	visited[ptr] = true
	defer delete(visited, ptr)
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < l.Len(); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(toString(l.Get(i), visited, false))
	}
	b.WriteByte(']')
	return b.String()
}
