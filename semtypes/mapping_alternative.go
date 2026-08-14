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

import "slices"

type MappingFieldInfo struct {
	Name string
	Type SemType
}

type MappingAlternative struct {
	semType SemType
	pos     *MappingAtomicType
	neg     []MappingAtomicType
}

func (a MappingAlternative) Type() SemType {
	return a.semType
}

func MappingAlternatives(cx Context, t SemType) []MappingAlternative {
	if t.some() == 0 {
		if (t.all() & Mapping.all()) == 0 {
			return nil
		}
		return []MappingAlternative{{semType: Mapping, pos: nil, neg: nil}}
	}

	paths := []bddPath{}
	bddPathsPositive(getComplexSubtypeData(t, btMapping).(bdd), &paths, bddPathFrom())
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
			alts = append(alts, MappingAlternative{semType: intersectionSemType, pos: intersectionAtomType, neg: negAtoms})
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
	ty := createBasicSemType(btMapping, bddAtom(typeAtom))
	return ty, atom, true
}

func mappingAlternativeFieldTypeAllowed(cx Context, actual, expected SemType) bool {
	if IsSubtype(cx, expected, Number) && IsSubtype(cx, actual, Number) {
		return true
	}
	return IsSubtype(cx, actual, expected)
}

// MappingAlternativeAllowsFields checks if a given mapping alternative can be used to construct a mapping value using given fields and default fields.
// NOTE: default fields are not part of the semtype so we allow them to be injected from outside.
func MappingAlternativeAllowsFields(cx Context, alt MappingAlternative, fields []MappingFieldInfo, defaultableFields func(*MappingAtomicType) []string) bool {
	pos := alt.pos
	if pos != nil {
		if len(pos.names) == 0 {
			// map<T>
			expectedTy := cellInnerVal(pos.rest)
			for _, each := range fields {
				fieldTy := each.Type
				if !mappingAlternativeFieldTypeAllowed(cx, fieldTy, expectedTy) {
					return false
				}
			}
		} else {
			// For each named field check if we have a matching field in the fields
			// 	if so check that given field is a subtype
			// 		(NOTE: according to spec this is not required but we use this to further narrow the alternatives; jBallerina also do this)
			// 	else check field is optional or we have a default
			// For all fields that is not matched against a named field check that it can match the rest
			matchedWithNamed := make([]bool, len(fields))
			hasDefaults := defaultableFields(pos)
			for i, name := range pos.names {
				matched := false
				for j, f := range fields {
					if f.Name != name {
						continue
					}
					matchedWithNamed[j] = true
					expectedType := cellInnerVal(pos.types[i])
					if !mappingAlternativeFieldTypeAllowed(cx, f.Type, expectedType) {
						return false
					}
					matched = true
					break
				}
				if matched {
					continue
				}
				if !slices.Contains(hasDefaults, name) && !pos.IsOptional(cx, name) {
					return false
				}
			}
			expectedTy := cellInnerVal(pos.rest)
			for i, matched := range matchedWithNamed {
				if matched {
					continue
				}
				if IsNever(expectedTy) {
					return false
				}
				ty := fields[i].Type
				if !mappingAlternativeFieldTypeAllowed(cx, ty, expectedTy) {
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
