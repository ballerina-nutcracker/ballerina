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

import "ballerina/model"

// CFGInvariantError represents a CFG invariant violation
type CFGInvariantError struct {
	FuncRef        model.SymbolRef
	BlockID        int
	BackedgeParent int
	Parents        []int
}

// ValidateInvariants checks that CFG invariants hold (e.g., backedgeParents is subset of parents).
// Returns a list of violations, or nil if all invariants hold.
func (cfg *PackageCFG) ValidateInvariants() []CFGInvariantError {
	var errors []CFGInvariantError
	for symRef, fcfg := range cfg.allFunctionCfgs {
		for _, bb := range fcfg.bbs {
			parentSet := make(map[int]bool, len(bb.parents))
			for _, p := range bb.parents {
				parentSet[p] = true
			}
			for _, p := range bb.backedgeParents {
				if !parentSet[p] {
					errors = append(errors, CFGInvariantError{
						FuncRef:        symRef,
						BlockID:        bb.id,
						BackedgeParent: p,
						Parents:        bb.parents,
					})
				}
			}
		}
	}
	return errors
}
