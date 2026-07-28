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

type floatSubtype struct {
	allowed bool
	values  []enumerableFloat
}

var _ ProperSubtypeData = &floatSubtype{}

func newFloatSubtypeFromBoolEnumerableFloat(allowed bool, value enumerableFloat) floatSubtype {
	this := floatSubtype{}
	this.allowed = allowed
	this.values = []enumerableFloat{value}
	return this
}

func newFloatSubtypeFromBoolEnumerableFloats(allowed bool, values []enumerableType[float64]) floatSubtype {
	this := floatSubtype{}
	this.allowed = allowed
	var floats []enumerableFloat
	for _, value := range values {
		floats = append(floats, newEnumerableFloatFromFloat64(value.Value()))
	}
	this.values = floats
	return this
}

func FloatConst(value float64) SemType {
	return getBasicSubtype(BTFloat, newFloatSubtypeFromBoolEnumerableFloat(true, newEnumerableFloatFromFloat64(value)))
}

func floatSubtypeSingleValue(d SubtypeData) common.Optional[float64] {
	if _, ok := d.(allOrNothingSubtype); ok {
		return common.OptionalEmpty[float64]()
	}
	v := d.(floatSubtype)
	if !v.allowed {
		return common.OptionalEmpty[float64]()
	}
	if len(v.values) != 1 {
		return common.OptionalEmpty[float64]()
	}
	return common.OptionalOf(v.values[0].value)
}

func floatSubtypeContains(d SubtypeData, f enumerableFloat) bool {
	if allOrNothingSubtype, ok := d.(allOrNothingSubtype); ok {
		return allOrNothingSubtype.IsAllSubtype()
	}
	v := d.(floatSubtype)
	if slices.Contains(v.values, f) {
		return v.allowed
	}
	return (!v.allowed)
}

func createFloatSubtype(allowed bool, values []enumerableType[float64]) ProperSubtypeData {
	if len(values) == 0 {
		if allowed {
			return createNothing()
		} else {
			return createAll()
		}
	}
	return newFloatSubtypeFromBoolEnumerableFloats(allowed, values)
}

func (f *floatSubtype) Allowed() bool {
	return f.allowed
}

func (f *floatSubtype) Values() []enumerableType[float64] {
	var values []enumerableType[float64]
	for _, value := range f.values {
		values = append(values, &value)
	}
	return values
}
