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

// SemType represents a type in Ballerina.
type SemType struct {
	allBits  basicTypeBitSet
	someBits basicTypeBitSet
	dataList []properSubtypeData
}

func (t SemType) all() basicTypeBitSet {
	return t.allBits & basicTypeMask
}

func (t SemType) some() basicTypeBitSet {
	return t.someBits & basicTypeMask
}

func (t SemType) subtypeDataList() []properSubtypeData {
	return t.dataList
}

// InstanceKey identifies a SemType's representation: copies of one value share a key, but equal types
// built separately do not, so it serves only as a cache key in front of a type-equality check.
type InstanceKey struct {
	allBits  basicTypeBitSet
	someBits basicTypeBitSet
	data     *properSubtypeData
}

func (t SemType) InstanceKey() InstanceKey {
	key := InstanceKey{allBits: t.allBits, someBits: t.someBits}
	if len(t.dataList) > 0 {
		key.data = &t.dataList[0]
	}
	return key
}
