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

type stringOps struct {
}

var _ basicTypeOps = &stringOps{}

func newStringOps() stringOps {
	this := stringOps{}
	return this
}

func (s *stringOps) Union(d1 subtypeData, d2 subtypeData) subtypeData {
	sd1 := d1.(stringSubtype)
	sd2 := d2.(stringSubtype)
	var chars []enumerableType[string]
	var nonChars []enumerableType[string]
	charsAllowed := enumerableSubtypeUnion(sd1.GetChar(), sd2.GetChar(), &chars)
	nonCharsAllowed := enumerableSubtypeUnion(sd1.GetNonChar(), sd2.GetNonChar(), &nonChars)
	return createStringSubtype(charStringSubtypeFrom(charsAllowed, chars), nonCharStringSubtypeFrom(nonCharsAllowed, nonChars))
}

func (s *stringOps) Intersect(d1 subtypeData, d2 subtypeData) subtypeData {
	if allOrNothing1, ok := d1.(*allOrNothingSubtype); ok {
		if allOrNothing1.IsAllSubtype() {
			return d2
		} else {
			return createNothing()
		}
	}
	if allOrNothing2, ok := d2.(*allOrNothingSubtype); ok {
		if allOrNothing2.IsAllSubtype() {
			return d1
		} else {
			return createNothing()
		}
	}
	sd1 := d1.(stringSubtype)
	sd2 := d2.(stringSubtype)
	var chars []enumerableType[string]
	var nonChars []enumerableType[string]
	charsAllowed := enumerableSubtypeIntersect(sd1.GetChar(), sd2.GetChar(), &chars)
	nonCharsAllowed := enumerableSubtypeIntersect(sd1.GetNonChar(), sd2.GetNonChar(), &nonChars)
	return createStringSubtype(charStringSubtypeFrom(charsAllowed, chars), nonCharStringSubtypeFrom(nonCharsAllowed, nonChars))
}

func (s *stringOps) Diff(d1 subtypeData, d2 subtypeData) subtypeData {
	return s.Intersect(d1, s.complement(d2))
}

func (s *stringOps) complement(d subtypeData) subtypeData {
	st := d.(stringSubtype)
	if len(st.GetChar().Values()) == 0 && len(st.GetNonChar().Values()) == 0 {
		if st.GetChar().Allowed() && st.GetNonChar().Allowed() {
			return createAll()
		} else if !st.GetChar().Allowed() && !st.GetNonChar().Allowed() {
			return createNothing()
		}
	}
	return createStringSubtype(charStringSubtypeFrom(!st.GetChar().Allowed(), st.GetChar().Values()), nonCharStringSubtypeFrom(!st.GetNonChar().Allowed(), st.GetNonChar().Values()))
}

func (s *stringOps) IsEmpty(cx Context, t subtypeData) bool {
	return notIsEmpty(cx, t)
}

func getStringSubtypeListCoverage(subtype stringSubtype, values []string) stringSubtypeListCoverage {
	var indices []int
	ch := subtype.GetChar()
	nonChar := subtype.GetNonChar()
	stringConsts := 0
	if ch.Allowed() {
		stringListIntersect(values, toStringArray(ch.Values()), &indices)
		stringConsts = len(ch.Values())
	} else if len(ch.Values()) == 0 {
		for i := range values {
			if len(values[i]) == 1 {
				indices = append(indices, i)
			}
		}
	}
	if nonChar.Allowed() {
		stringListIntersect(values, toStringArray(nonChar.Values()), &indices)
		stringConsts += len(nonChar.Values())
	} else if len(nonChar.Values()) == 0 {
		for i := range values {
			if len(values[i]) != 1 {
				indices = append(indices, i)
			}
		}
	}
	return stringSubtypeListCoverageFrom(stringConsts == len(indices), indices)
}

func toStringArray(ar []enumerableType[string]) []string {
	strings := make([]string, len(ar))
	for i, value := range ar {
		strings[i] = value.Value()
	}
	return strings
}

func stringListIntersect(values []string, target []string, indices *[]int) {
	i1 := 0
	i2 := 0
	len1 := len(values)
	len2 := len(target)
	for {
		if i1 >= len1 || i2 >= len2 {
			break
		} else {
			comp := compareEnumerable(enumerableStringFrom(values[i1]), enumerableStringFrom(target[i2]))
			switch comp {
			case eq:
				*indices = append(*indices, i1)
				i1 = i1 + 1
				i2 = i2 + 1
			case lt:
				i1 = i1 + 1
			case gt:
				i2 = i2 + 1
			default:
				panic("Invalid comparison value!")
			}
		}
	}
}
