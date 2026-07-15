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

type FutureDefinition struct {
	mappingDefinition MappingDefinition
}

var _ Definition = &FutureDefinition{}

func NewFutureDefinition() FutureDefinition {
	return FutureDefinition{mappingDefinition: NewMappingDefinition()}
}

func (f *FutureDefinition) GetSemType(env Env) SemType {
	return futureContaining(f.mappingDefinition.GetSemType(env))
}

func (f *FutureDefinition) Define(env Env, constraint SemType) SemType {
	mappingType := f.mappingDefinition.Define(env, nil, constraint)
	return futureContaining(mappingType)
}

func FutureContaining(env Env, constraint SemType) SemType {
	definition := NewFutureDefinition()
	return definition.Define(env, constraint)
}

func futureContaining(mappingType SemType) SemType {
	bdd := subtypeDataAt(mappingType, btMapping).(bdd)
	return createBasicSemType(btFuture, bdd)
}

// FutureEventualType extracts T from future<T>.
// It returns Val for bare future and nil when futureTy is not a subtype of Future.
func FutureEventualType(ctx Context, futureTy SemType) SemType {
	if !IsSubtypeSimple(futureTy, Future) {
		return SemType{}
	}
	if futureTy.some() == 0 {
		return Val
	}
	mappingTy := convertFutureToMapping(ctx, futureTy)
	return MappingMemberTypeInnerVal(ctx, mappingTy, String)
}

func convertFutureToMapping(ctx Context, ty SemType) SemType {
	futureTy := Intersect(ty, Future)
	if IsEmpty(ctx, futureTy) {
		return SemType{}
	}
	bdd := subtypeDataAt(futureTy, btFuture)
	return createBasicSemType(btMapping, bdd)
}
