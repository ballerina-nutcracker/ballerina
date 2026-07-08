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

package semantics

import (
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

// collectIncludedMembers resolves each included type, validates it is a subtype of expectedBasicType,
// and returns all their flattened InclusionMembers.
func collectIncludedMembers(t typeResolver, inclusions []model.SymbolRef, depth int) ([]model.InclusionMember, bool) {
	var result []model.InclusionMember
	for _, symRef := range inclusions {
		t.ensureResolved(symRef, depth)
		incSym := getMemberCarrier(t, symRef)
		if incSym == nil {
			t.internalError("inclusion symbol is not a type symbol", diagnostics.Location{})
			return nil, true
		}
		if semtypes.IsZero(t.symbolType(symRef)) {
			return nil, true
		}
		result = append(result, incSym.Members()...)
	}
	return result, false
}
