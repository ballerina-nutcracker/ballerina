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

type ObjectAlternative struct {
	objectType       SemType
	initFunctionType SemType
}

func (a ObjectAlternative) Type() SemType {
	return a.objectType
}

func (a ObjectAlternative) InitFunctionType() SemType {
	return a.initFunctionType
}

// ObjectPositiveAtom returns the sole positive object mapping atom, excluding
// the readonly constraint.
func ObjectPositiveAtom(cx Context, t SemType) (*MappingAtomicType, bool) {
	mappingTy := convertObjectToMappingTy(cx, t)
	if mappingTy.some() == 0 {
		return nil, false
	}
	paths := []bddPath{}
	bddPathsPositive(getComplexSubtypeData(mappingTy, btMapping).(bdd), &paths, bddPathFrom())
	if len(paths) != 1 || len(paths[0].neg) != 0 {
		return nil, false
	}
	atoms := paths[0].pos
	var candidate atom
	switch len(atoms) {
	case 1:
		if isObjectReadonlyAtom(atoms[0]) {
			return nil, false
		}
		candidate = atoms[0]
	case 2:
		switch {
		case isObjectReadonlyAtom(atoms[0]):
			candidate = atoms[1]
		case isObjectReadonlyAtom(atoms[1]):
			candidate = atoms[0]
		default:
			return nil, false
		}
	default:
		return nil, false
	}
	mappingAtom := cx.MappingAtomType(candidate)
	if mappingAtom == nil {
		return nil, false
	}
	return mappingAtom, true
}

func isObjectReadonlyAtom(atom atom) bool {
	rec, ok := atom.(*recAtom)
	return ok && rec.index() == bddRecAtomObjectReadonly
}

func IsAtomicObjectType(cx Context, t SemType) bool {
	return IsSubtype(cx, t, Object) && len(ObjectAlternatives(cx, t)) == 1
}

func ObjectAlternatives(cx Context, t SemType) []ObjectAlternative {
	mappingTy := convertObjectToMappingTy(cx, t)
	mappingAlternatives := MappingAlternatives(cx, mappingTy)
	var alts []ObjectAlternative
	initKey := StringConst("init")
	for _, each := range mappingAlternatives {
		if len(each.neg) > 0 {
			continue
		}
		objectTy := convertMappingToObjectTy(cx, each.semType)
		initTy := ObjectMemberType(cx, initKey, objectTy)
		alts = append(alts, ObjectAlternative{
			objectType:       objectTy,
			initFunctionType: initTy,
		})
	}

	return alts
}
