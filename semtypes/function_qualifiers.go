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

type FunctionQualifiers struct {
	semType SemType
}

func newFunctionQualifiersFromSemType(semType SemType) FunctionQualifiers {
	this := FunctionQualifiers{}
	this.semType = semType
	return this
}

func FunctionQualifiersFrom(env Env, isolated bool, transactional bool) FunctionQualifiers {
	return newFunctionQualifiersFromSemType(createSemType(env, isolated, transactional))
}

func createSemType(env Env, isolated bool, transactional bool) SemType {
	ld := NewListDefinition()
	var isolatedType SemType
	if isolated {
		isolatedType = BooleanConst(true)
	} else {
		isolatedType = BOOLEAN
	}
	var transactionalType SemType
	if transactional {
		transactionalType = BOOLEAN
	} else {
		transactionalType = BooleanConst(false)
	}
	return ld.DefineListTypeWrappedWithEnvSemTypesInt(env, []SemType{isolatedType, transactionalType}, 2)
}
