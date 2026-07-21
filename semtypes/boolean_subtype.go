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
	"ballerina/common"
)

type booleanSubtype struct {
	Value bool
}

var _ ProperSubtypeData = &booleanSubtype{}

func newBooleanSubtypeFromBool(value bool) booleanSubtype {
	this := booleanSubtype{}
	this.Value = value
	return this
}

func booleanSubtypeFrom(value bool) booleanSubtype {
	return newBooleanSubtypeFromBool(value)
}

func booleanSubtypeContains(d SubtypeData, b bool) bool {
	if allOrNothingSubtype, ok := d.(allOrNothingSubtype); ok {
		return allOrNothingSubtype.IsAllSubtype()
	}
	r := d.(booleanSubtype)
	return r.Value == b
}

func BooleanConst(value bool) SemType {
	t := booleanSubtypeFrom(value)
	return getBasicSubtype(BTBoolean, t)
}

func booleanSubtypeSingleValue(d SubtypeData) common.Optional[bool] {
	if _, ok := d.(allOrNothingSubtype); ok {
		return common.OptionalEmpty[bool]()
	}
	b := d.(booleanSubtype)
	value := b.Value
	return common.OptionalOf(value)
}
