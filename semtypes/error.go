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

import "ballerina-lang-go/common"

func ErrorDetailType(ctx Context, errorType SemType) (SemType, bool) {
	errorType = Intersect(errorType, ERROR)
	if IsNever(errorType) || !IsSubtype(ctx, errorType, ERROR) {
		return SemType{}, false
	}

	if IsSameType(ctx, errorType, ERROR) {
		return errorDetailTop(ctx), true
	}
	mappingSd := stripDistinctAtomsFromBdd(subtypeData(errorType, BTError).(Bdd))
	if allOrNothing, ok := mappingSd.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return errorDetailTop(ctx), true
		}
		return SemType{}, false
	}
	return getBasicSubtype(BTMapping, mappingSd.(ProperSubtypeData)), true
}

func errorDetailTop(ctx Context) SemType {
	md := NewMappingDefinition()
	return md.DefineMappingTypeWrapped(ctx.Env(), nil, CreateCloneable(ctx))
}

func stripErrorDistinctAtoms(ty SemType) SemType {
	return stripDistinctAtomsFromSemType(ty, BTError, stripDistinctAtomsFromBdd)
}

func stripDistinctAtomsFromBdd(bdd Bdd) Bdd {
	var paths []bddPath
	bddPathsPositive(bdd, &paths, bddPathFrom())
	if len(paths) == 0 {
		return bddNothing()
	}
	result := paths[0].bdd
	for _, path := range paths[1:] {
		result = bddUnion(result, path.bdd)
	}
	return result
}

func ErrorWithDetail(detail SemType) SemType {
	mappingSd := subtypeData(detail, BTMapping)
	if allOrNothingSubtype, ok := mappingSd.(allOrNothingSubtype); ok {
		if allOrNothingSubtype.IsAllSubtype() {
			return ERROR
		} else {
			return NEVER
		}
	}
	sd := bddIntersect(mappingSd.(Bdd), BDD_SUBTYPE_RO)
	if sd == BDD_SUBTYPE_RO {
		return ERROR
	}
	return getBasicSubtype(BTError, sd.(ProperSubtypeData))
}

func ErrorDistinct(distinctId int) SemType {
	common.Assert(func() bool { return distinctId >= 0 })
	bdd := bddAtom(new(createDistinctRecAtom(((-distinctId) - 1))))
	return getBasicSubtype(BTError, bdd)
}
