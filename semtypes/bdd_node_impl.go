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

type bddNodeImpl struct {
	_atom     atom
	_left     Bdd
	_middle   Bdd
	_right    Bdd
	canonical bddKey
}

var _ bddNode = &bddNodeImpl{}

func (b *bddNodeImpl) atom() atom {
	return b._atom
}

func (b *bddNodeImpl) left() Bdd {
	return b._left
}

func (b *bddNodeImpl) middle() Bdd {
	return b._middle
}

func (b *bddNodeImpl) right() Bdd {
	return b._right
}

func newBddNodeImpl(atom atom, left, middle, right Bdd) *bddNodeImpl {
	return &bddNodeImpl{
		_atom:   atom,
		_left:   left,
		_middle: middle,
		_right:  right,
		canonical: internBddNodeKey(bddNodeKey{
			atom:   atom.canonicalKey(),
			left:   left.canonicalKey(),
			middle: middle.canonicalKey(),
			right:  right.canonicalKey(),
		}),
	}
}

func (b *bddNodeImpl) canonicalKey() bddKey {
	return b.canonical
}
