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

import "math/bits"

type unpackComplexSemType struct{}

func newUnpackComplexSemType() unpackComplexSemType {
	this := unpackComplexSemType{}
	return this
}

func unpack(t SemType) []basicSubtype {
	some := t.some()
	var subtypeList []basicSubtype
	for _, data := range t.subtypeDataList() {
		code := bits.TrailingZeros(uint(some))
		subtypeList = append(subtypeList, basicSubtypeFrom(basicTypeCodeFrom(code), data))
		some = some & ^(1 << code)
	}
	return subtypeList
}
