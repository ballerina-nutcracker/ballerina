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

package cfg

import "github.com/ballerina-nutcracker/ballerina/model"

type InvariantError struct {
	FuncRef        model.SymbolRef
	BlockID        int
	BackedgeParent int
	Parents        []int
}

func ValidateInvariants(graph *PackageCFG) []InvariantError {
	var errors []InvariantError
	for symRef, functionGraph := range graph.allFunctionCfgs {
		for _, block := range functionGraph.bbs {
			parentSet := make(map[int]bool, len(block.parents))
			for _, parent := range block.parents {
				parentSet[parent] = true
			}
			for _, parent := range block.backedgeParents {
				if !parentSet[parent] {
					errors = append(errors, InvariantError{
						FuncRef:        symRef,
						BlockID:        block.id,
						BackedgeParent: parent,
						Parents:        block.parents,
					})
				}
			}
		}
	}
	return errors
}
