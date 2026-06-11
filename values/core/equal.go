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
	"unsafe"

	"ballerina-lang-go/decimal"
)

// refPair identifies an ordered pair of container references being compared.
type refPair struct {
	a, b unsafe.Pointer
}

// DeepEquals implements DeepEquals abstract operation [https://ballerina.io/spec/lang/master/#DeepEquals].
func DeepEquals(v1, v2 BalValue) bool {
	return deepEqual(v1, v2, nil)
}

func deepEqual(v1, v2 BalValue, visited map[refPair]struct{}) bool {
	if v1 == nil || v2 == nil {
		return v1 == nil && v2 == nil
	}
	switch a := v1.(type) {
	case int64:
		b, ok := v2.(int64)
		return ok && a == b
	case float64:
		b, ok := v2.(float64)
		return ok && FloatDeepEqual(a, b)
	case string:
		b, ok := v2.(string)
		return ok && a == b
	case bool:
		b, ok := v2.(bool)
		return ok && a == b
	case *decimal.Decimal:
		b, ok := v2.(*decimal.Decimal)
		return ok && a.Cmp(b) == 0
	case *List:
		b, ok := v2.(*List)
		if !ok {
			return false
		}
		return listDeepEqual(a, b, visited)
	case *Map:
		b, ok := v2.(*Map)
		if !ok {
			return false
		}
		return mapDeepEqual(a, b, visited)
	case XMLValue:
		b, ok := v2.(XMLValue)
		if !ok {
			return false
		}
		return xmlDeepEqual(a, b, visited)
	default:
		return v1 == v2
	}
}

// markVisited records the pair (a,b) as in-progress. If the same pair (in
// either order) is already being compared, equality is assumed to break
// cycles. The returned visited map is guaranteed non-nil.
func markVisited(a, b unsafe.Pointer, visited map[refPair]struct{}) (map[refPair]struct{}, bool) {
	if visited != nil {
		if _, ok := visited[refPair{a, b}]; ok {
			return visited, true
		}
		if _, ok := visited[refPair{b, a}]; ok {
			return visited, true
		}
	} else {
		visited = make(map[refPair]struct{})
	}
	visited[refPair{a, b}] = struct{}{}
	return visited, false
}

func listDeepEqual(a, b *List, visited map[refPair]struct{}) bool {
	if a == b {
		return true
	}
	if a.Len() != b.Len() {
		return false
	}
	visited, cycle := markVisited(unsafe.Pointer(a), unsafe.Pointer(b), visited)
	if cycle {
		return true
	}
	for i, n := 0, a.Len(); i < n; i++ {
		if !deepEqual(a.Get(i), b.Get(i), visited) {
			return false
		}
	}
	return true
}

func mapDeepEqual(a, b *Map, visited map[refPair]struct{}) bool {
	if a == b {
		return true
	}
	if a.Len() != b.Len() {
		return false
	}
	visited, cycle := markVisited(unsafe.Pointer(a), unsafe.Pointer(b), visited)
	if cycle {
		return true
	}
	for e := a.head; e != nil; e = e.next {
		other, ok := b.Get(e.key)
		if !ok {
			return false
		}
		if !deepEqual(e.value, other, visited) {
			return false
		}
	}
	return true
}

func xmlDeepEqual(a, b XMLValue, visited map[refPair]struct{}) bool {
	if a == b {
		return true
	}
	switch x := a.(type) {
	case *XMLText:
		y, ok := b.(*XMLText)
		return ok && x.Body == y.Body
	case *XMLProcessingInstruction:
		y, ok := b.(*XMLProcessingInstruction)
		return ok && x.Target == y.Target && x.Data == y.Data
	case *XMLComment:
		y, ok := b.(*XMLComment)
		return ok && x.Body == y.Body
	case *XMLElement:
		y, ok := b.(*XMLElement)
		return ok && xmlElementDeepEqual(x, y, visited)
	case *XMLSequence:
		y, ok := b.(*XMLSequence)
		return ok && xmlSequenceDeepEqual(x, y, visited)
	default:
		panic(fmt.Sprintf("internal error: unsupported XML value type %T", a))
	}
}

func xmlElementDeepEqual(a, b *XMLElement, visited map[refPair]struct{}) bool {
	if a == b {
		return true
	}
	visited, cycle := markVisited(unsafe.Pointer(a), unsafe.Pointer(b), visited)
	if cycle {
		return true
	}
	return a.Name == b.Name &&
		mapDeepEqual(a.Attributes, b.Attributes, visited) &&
		mapDeepEqual(a.Namespaces, b.Namespaces, visited) &&
		xmlDeepEqual(a.Children, b.Children, visited)
}

func xmlSequenceDeepEqual(a, b *XMLSequence, visited map[refPair]struct{}) bool {
	if a == b {
		return true
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	visited, cycle := markVisited(unsafe.Pointer(a), unsafe.Pointer(b), visited)
	if cycle {
		return true
	}
	for i := range a.Children {
		if !xmlDeepEqual(a.Children[i], b.Children[i], visited) {
			return false
		}
	}
	return true
}
