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

// Port of jBallerina's QualifiedName, its semantic and syntactic variants, and QualifiedNameMap.

package native

import "slices"

// nsNotDefined marks a name whose record field or type has no @Namespace annotation; it matches any
// namespace.
const nsNotDefined = "$$ns_annot_not_defined$$"

type attributeState int

const (
	stateNotDefined attributeState = iota
	stateAttribute
	stateElement
)

// qname is a qualified XML name. A semantic name compares the namespace URI and local part; a syntactic
// name also compares the prefix.
type qname struct {
	ns       string
	local    string
	prefix   string
	state    attributeState
	semantic bool
}

func newQName(ns, local, prefix string, semantic bool) qname {
	return qname{ns: ns, local: local, prefix: prefix, semantic: semantic}
}

func newQNameWithState(ns, local, prefix string, state attributeState, semantic bool) qname {
	return qname{ns: ns, local: local, prefix: prefix, state: state, semantic: semantic}
}

func (q qname) key() string {
	if q.semantic {
		return q.ns + "\x00" + q.local + "\x00" + string(rune('0'+q.state))
	}
	return q.prefix + "\x00" + q.ns + "\x00" + q.local + "\x00" + string(rune('0'+q.state))
}

func isSameNamespace(q1, q2 qname) bool {
	if q1.ns == nsNotDefined || q2.ns == nsNotDefined {
		return true
	}
	if q1.semantic && q2.semantic {
		return q1.ns == q2.ns
	}
	return q1.ns == q2.ns && q1.prefix == q2.prefix
}

func isSameAttributeFlag(s1, s2 attributeState) bool {
	return s1 == stateNotDefined || s2 == stateNotDefined || s1 == s2
}

// isEqualQName is DataUtils.isEqualQualifiedName.
func isEqualQName(q1, q2 *qname) bool {
	if q1 == nil || q2 == nil {
		return false
	}
	if q1.key() == q2.key() {
		return true
	}
	return q1.local == q2.local && (q1.ns == nsNotDefined || q2.ns == nsNotDefined)
}

type qnameEntry[V any] struct {
	name  qname
	value V
}

// qnameMap is QualifiedNameMap: an exact map from names plus an index by local part that falls back to
// namespace-tolerant matching. Like the Java class, an exact removal leaves the name in the local-part index.
type qnameMap[V any] struct {
	keys    []string
	members map[string]qnameEntry[V]
	byLocal map[string][]qname
}

func newQNameMap[V any]() *qnameMap[V] {
	return &qnameMap[V]{members: make(map[string]qnameEntry[V]), byLocal: make(map[string][]qname)}
}

func (m *qnameMap[V]) removeMember(name qname) (V, bool) {
	k := name.key()
	entry, ok := m.members[k]
	if ok {
		delete(m.members, k)
		m.keys = slices.DeleteFunc(m.keys, func(existing string) bool { return existing == k })
	}
	return entry.value, ok
}

func (m *qnameMap[V]) remove(name qname) (V, bool) {
	if value, ok := m.removeMember(name); ok {
		return value, true
	}
	var zero V
	names, ok := m.byLocal[name.local]
	if !ok {
		return zero, false
	}
	for i, candidate := range names {
		if isSameNamespace(candidate, name) && isSameAttributeFlag(candidate.state, name.state) {
			value, found := m.removeMember(candidate)
			names = slices.Delete(names, i, i+1)
			m.updateByLocal(name.local, names)
			return value, found
		}
	}
	return zero, false
}

func (m *qnameMap[V]) updateByLocal(local string, names []qname) {
	if len(names) == 0 {
		delete(m.byLocal, local)
		return
	}
	m.byLocal[local] = names
}

func (m *qnameMap[V]) contains(name qname) bool {
	if _, ok := m.members[name.key()]; ok {
		return true
	}
	for _, candidate := range m.byLocal[name.local] {
		if isSameNamespace(candidate, name) && isSameAttributeFlag(candidate.state, name.state) {
			return true
		}
	}
	return false
}

func (m *qnameMap[V]) containsLocal(local string) bool {
	_, ok := m.byLocal[local]
	return ok
}

func (m *qnameMap[V]) put(name qname, value V) {
	k := name.key()
	if _, ok := m.members[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.members[k] = qnameEntry[V]{name: name, value: value}
	m.byLocal[name.local] = append(m.byLocal[name.local], name)
}

func (m *qnameMap[V]) get(name qname) (V, bool) {
	if entry, ok := m.members[name.key()]; ok {
		return entry.value, true
	}
	var zero V
	for _, candidate := range m.byLocal[name.local] {
		if isSameNamespace(candidate, name) {
			entry, ok := m.members[candidate.key()]
			return entry.value, ok
		}
	}
	return zero, false
}

func (m *qnameMap[V]) getOrDefault(name qname, defaultValue V) V {
	if value, ok := m.get(name); ok {
		return value
	}
	return defaultValue
}

func (m *qnameMap[V]) isEmpty() bool {
	return len(m.members) == 0
}

func (m *qnameMap[V]) matchedName(name qname) (qname, bool) {
	if _, ok := m.members[name.key()]; ok {
		return name, true
	}
	for _, candidate := range m.byLocal[name.local] {
		if isSameNamespace(candidate, name) && isSameAttributeFlag(candidate.state, name.state) {
			return candidate, true
		}
	}
	return qname{}, false
}

func (m *qnameMap[V]) entries() []qnameEntry[V] {
	result := make([]qnameEntry[V], 0, len(m.keys))
	for _, k := range m.keys {
		result = append(result, m.members[k])
	}
	return result
}

func (m *qnameMap[V]) lastName() (qname, bool) {
	if len(m.keys) == 0 {
		return qname{}, false
	}
	return m.members[m.keys[len(m.keys)-1]].name, true
}

func (m *qnameMap[V]) clear() {
	m.keys = nil
	m.members = make(map[string]qnameEntry[V])
	m.byLocal = make(map[string][]qname)
}
