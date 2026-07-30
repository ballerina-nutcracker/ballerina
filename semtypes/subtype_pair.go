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

type subtypePair struct {
	basicTypeCode basicTypeCode
	SubtypeData1  properSubtypeData
	SubtypeData2  properSubtypeData
}

func newSubtypePairFromBasicTypeCodeProperSubtypeDataProperSubtypeData(basicTypeCode basicTypeCode, subtypeData1 properSubtypeData, subtypeData2 properSubtypeData) subtypePair {
	this := subtypePair{}
	this.basicTypeCode = basicTypeCode
	this.SubtypeData1 = subtypeData1
	this.SubtypeData2 = subtypeData2
	return this
}

func createSubTypePair(basicTypeCode basicTypeCode, subtypeData1 properSubtypeData, subtypeData2 properSubtypeData) subtypePair {
	return newSubtypePairFromBasicTypeCodeProperSubtypeDataProperSubtypeData(basicTypeCode, subtypeData1, subtypeData2)
}
