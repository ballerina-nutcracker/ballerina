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
	"iter"

	"ballerina-lang-go/common"
)

type fieldPair struct {
	Name   string
	Type1  SemType
	Type2  SemType
	Index1 int
	Index2 int
}

func createFieldPair(name string, type1 SemType, type2 SemType, index1 int, index2 int) fieldPair {

	return fieldPair{
		Name:   name,
		Type1:  type1,
		Type2:  type2,
		Index1: index1,
		Index2: index2,
	}
}

type mappingPairIterator struct {
	names1          []string
	names2          []string
	types1          []SemType
	types2          []SemType
	len1            int
	len2            int
	i1              int
	i2              int
	rest1           SemType
	rest2           SemType
	doneIteration   bool
	shouldCalculate bool
	cache           *fieldPair
}

func (i *mappingPairIterator) hasNext() bool {
	if i.doneIteration {
		return false
	}
	if i.shouldCalculate {
		cache := i.internalNext()
		if cache == nil {
			i.doneIteration = true
		}
		i.cache = cache
		i.shouldCalculate = false
	}
	return !i.doneIteration
}

func (i *mappingPairIterator) next() fieldPair {
	if i.doneIteration {
		panic("Exhausted iterator")
	}
	if i.shouldCalculate {
		cache := i.internalNext()
		if cache == nil {
			panic("unexpected nil cache")
		}
		i.cache = cache
	}
	i.shouldCalculate = true
	return *i.cache
}

func (i *mappingPairIterator) internalNext() *fieldPair {
	var p *fieldPair
	if i.i1 >= i.len1 {
		if i.i2 >= i.len2 {
			return nil
		}
		p = new(createFieldPair(i.curName2(), i.rest1, i.curType2(), -1, i.i2))
		i.i2++
	} else if i.i2 >= i.len2 {
		p = new(createFieldPair(i.curName1(), i.curType1(), i.rest2, i.i1, -1))
		i.i1++
	} else {
		name1 := i.curName1()
		name2 := i.curName2()
		if codePointCompare(name1, name2) {
			p = new(createFieldPair(name1, i.curType1(), i.rest2, i.i1, -1))
			i.i1++
		} else if codePointCompare(name2, name1) {
			p = new(createFieldPair(name2, i.rest1, i.curType2(), -1, i.i2))
			i.i2++
		} else {
			p = new(createFieldPair(name1, i.curType1(), i.curType2(), i.i1, i.i2))
			i.i1++
			i.i2++
		}
	}
	return p
}

func (i *mappingPairIterator) curType1() SemType {
	return i.types1[i.i1]
}

func (i *mappingPairIterator) curName1() string {
	return i.names1[i.i1]
}

func (i *mappingPairIterator) curType2() SemType {
	return i.types2[i.i2]
}

func (i *mappingPairIterator) curName2() string {
	return i.names2[i.i2]
}

func (i *mappingPairIterator) reset() {
	i.i1 = 0
	i.i2 = 0
}

func (i *mappingPairIterator) index1(name string) common.Optional[int] {
	i1Prev := i.i1 - 1
	if i1Prev >= 0 && i.names1[i1Prev] == name {
		return common.OptionalOf(i1Prev)
	}
	return common.OptionalEmpty[int]()
}

func (i *mappingPairIterator) toIterator() iter.Seq[fieldPair] {
	return func(yield func(fieldPair) bool) {
		for i.hasNext() {
			if !yield(i.next()) {
				break
			}
		}
	}
}

func newFieldPairs(m1 *MappingAtomicType, m2 *MappingAtomicType) iter.Seq[fieldPair] {
	i := &mappingPairIterator{
		names1:          m1.Names,
		names2:          m2.Names,
		types1:          m1.Types,
		types2:          m2.Types,
		len1:            len(m1.Names),
		len2:            len(m2.Names),
		rest1:           m1.Rest,
		rest2:           m2.Rest,
		shouldCalculate: true,
	}
	return i.toIterator()
}
