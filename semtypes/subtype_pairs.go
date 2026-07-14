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
	"math/bits"
)

type iterState byte

const (
	iterNeedsCalc iterState = iota
	iterHasCache
	iterDone
)

type subtypePairIterator struct {
	cache subtypePair
	list1 []ProperSubtypeData
	list2 []ProperSubtypeData
	some1 basicTypeBitSet
	some2 basicTypeBitSet
	bits  basicTypeBitSet
	state iterState
}

func (i *subtypePairIterator) include(code BasicTypeCode) bool {
	return (i.bits & (1 << code.Code())) != 0
}

func (i *subtypePairIterator) hasNext() bool {
	if i.state == iterNeedsCalc {
		cache, ok := i.internalNext()
		if ok {
			i.cache = cache
			i.state = iterHasCache
		} else {
			i.state = iterDone
		}
	}
	return i.state != iterDone
}

func (i *subtypePairIterator) next() subtypePair {
	if i.state == iterNeedsCalc {
		i.cache, _ = i.internalNext()
	}
	i.state = iterNeedsCalc
	return i.cache
}

func (i *subtypePairIterator) advance1() (BasicTypeCode, SubtypeData) {
	code := basicTypeCodeFrom(bits.TrailingZeros(uint(i.some1)))
	data := i.list1[0]
	i.list1 = i.list1[1:]
	i.some1 &^= 1 << code.Code()
	return code, data
}

func (i *subtypePairIterator) advance2() (BasicTypeCode, SubtypeData) {
	code := basicTypeCodeFrom(bits.TrailingZeros(uint(i.some2)))
	data := i.list2[0]
	i.list2 = i.list2[1:]
	i.some2 &^= 1 << code.Code()
	return code, data
}

func (i *subtypePairIterator) internalNext() (subtypePair, bool) {
	for {
		has1 := i.some1 != 0
		has2 := i.some2 != 0
		if !has1 && !has2 {
			return subtypePair{}, false
		}
		if !has1 {
			code, data2 := i.advance2()
			if i.include(code) {
				return createSubTypePair(code, nil, data2), true
			}
		} else if !has2 {
			code, data1 := i.advance1()
			if i.include(code) {
				return createSubTypePair(code, data1, nil), true
			}
		} else {
			code1 := basicTypeCodeFrom(bits.TrailingZeros(uint(i.some1)))
			code2 := basicTypeCodeFrom(bits.TrailingZeros(uint(i.some2)))
			if code1 == code2 {
				_, data1 := i.advance1()
				_, data2 := i.advance2()
				if i.include(code1) {
					return createSubTypePair(code1, data1, data2), true
				}
			} else if code1.Code() < code2.Code() {
				_, data1 := i.advance1()
				if i.include(code1) {
					return createSubTypePair(code1, data1, nil), true
				}
			} else {
				_, data2 := i.advance2()
				if i.include(code2) {
					return createSubTypePair(code2, nil, data2), true
				}
			}
		}
	}
}

func newSubtypePairs(s1, s2 SemType, b basicTypeBitSet) subtypePairIterator {
	it := subtypePairIterator{bits: b, state: iterNeedsCalc}
	if s1.some() != 0 {
		it.list1 = s1.subtypeDataList()
		it.some1 = s1.some()
	}
	if s2.some() != 0 {
		it.list2 = s2.subtypeDataList()
		it.some2 = s2.some()
	}
	return it
}
