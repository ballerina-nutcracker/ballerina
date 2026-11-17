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

import "math"

type enumerableSubtype[T any] interface {
	Allowed() bool
	Values() []enumerableType[T]
}

var (
	lt = (-1)
	eq = 0
	gt = 1
)

// The union of two subtypes is the complement of the intersection of their complements.
func enumerableSubtypeUnion[T any](t1 enumerableSubtype[T], t2 enumerableSubtype[T], result *[]enumerableType[T]) bool {
	return !enumerableValuesIntersect(!t1.Allowed(), t1.Values(), !t2.Allowed(), t2.Values(), result)
}

func enumerableSubtypeIntersect[T any](t1 enumerableSubtype[T], t2 enumerableSubtype[T], result *[]enumerableType[T]) bool {
	return enumerableValuesIntersect(t1.Allowed(), t1.Values(), t2.Allowed(), t2.Values(), result)
}

func enumerableValuesIntersect[T any](allowed1 bool, values1 []enumerableType[T], allowed2 bool, values2 []enumerableType[T], result *[]enumerableType[T]) bool {
	switch {
	case allowed1 && allowed2:
		enumerableListIntersect(values1, values2, result)
		return true
	case !allowed1 && !allowed2:
		enumerableListUnion(values1, values2, result)
		return false
	case allowed1:
		enumerableListDiff(values1, values2, result)
		return true
	default:
		enumerableListDiff(values2, values1, result)
		return true
	}
}

func enumerableListUnion[T any](v1 []enumerableType[T], v2 []enumerableType[T], result *[]enumerableType[T]) {
	i1 := 0
	i2 := 0
	len1 := len(v1)
	len2 := len(v2)
	for {
		if i1 >= len1 {
			if i2 >= len2 {
				break
			}
			*result = append(*result, v2[i2])
			i2 = (i2 + 1)
		} else if i2 >= len2 {
			*result = append(*result, v1[i1])
			i1 = (i1 + 1)
		} else {
			s1 := v1[i1]
			s2 := v2[i2]
			switch compareEnumerable(s1, s2) {
			case eq:
				*result = append(*result, s1)
				i1 = (i1 + 1)
				i2 = (i2 + 1)
			case lt:
				*result = append(*result, s1)
				i1 = (i1 + 1)
			case gt:
				*result = append(*result, s2)
				i2 = (i2 + 1)
			}
		}
	}
}

func enumerableListIntersect[T any](v1 []enumerableType[T], v2 []enumerableType[T], result *[]enumerableType[T]) {
	i1 := 0
	i2 := 0
	len1 := len(v1)
	len2 := len(v2)
	for {
		if (i1 >= len1) || (i2 >= len2) {
			break
		} else {
			s1 := v1[i1]
			s2 := v2[i2]
			switch compareEnumerable(s1, s2) {
			case eq:
				*result = append(*result, s1)
				i1 = (i1 + 1)
				i2 = (i2 + 1)
			case lt:
				i1 = (i1 + 1)
			case gt:
				i2 = (i2 + 1)
			}
		}
	}
}

func enumerableListDiff[T any](v1 []enumerableType[T], v2 []enumerableType[T], result *[]enumerableType[T]) {
	i1 := 0
	i2 := 0
	len1 := len(v1)
	len2 := len(v2)
	for i1 < len1 {
		if i2 >= len2 {
			*result = append(*result, v1[i1])
			i1 = (i1 + 1)
		} else {
			s1 := v1[i1]
			s2 := v2[i2]
			switch compareEnumerable(s1, s2) {
			case eq:
				i1 = (i1 + 1)
				i2 = (i2 + 1)
			case lt:
				*result = append(*result, s1)
				i1 = (i1 + 1)
			case gt:
				i2 = (i2 + 1)
			}
		}
	}
}

//shallow:allow shared generic comparison adapter keeps enumerable operations consistent
func compareEnumerable[T any](v1 enumerableType[T], v2 enumerableType[T]) int {
	return v1.Compare(v2)
}

func bFloatEq(f1 float64, f2 float64) bool {
	if math.IsNaN(f1) {
		return math.IsNaN(f2)
	}
	return (f1 == f2)
}
