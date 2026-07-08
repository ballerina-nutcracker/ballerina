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

type bddNodeSimple struct {
	_atom     atom
	canonical bddKey
}

var _ bddNode = &bddNodeSimple{}

func (b *bddNodeSimple) left() Bdd {
	return bddAll()
}

func (b *bddNodeSimple) middle() Bdd {
	return bddNothing()
}

func (b *bddNodeSimple) right() Bdd {
	return bddNothing()
}

func (b *bddNodeSimple) atom() atom {
	return b._atom
}

func newBddNodeSimple(atom atom) *bddNodeSimple {
	return &bddNodeSimple{
		_atom: atom,
		canonical: internBddNodeKey(bddNodeKey{
			atom:   atom.canonicalKey(),
			left:   bddKeyAll,
			middle: bddKeyNothing,
			right:  bddKeyNothing,
		}),
	}
}

func (b *bddNodeSimple) canonicalKey() bddKey {
	return b.canonical
}
