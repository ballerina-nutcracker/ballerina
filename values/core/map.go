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

package core

import (
	"fmt"
	"strings"
	"unsafe"

	"ballerina-lang-go/semtypes"
)

type mapEntry struct {
	key        string
	value      BalValue
	prev, next *mapEntry
}

// MapEntry is an ordered key/value pair used to seed a Map at construction.
type MapEntry struct {
	Key   string
	Value BalValue
}

type Map struct {
	Type       semtypes.SemType
	atomic     *semtypes.MappingAtomicType
	isReadonly bool

	data       map[string]*mapEntry
	head, tail *mapEntry
}

// NewMap constructs a fully initialized map without applying any inherent
// type or readonly checks on the seed entries. Entries are inserted in the
// order given.
func NewMap(ty semtypes.SemType, atomic *semtypes.MappingAtomicType, isReadonly bool, entries []MapEntry) *Map {
	if atomic == nil {
		panic("values.NewMap: atomic type must not be nil")
	}
	m := &Map{
		Type:       ty,
		atomic:     atomic,
		isReadonly: isReadonly,
		data:       make(map[string]*mapEntry, len(entries)),
	}
	for _, e := range entries {
		if existing, ok := m.data[e.Key]; ok {
			existing.value = e.Value
			continue
		}
		entry := &mapEntry{key: e.Key, value: e.Value}
		m.data[e.Key] = entry
		m.appendEntry(entry)
	}
	return m
}

// ShouldDeleteOnNilStore reports whether assigning nil to the given key
// should delete the entry. True iff the key names a declared optional
// field whose declared value type does not contain nil.
func (m *Map) ShouldDeleteOnNilStore(cx semtypes.Context, key string) bool {
	keyTy := semtypes.StringConst(key)
	fieldTy := semtypes.MappingMemberTypeInner(cx, m.Type, keyTy)
	if semtypes.ContainsBasicType(fieldTy, semtypes.NIL) {
		return false
	}
	return semtypes.AllMappingAtomsHaveOptionalFieldByName(cx, m.Type, key)
}

func (m *Map) Get(key string) (BalValue, bool) {
	if e, ok := m.data[key]; ok {
		return e.value, true
	}
	return nil, false
}

// FillingGet returns the value at key, inserting a fresh filler value when
// the key is absent. Used to support nested member lvalue assignments like
// `m[k1][k2] = v`, where intermediate containers must be auto-created.
// Panics if insertion is required and the map is readonly. Filler values are
// not type-checked against the inherent type.
func (m *Map) FillingGet(tc semtypes.Context, key string, filler FillerFactory) BalValue {
	if e, ok := m.data[key]; ok {
		return e.value
	}
	m.checkMutable()
	if filler == nil {
		panic(NewErrorWithMessage("no filler value"))
	}
	v := filler()
	m.putUnchecked(key, v)
	return v
}

// Put stores value at key. Panics if the map is readonly or value does not
// belong to the inherent member type at key.
func (m *Map) Put(tc semtypes.Context, key string, value BalValue) {
	m.checkMutable()
	m.checkMemberType(tc, key, value)
	m.putUnchecked(key, value)
}

func (m *Map) putUnchecked(key string, value BalValue) {
	if e, ok := m.data[key]; ok {
		e.value = value
		return
	}
	e := &mapEntry{key: key, value: value}
	m.data[key] = e
	m.appendEntry(e)
}

// Delete removes the entry for key. Panics if the map is readonly.
func (m *Map) Delete(tc semtypes.Context, key string) {
	m.checkMutable()
	e, ok := m.data[key]
	if !ok {
		return
	}
	m.unlinkEntry(e)
	delete(m.data, key)
}

func (m *Map) checkMutable() {
	if m.isReadonly {
		panic(NewErrorWithMessage("inherent type violation: cannot mutate readonly value"))
	}
}

func (m *Map) checkMemberType(tc semtypes.Context, key string, value BalValue) {
	memberTy := m.atomic.FieldInnerVal(key)
	valueTy := SemTypeForValue(value)
	if !semtypes.IsSubtype(tc, valueTy, memberTy) {
		panic(NewErrorWithMessage("inherent type violation"))
	}
}

func (m *Map) appendEntry(e *mapEntry) {
	if m.tail == nil {
		m.head, m.tail = e, e
		return
	}
	e.prev = m.tail
	m.tail.next = e
	m.tail = e
}

func (m *Map) unlinkEntry(e *mapEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		m.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		m.tail = e.prev
	}
	e.prev, e.next = nil, nil
}

func (m *Map) Len() int {
	return len(m.data)
}

func (m *Map) IsReadonly() bool {
	return m.isReadonly
}

func (m *Map) Keys() []string {
	keys := make([]string, 0, len(m.data))
	for e := m.head; e != nil; e = e.next {
		keys = append(keys, e.key)
	}
	return keys
}

func (m *Map) String(visited map[uintptr]bool) string {
	ptr := uintptr(unsafe.Pointer(m))
	if visited[ptr] {
		return "{...}"
	}
	visited[ptr] = true
	defer delete(visited, ptr)

	var b strings.Builder
	b.WriteByte('{')
	i := 0
	for e := m.head; e != nil; e = e.next {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%q", e.key)
		b.WriteByte(':')
		b.WriteString(toString(e.value, visited, false))
		i++
	}
	b.WriteByte('}')
	return b.String()
}
