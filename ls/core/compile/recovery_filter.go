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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package compile

import "github.com/ballerina-nutcracker/ballerina/ast"

// compilationUnitsWithoutBadTopLevelNodes creates package-assembly views of
// recovered units without mutating the originals retained by the driver.
//
// Why this exists:
//   - Recovery-mode node building preserves a malformed module member as a
//     BLangBadTopLevelNode so the LS can retain its diagnostic and valid sibling
//     declarations.
//   - nodebuilder.ToPackageFromCompilationUnits is intentionally strict and
//     panics when that placeholder reaches package assembly.
//   - Filtering the assembly view follows the ls-ref approach and lets this
//     package reuse the compiler's declaration-assembly logic instead of
//     duplicating its top-level-node switch.
//
// Why this stays in the LS:
//   - Making the compiler assembler silently skip bad nodes would change its
//     contract for every compiler consumer and could hide an invalid AST.
//   - A compiler-wide recovery-aware assembler would require a separately
//     reviewed public API and compiler-pipeline corpus coverage.
//   - Ticket 39 is intentionally limited to changes under ls/.
//
// Current limitations:
//   - Only BLangBadTopLevelNode is filtered; malformed statements, expressions,
//     types, and identifiers remain outside this top-level recovery slice.
//   - Filtered units share valid child-node pointers with the recovered units
//     and are safe only as short-lived, read-only package-assembly inputs.
//   - A filtered unit copies the compilation-unit metadata known today. New
//     metadata added to BLangCompilationUnit must also be preserved here.
func compilationUnitsWithoutBadTopLevelNodes(units []*ast.BLangCompilationUnit) []*ast.BLangCompilationUnit {
	filtered := make([]*ast.BLangCompilationUnit, len(units))
	for i, unit := range units {
		filtered[i] = compilationUnitWithoutBadTopLevelNodes(unit)
	}
	return filtered
}

func compilationUnitWithoutBadTopLevelNodes(unit *ast.BLangCompilationUnit) *ast.BLangCompilationUnit {
	if unit == nil {
		return nil
	}

	topLevelNodes := make([]ast.TopLevelNode, 0, len(unit.TopLevelNodes))
	changed := false
	for _, node := range unit.TopLevelNodes {
		if _, ok := node.(*ast.BLangBadTopLevelNode); ok {
			changed = true
			continue
		}
		topLevelNodes = append(topLevelNodes, node)
	}
	if !changed {
		return unit
	}

	filtered := &ast.BLangCompilationUnit{
		TopLevelNodes: topLevelNodes,
		Name:          unit.Name,
		Scope:         unit.Scope,
	}
	filtered.SetPackageID(unit.GetPackageID())
	filtered.SetPosition(unit.GetPosition())
	filtered.SetDeterminedType(unit.GetDeterminedType())
	return filtered
}
