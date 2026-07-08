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

// StreamDefinition represents a stream<T, C> type descriptor.
//
// @since 2201.12.0
type StreamDefinition struct {
	listDefinition ListDefinition
}

var _ Definition = &StreamDefinition{}

func NewStreamDefinition() StreamDefinition {
	this := StreamDefinition{}
	this.listDefinition = NewListDefinition()
	return this
}

func streamContaining(tupleType SemType) SemType {
	bdd := subtypeData(tupleType, BTList)
	return createBasicSemType(BTStream, bdd)
}

func (s *StreamDefinition) GetSemType(env Env) SemType {
	return streamContaining(s.listDefinition.GetSemType(env))
}

func (s *StreamDefinition) Define(env Env, valueTy SemType, completionTy SemType) SemType {
	if sameSemType(VAL, completionTy) && sameSemType(VAL, valueTy) {
		return STREAM
	}
	tuple := s.listDefinition.TupleTypeWrapped(env, valueTy, completionTy)
	return streamContaining(tuple)
}
