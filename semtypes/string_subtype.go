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
	"slices"

	"ballerina/common"
)

type stringSubtypeListCoverage struct {
	IsSubtype bool
	Indices   []int
}

type stringSubtype struct {
	charData    charStringSubtype
	nonCharData nonCharStringSubtype
}

var (
	EMPTY_STRING_ARR                   = []enumerableType[string]{}
	EMPTY_CHAR_ARR                     = []enumerableType[string]{}
	_                ProperSubtypeData = &stringSubtype{}
)

func newStringSubtypeListCoverageFromBoolInts(isSubtype bool, indices []int) stringSubtypeListCoverage {
	this := stringSubtypeListCoverage{}
	this.IsSubtype = isSubtype
	this.Indices = indices
	return this
}

func stringSubtypeListCoverageFrom(isSubtype bool, indices []int) stringSubtypeListCoverage {
	return newStringSubtypeListCoverageFromBoolInts(isSubtype, indices)
}

func newStringSubtypeFromCharStringSubtypeNonCharStringSubtype(charData charStringSubtype, nonCharData nonCharStringSubtype) stringSubtype {
	this := stringSubtype{}
	this.charData = charData
	this.nonCharData = nonCharData
	return this
}

func stringSubtypeFrom(chara charStringSubtype, nonChar nonCharStringSubtype) stringSubtype {
	return newStringSubtypeFromCharStringSubtypeNonCharStringSubtype(chara, nonChar)
}

func stringSubtypeContains(d SubtypeData, s string) bool {
	if allOrNothingSubtype, ok := d.(allOrNothingSubtype); ok {
		return allOrNothingSubtype.IsAllSubtype()
	}
	st := d.(stringSubtype)
	chara := st.charData
	nonChar := st.nonCharData
	if len(s) == 1 {
		charString := enumerableCharStringFrom(s)
		if slices.Contains(chara.Values(), charString) {
			return chara.Allowed()
		}
		return !nonChar.Allowed()
	}
	stringString := enumerableStringFrom(s)
	if slices.Contains(nonChar.Values(), stringString) {
		return nonChar.Allowed()
	}
	return !nonChar.Allowed()
}

func createStringSubtype(chara charStringSubtype, nonChar nonCharStringSubtype) SubtypeData {
	if len(chara.Values()) == 0 && len(nonChar.Values()) == 0 {
		if (!chara.allowed) && (!nonChar.allowed) {
			return createAll()
		} else if chara.allowed && nonChar.allowed {
			return createNothing()
		}
	}
	return stringSubtypeFrom(chara, nonChar)
}

func stringSubtypeSingleValue(d SubtypeData) common.Optional[string] {
	if _, ok := d.(allOrNothingSubtype); ok {
		return common.OptionalEmpty[string]()
	}
	st := d.(stringSubtype)
	chara := st.charData
	nonChar := st.nonCharData
	var charCount int
	if chara.Allowed() {
		charCount = len(chara.Values())
	} else {
		charCount = 2
	}
	var nonCharCount int
	if nonChar.Allowed() {
		nonCharCount = len(nonChar.Values())
	} else {
		nonCharCount = 2
	}
	if charCount+nonCharCount == 1 {
		if charCount != 0 {
			return common.OptionalOf(chara.Values()[0].Value())
		}
		return common.OptionalOf(nonChar.Values()[0].Value())
	}
	return common.OptionalEmpty[string]()
}

func StringConst(value string) SemType {
	var chara charStringSubtype
	var nonChar nonCharStringSubtype
	if codePointCount(value, 0, len(value)) == 1 {
		chara = charStringSubtypeFrom(true, []enumerableType[string]{enumerableCharStringFrom(value)})
		nonChar = nonCharStringSubtypeFrom(true, EMPTY_STRING_ARR)
	} else {
		chara = charStringSubtypeFrom(true, EMPTY_CHAR_ARR)
		nonChar = nonCharStringSubtypeFrom(true, []enumerableType[string]{enumerableStringFrom(value)})
	}
	return getBasicSubtype(BTString, newStringSubtypeFromCharStringSubtypeNonCharStringSubtype(chara, nonChar))
}

func codePointCount(s string, start, end int) int {
	return len([]rune(s[start:end]))
}

func (s *stringSubtype) GetChar() enumerableSubtype[string] {
	return &s.charData
}

func (s *stringSubtype) GetNonChar() enumerableSubtype[string] {
	return &s.nonCharData
}

func stringChar() SemType {
	st := newStringSubtypeFromCharStringSubtypeNonCharStringSubtype(
		charStringSubtypeFrom(false, EMPTY_CHAR_ARR),
		nonCharStringSubtypeFrom(true, EMPTY_STRING_ARR),
	)
	return getBasicSubtype(BTString, st)
}
