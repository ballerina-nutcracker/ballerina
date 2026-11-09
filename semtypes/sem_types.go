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

var SignedInt8 = intWidthSigned(8)
var SignedInt16 = intWidthSigned(16)
var SignedInt32 = intWidthSigned(32)
var UnsignedInt8 = Byte
var UnsignedInt16 = intWidthUnsigned(16)
var UnsignedInt32 = intWidthUnsigned(32)

func ContainsBasicType(t1 SemType, t2 SemType) bool {
	return ((widenToBasicTypeBits(t1) & t2.all()) != 0)
}

//shallow:allow public list projection hides the private projection implementation
func ListProj(context Context, t SemType, key SemType) SemType {
	return listProjInnerVal(context, t, key)
}
