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

type intOps struct{}

var _ basicTypeOps = &intOps{}

func newIntOps() intOps {
	this := intOps{}
	return this
}

var intOpsInstance = newIntOps()

func (i *intOps) Union(d1 subtypeData, d2 subtypeData) subtypeData {
	v1 := d1.(intSubtype)
	v2 := d2.(intSubtype)
	v := rangeListUnion(v1.Ranges, v2.Ranges)
	if len(v) == 1 && v[0].Min == minValue && v[0].Max == maxValue {
		return createAll()
	}
	return createIntSubtype(v...)
}

func (i *intOps) Intersect(d1 subtypeData, d2 subtypeData) subtypeData {
	v1 := d1.(intSubtype)
	v2 := d2.(intSubtype)
	v := rangeListIntersect(v1.Ranges, v2.Ranges)
	if len(v) == 0 {
		return createNothing()
	}
	return createIntSubtype(v...)
}

func (i *intOps) Diff(d1 subtypeData, d2 subtypeData) subtypeData {
	v1 := d1.(intSubtype)
	v2 := d2.(intSubtype)
	v := rangeListIntersect(v1.Ranges, rangeListComplement(v2.Ranges))
	if len(v) == 0 {
		return createNothing()
	}
	return createIntSubtype(v...)
}

func (i *intOps) complement(d subtypeData) subtypeData {
	v := d.(intSubtype)
	return createIntSubtype(rangeListComplement(v.Ranges)...)
}

func intSubtypeOverlapRange(subtype intSubtype, r intRange) bool {
	subtypeData := intOpsInstance.Intersect(subtype, createIntSubtype(r))
	if allOrNothingSubtype, ok := subtypeData.(allOrNothingSubtype); ok {
		return !allOrNothingSubtype.IsNothingSubtype()
	}
	return true
}

func intSubtypeMax(subtype intSubtype) int64 {
	return subtype.Ranges[len(subtype.Ranges)-1].Max
}

func intSubtypeMin(subtype intSubtype) int64 {
	return subtype.Ranges[0].Min
}

func (i *intOps) IsEmpty(cx Context, t subtypeData) bool {
	return notIsEmpty(cx, t)
}

func rangeListUnion(v1 []intRange, v2 []intRange) []intRange {
	var result []intRange
	i1 := 0
	i2 := 0
	len1 := len(v1)
	len2 := len(v2)

	for {
		if i1 >= len1 {
			if i2 >= len2 {
				break
			}
			result = rangeUnionPush(result, v2[i2])
			i2 += 1
		} else if i2 >= len2 {
			result = rangeUnionPush(result, v1[i1])
			i1 += 1
		} else {
			r1 := v1[i1]
			r2 := v2[i2]
			combined := getRangeUnion(r1, r2)
			if combined.Status == 0 {
				result = rangeUnionPush(result, *combined.Range)
				i1 += 1
				i2 += 1
			} else if combined.Status < 0 {
				result = rangeUnionPush(result, r1)
				i1 += 1
			} else {
				result = rangeUnionPush(result, r2)
				i2 += 1
			}
		}
	}
	return result
}

func rangeUnionPush(ranges []intRange, next intRange) []intRange {
	lastIndex := len(ranges) - 1
	if lastIndex < 0 {
		return append(ranges, next)
	}
	combined := getRangeUnion(ranges[lastIndex], next)
	if combined.Status == 0 {
		ranges[lastIndex] = *combined.Range
		return ranges
	}
	return append(ranges, next)
}

func getRangeUnion(r1 intRange, r2 intRange) rangeUnion {
	if r1.Max < r2.Min {
		if r1.Max+1 != r2.Min {
			return rangeUnionFrom(-1)
		}
	}
	if r2.Max < r1.Min {
		if r2.Max+1 != r1.Min {
			return rangeUnionFrom(1)
		}
	}
	return fromWithRange(rangeFrom(min(r1.Min, r2.Min), max(r1.Max, r2.Max)))
}

func rangeListIntersect(v1 []intRange, v2 []intRange) []intRange {
	var result []intRange
	i1 := 0
	i2 := 0
	len1 := len(v1)
	len2 := len(v2)
	for {
		if i1 >= len1 || i2 >= len2 {
			break
		} else {
			r1 := v1[i1]
			r2 := v2[i2]
			combined := rangeIntersect(r1, r2)
			if combined.Status == 0 {
				result = append(result, *combined.Range)
				i1 += 1
				i2 += 1
			} else if combined.Status < 0 {
				i1 += 1
			} else {
				i2 += 1
			}
		}
	}
	return result
}

func rangeIntersect(r1 intRange, r2 intRange) rangeUnion {
	if r1.Max < r2.Min {
		return rangeUnionFrom(-1)
	}
	if r2.Max < r1.Min {
		return rangeUnionFrom(1)
	}
	return fromWithRange(rangeFrom(max(r1.Min, r2.Min), min(r1.Max, r2.Max)))
}

func rangeListComplement(v []intRange) []intRange {
	var result []intRange
	length := len(v)
	minVal := v[0].Min
	if minVal > minValue {
		result = append(result, rangeFrom(minValue, minVal-1))
	}
	for i := 1; i < length; i++ {
		result = append(result, rangeFrom(v[i-1].Max+1, v[i].Min-1))
	}
	maxVal := v[len(v)-1].Max
	if maxVal < maxValue {
		result = append(result, rangeFrom(maxVal+1, maxValue))
	}
	return result
}
