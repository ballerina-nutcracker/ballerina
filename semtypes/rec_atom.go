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

import "fmt"

type recAtom struct {
	idx  int
	kind recAtomKind
}

// ZERO is the shared "fully readonly" sentinel recursive atom. It is used
// interchangeably as the readonly placeholder for both list and mapping BDDs
// (see BDD_SUBTYPE_RO), so its kind is not meaningful; the to-string cycle
// detector always special-cases index 0 before consulting kind or the
// visited set.
var ZERO = newRecAtomFromInt(BDD_REC_ATOM_READONLY, recAtomKindList)
var _ atom = &recAtom{}

func newRecAtomFromInt(index int, kind recAtomKind) recAtom {
	this := recAtom{}

	this.idx = index
	this.kind = kind
	return this
}

func createRecAtom(index int, kind recAtomKind) recAtom {
	if index == BDD_REC_ATOM_READONLY {
		return ZERO
	}
	return newRecAtomFromInt(index, kind)
}

func createXMLRecAtom(index int) recAtom {
	return newRecAtomFromInt(index, recAtomKindXML)
}

func createDistinctRecAtom(index int) recAtom {
	return newRecAtomFromInt(index, recAtomKindDistinct)
}

func isDistinctRecAtom(atom atom) bool {
	rec, ok := atom.(*recAtom)
	return ok && rec.index() < 0
}

func (r *recAtom) index() int {
	return r.idx
}

func (r *recAtom) canonicalKey() atomKey {
	return recAtomKey(r.idx, r.kind)
}

func (r *recAtom) String() string {
	return fmt.Sprintf("r%d", r.idx)
}
