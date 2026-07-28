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
	"fmt"
	"strings"

	"ballerina/common"
)

type intSubtype struct {
	Ranges []intRange
}

var _ ProperSubtypeData = &intSubtype{}

func newIntSubtypeFromRanges(ranges []intRange) intSubtype {
	this := intSubtype{}
	this.Ranges = ranges
	return this
}

func createIntSubtype(ranges ...intRange) intSubtype {
	return newIntSubtypeFromRanges(ranges)
}

func createSingleRangeSubtype(min, max int64) intSubtype {
	return newIntSubtypeFromRanges([]intRange{rangeFrom(min, max)})
}

func IntConst(value int64) SemType {
	return getBasicSubtype(BTInt, createSingleRangeSubtype(value, value))
}

func validIntWidth(signed bool, bits int64) {
	if bits <= 0 {
		var message string
		if bits == 0 {
			message = "zero"
		} else {
			message = "negative"
		}
		message = message + " width in bits"
		panic(message)
	}
	if signed {
		if bits > 64 {
			panic("width of signed integers limited to 64")
		}
	} else {
		if bits > 63 {
			panic("width of unsigned integers limited to 63")
		}
	}
}

func validIntWidthSigned(bits int) {
	validIntWidth(true, int64(bits))
}

func validIntWidthUnsigned(bits int) {
	validIntWidth(false, int64(bits))
}

func intWidthSigned(bits int64) SemType {
	validIntWidth(true, bits)
	if bits == 64 {
		return INT
	}
	t := createSingleRangeSubtype((-(int64(1) << (bits - int64(1)))), ((int64(1) << (bits - int64(1))) - int64(1)))
	return getBasicSubtype(BTInt, t)
}

func intWidthUnsigned(bits int) SemType {
	validIntWidth(false, int64(bits))
	t := createSingleRangeSubtype(int64(0), ((int64(1) << bits) - int64(1)))
	return getBasicSubtype(BTInt, t)
}

func intSubtypeWidenUnsigned(d SubtypeData) SubtypeData {
	if _, ok := d.(allOrNothingSubtype); ok {
		return d
	}
	v := d.(intSubtype)
	if v.Ranges[0].Min < int64(0) {
		return createAll()
	}
	r := v.Ranges[len(v.Ranges)-1]
	i := int64(8)
	for i <= int64(32) {
		if r.Max < (int64(1) << i) {
			w := createSingleRangeSubtype(int64(0), ((int64(1) << i) - 1))
			return w
		}
		i = (i * 2)
	}
	return createAll()
}

func intSubtypeSingleValue(d SubtypeData) common.Optional[int64] {
	if _, ok := d.(allOrNothingSubtype); ok {
		return common.OptionalEmpty[int64]()
	}
	v := d.(intSubtype)
	if len(v.Ranges) != 1 {
		return common.OptionalEmpty[int64]()
	}
	min := v.Ranges[0].Min
	if min != v.Ranges[0].Max {
		return common.OptionalEmpty[int64]()
	}
	return common.OptionalOf(min)
}

func (i intSubtype) String() string {
	var builder strings.Builder
	builder.WriteString("(int")
	for _, r := range i.Ranges {
		builder.WriteString(" ")
		if r.Min == r.Max {
			fmt.Fprintf(&builder, "%d", r.Min)
		} else {
			fmt.Fprintf(&builder, "%d..%d", r.Min, r.Max)
		}
	}
	builder.WriteString(")")
	return builder.String()
}

func intSubtypeContains(d SubtypeData, n int64) bool {
	if allOrNothingSubtype, ok := d.(allOrNothingSubtype); ok {
		return allOrNothingSubtype.IsAllSubtype()
	}
	v := d.(intSubtype)
	for _, r := range v.Ranges {
		if (r.Min <= n) && (n <= r.Max) {
			return true
		}
	}
	return false
}
