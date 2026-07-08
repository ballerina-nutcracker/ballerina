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

func createComplexSemType(allBitset basicTypeBitSet, subtypeList ...basicSubtype) SemType {
	return createComplexSemTypeWithAllBitSetSubtypeList(allBitset, subtypeList)
}

func createComplexSemTypeWithAllBitSetSomeBitSetSubtypeDataList(allBitset, someBitset basicTypeBitSet, subtypeDataList []ProperSubtypeData) SemType {
	return SemType{
		allBits:  semTypeMarker | (allBitset & basicTypeMask),
		someBits: someBitset & basicTypeMask,
		dataList: subtypeDataList,
	}
}

func createComplexSemTypeWithAllBitSetSubtypeList(allBitset basicTypeBitSet, subtypeList []basicSubtype) SemType {
	var some basicTypeBitSet = 0
	dataList := make([]ProperSubtypeData, len(subtypeList))
	for i, basicSubtype := range subtypeList {
		dataList[i] = basicSubtype.SubtypeData
		c := basicSubtype.BasicTypeCode.Code()
		some = (some | (1 << c))
	}
	return createComplexSemTypeWithAllBitSetSomeBitSetSubtypeDataList(allBitset, some, dataList)
}
