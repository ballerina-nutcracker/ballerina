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

type MappingFieldInfo struct {
	Name string
	Ty   SemType
}

type MappingAlternative struct {
	SemType SemType
	Pos     *MappingAtomicType
	neg     []MappingAtomicType
}

func MappingAlternatives(cx Context, t SemType) []MappingAlternative {
	if t.some() == 0 {
		if (t.all() & MAPPING.all()) == 0 {
			return nil
		}
		return []MappingAlternative{{SemType: MAPPING, Pos: nil, neg: nil}}
	}

	paths := []bddPath{}
	bddPathsPositive(getComplexSubtypeData(t, BTMapping).(Bdd), &paths, bddPathFrom())
	alts := []MappingAlternative{}
	for _, bddPath := range paths {
		posAtoms := make([]*MappingAtomicType, len(bddPath.pos))
		for i := 0; i < len(bddPath.pos); i++ {
			posAtoms[i] = cx.MappingAtomType(bddPath.pos[i])
		}
		intersectionSemType, intersectionAtomType, ok := intersectMappingAtoms(cx.Env(), posAtoms)
		if ok {
			negAtoms := make([]MappingAtomicType, len(bddPath.neg))
			for i := 0; i < len(bddPath.neg); i++ {
				negAtoms[i] = *cx.MappingAtomType(bddPath.neg[i])
			}
			alts = append(alts, MappingAlternative{SemType: intersectionSemType, Pos: intersectionAtomType, neg: negAtoms})
		}
	}
	return alts
}

func intersectMappingAtoms(env Env, atoms []*MappingAtomicType) (SemType, *MappingAtomicType, bool) {
	if len(atoms) == 0 {
		return SemType{}, nil, false
	}
	atom := atoms[0]
	for i := 1; i < len(atoms); i++ {
		result := intersectMapping(env, atom, atoms[i])
		if result == nil {
			return SemType{}, nil, false
		}
		atom = result
	}
	typeAtom := env.mappingAtom(atom)
	ty := createBasicSemType(BTMapping, bddAtom(typeAtom))
	return ty, atom, true
}

// NOTE: selection is not affected by default values according to the spec, it is purely by field names
// But we are checking the type as well to allow things like map<int>|map<string> given jballerina already allow this
// and it's (mostly) straightforward to support it. Edge case is when we have numeric types, we can't
// determine a literal in rhs to be which numeric type without deciding the type in lhs. We currently work around this
// by widening both to numeric
func MappingAlternativeAllowsFields(cx Context, alt MappingAlternative, fields []MappingFieldInfo) bool {
	pos := alt.Pos
	if pos != nil {
		if len(pos.Names) == 0 {
			// map<T>
			for _, each := range fields {
				fieldTy := each.Ty
				fieldName := each.Name
				expectedTy := pos.FieldInnerVal(fieldName)
				if !IsSubtype(cx, fieldTy, expectedTy) {
					return false
				}

			}
		} else {
			i := 0
			n := len(fields)
		names:
			for _, name := range pos.Names {
				for {
					if i >= n {
						if pos.IsOptional(cx, name) {
							continue names
						}
						return false
					}
					fieldName := fields[i].Name
					fieldTy := fields[i].Ty
					expectedTy := pos.FieldInnerVal(fieldName)
					if IsSubtype(cx, expectedTy, NUMBER) && IsSubtype(cx, fieldTy, NUMBER) {
						expectedTy = NUMBER
					}
					if IsNever(expectedTy) || !IsSubtype(cx, fieldTy, expectedTy) {
						return false
					}
					if fieldName == name {
						i += 1
						continue names
					}
					if fieldName > name {
						if pos.IsOptional(cx, name) {
							continue names
						}
						return false
					}
					// in < case only type check is needed and FieldInnerVal give the rest type correctly
					i += 1
				}
			}
			for ; i < n; i++ {
				expectedTy := pos.FieldInnerVal(fields[i].Name)
				if IsNever(expectedTy) || !IsSubtype(cx, fields[i].Ty, expectedTy) {
					return false
				}
			}
		}
	}
	if len(alt.neg) != 0 {
		panic("unexpected negative atom in mapping alternative")
	}
	return true
}
