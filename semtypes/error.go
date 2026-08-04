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

import "github.com/ballerina-nutcracker/ballerina/common"

func ErrorDetailType(ctx Context, errorType SemType) (SemType, bool) {
	errorType = Intersect(errorType, Error)
	if IsNever(errorType) || !IsSubtype(ctx, errorType, Error) {
		return SemType{}, false
	}

	if IsSameType(ctx, errorType, Error) {
		return errorDetailTop(ctx), true
	}
	mappingSd := stripDistinctAtomsFromBdd(subtypeDataAt(errorType, btError).(bdd))
	if allOrNothing, ok := mappingSd.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return errorDetailTop(ctx), true
		}
		return SemType{}, false
	}
	return getBasicSubtype(btMapping, mappingSd.(properSubtypeData)), true
}

func errorDetailTop(ctx Context) SemType {
	md := NewMappingDefinition()
	return md.Define(ctx.Env(), nil, CreateCloneable(ctx))
}

func stripErrorDistinctAtoms(ty SemType) SemType {
	return stripDistinctAtomsFromSemType(ty, btError, stripDistinctAtomsFromBdd)
}

func stripDistinctAtomsFromBdd(bdd bdd) bdd {
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
	mappingSd := subtypeDataAt(detail, btMapping)
	if allOrNothingSubtype, ok := mappingSd.(allOrNothingSubtype); ok {
		if allOrNothingSubtype.IsAllSubtype() {
			return Error
		} else {
			return Never
		}
	}
	sd := bddIntersect(mappingSd.(bdd), bddSubtypeRo)
	if sd == bddSubtypeRo {
		return Error
	}
	return getBasicSubtype(btError, sd.(properSubtypeData))
}

func ErrorDistinct(distinctId int) SemType {
	common.Assert(func() bool { return distinctId >= 0 })
	bdd := bddAtom(new(createDistinctRecAtom(((-distinctId) - 1))))
	return getBasicSubtype(btError, bdd)
}
