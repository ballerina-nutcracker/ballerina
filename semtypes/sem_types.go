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
	"strings"

	"ballerina-lang-go/decimal"
)

var SINT8 = intWidthSigned(8)
var SINT16 = intWidthSigned(16)
var SINT32 = intWidthSigned(32)
var UINT8 = BYTE
var UINT16 = intWidthUnsigned(16)
var UINT32 = intWidthUnsigned(32)
var CHAR = STRING_CHAR

func decimalConstFromStringValue(value string) SemType {
	if strings.Contains(value, "d") || strings.Contains(value, "D") {
		value = value[:len(value)-1]
	}
	d, err := decimal.FromString(value)
	if err != nil {
		panic("failed to parse decimal literal: " + err.Error())
	}
	return DecimalConst(*d)
}

func unionWithSemTypeSemTypesSemType(first SemType, second SemType, rest ...SemType) SemType {
	u := Union(first, second)
	for _, s := range rest {
		u = Union(u, s)
	}
	return u
}

func intersectWithSemTypeSemTypesSemType(first SemType, second SemType, rest ...SemType) SemType {
	i := Intersect(first, second)
	for _, s := range rest {
		i = Intersect(i, s)
	}
	return i
}

func isSubtypeSimpleNotNever(t1 SemType, t2 SemType) bool {
	return ((!IsNever(t1)) && IsSubtypeSimple(t1, t2))
}

func ContainsBasicType(t1 SemType, t2 SemType) bool {
	return ((widenToBasicTypeBits(t1) & t2.all()) != 0)
}

func containsType(context Context, ty SemType, typeToBeContained SemType) bool {
	return IsSameType(context, Intersect(ty, typeToBeContained), typeToBeContained)
}

func ListProj(context Context, t SemType, key SemType) SemType {
	return listProjInnerVal(context, t, key)
}

func listMemberType(context Context, t SemType, key SemType) SemType {
	return ListMemberTypeInnerVal(context, t, key)
}
