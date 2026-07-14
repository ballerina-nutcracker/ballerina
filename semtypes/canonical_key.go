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

import "sync"

type atomKey struct {
	index int
	gen   uint64
	rec   bool
}

type bddKey int

const (
	bddKeyNothing bddKey = iota + 1
	bddKeyAll
)

type bddNodeKey struct {
	atom   atomKey
	left   bddKey
	middle bddKey
	right  bddKey
}

// This is what allows us the flatten the whole BDD tree in not a flat struct
// We can prove the correctness assuming atomKey is a canonical representation and Bdd's them self
// are immutable. We give each Bdd node at construction a unique id. That means by definition lowest nodes should
// get a key before parents. Then by induction we can prove this is uniquely capturing the tree structure
// (assuming we don't overflow)
var bddKeyInterner = struct {
	sync.Mutex
	next bddKey
	keys map[bddNodeKey]bddKey
}{
	next: bddKeyAll + 1,
	keys: make(map[bddNodeKey]bddKey),
}

func typeAtomKey(index int, gen uint64) atomKey {
	return atomKey{index: index, gen: gen}
}

func recAtomKey(index int) atomKey {
	return atomKey{index: index, rec: true}
}

func internBddNodeKey(key bddNodeKey) bddKey {
	bddKeyInterner.Lock()
	defer bddKeyInterner.Unlock()
	if existing, ok := bddKeyInterner.keys[key]; ok {
		return existing
	}
	result := bddKeyInterner.next
	bddKeyInterner.next++
	bddKeyInterner.keys[key] = result
	return result
}
