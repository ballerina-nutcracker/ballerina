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

type functionAtomicType struct {
	ParamType  SemType
	RetType    SemType
	Qualifiers SemType
	IsGeneric  bool
}

var _ atomicType = &functionAtomicType{}

func functionAtomicTypeFrom(paramType SemType, rest SemType, qualifiers SemType) functionAtomicType {

	return newFunctionAtomicType(paramType, rest, qualifiers, false)
}

func functionAtomicTypeGenericFrom(paramType SemType, rest SemType, qualifiers SemType) functionAtomicType {

	return newFunctionAtomicType(paramType, rest, qualifiers, true)
}

func newFunctionAtomicType(paramType SemType, retType SemType, qualifiers SemType, isGeneric bool) functionAtomicType {
	this := functionAtomicType{}
	this.ParamType = paramType
	this.RetType = retType
	this.Qualifiers = qualifiers
	this.IsGeneric = isGeneric
	return this
}

func (f *functionAtomicType) atomKind() kind {
	return kind_FUNCTION_ATOM
}
