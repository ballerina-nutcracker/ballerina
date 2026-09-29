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

package types

import (
	"fmt"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// collectIncludedMembers resolves each included type, validates that it can be included,
// and returns all their flattened InclusionMembers. positions is parallel to inclusions and pos is the position of
// the including type.
func collectIncludedMembers(t typeResolver, inclusions []model.SymbolRef, positions []diagnostics.Location, pos diagnostics.Location, depth int) ([]model.InclusionMember, bool) {
	var result []model.InclusionMember
	for i, symRef := range inclusions {
		t.ensureResolved(symRef, depth)
		incSym := getMemberCarrier(t, symRef)
		if incSym == nil {
			t.internalError("inclusion symbol is not a type symbol", diagnostics.Location{})
			return nil, false
		}
		if semtypes.IsZero(t.symbolType(symRef)) {
			t.semanticError("error resolving type inclusion", pos)
			return nil, false
		}
		if !validateInclusionMemberVisibility(t, symRef, incSym, positions[i]) {
			return nil, false
		}
		result = append(result, incSym.Members()...)
	}
	return result, true
}

// validateInclusionMemberVisibility checks the visibility regions of the included type's members. Regions are read
// from the included type itself, so they identify the module and class that declared each member even when the
// inclusion refers to an alias.
func validateInclusionMemberVisibility(t typeResolver, symRef model.SymbolRef, incSym model.MemberCarrier, pos diagnostics.Location) bool {
	currentModule := memberVisibility(t, false, "")
	for _, m := range incSym.Members() {
		region, ok := inclusionMemberRegion(t, symRef, m)
		if !ok || region == semtypes.VisibilityPublic {
			continue
		}
		module, _, isPrivate := strings.Cut(region, ":")
		if isPrivate {
			t.semanticError(fmt.Sprintf("incompatible type reference '%s': a referenced class cannot have private fields or methods",
				t.symbolName(symRef)), pos)
			return false
		}
		if module != currentModule {
			t.semanticError(fmt.Sprintf("incompatible type reference '%s': a referenced type across modules cannot have non-public fields or methods",
				t.symbolName(symRef)), pos)
			return false
		}
	}
	return true
}

// inclusionMemberRegion returns the visibility region of an included object member. It returns false for members
// that have no region, such as record fields.
func inclusionMemberRegion(t typeResolver, symRef model.SymbolRef, m model.InclusionMember) (string, bool) {
	if m.MemberKind() == model.InclusionMemberKindRestType {
		return "", false
	}
	visibilityTy := semtypes.ObjectMemberVisibility(t.typeContext(), semtypes.StringConst(m.MemberName()), t.symbolType(symRef))
	if semtypes.IsZero(visibilityTy) {
		return "", false
	}
	shape := semtypes.SingleShape(visibilityTy)
	if shape.IsEmpty() {
		return "", false
	}
	region, ok := shape.Get().Value.(string)
	return region, ok
}
