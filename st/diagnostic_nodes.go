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

package st

import "iter"

// DiagnosticSTNodes visits every node with its own diagnostics, including invalid
// minutiae. Positions include leading minutiae, like Node.Position().
func DiagnosticSTNodes(node STNode, position int) iter.Seq2[STNode, int] {
	return positionedSTNodes(node, position, true)
}

// TextRange locates a diagnostic's source, including a discarded lexer token
// referenced by the diagnostic rather than the following token it is attached to.
func (d STNodeDiagnostic) TextRange(node STNode, position int) TextRange {
	for _, arg := range d.args {
		target, ok := arg.(STNode)
		if !ok {
			continue
		}
		for child, childPosition := range positionedSTNodes(node, position, false) {
			if child == target {
				return diagnosticNodeRange(child, childPosition)
			}
		}
	}
	return diagnosticNodeRange(node, position)
}

func diagnosticNodeRange(node STNode, position int) TextRange {
	start := position + int(node.WidthWithLeadingMinutiae()) - int(node.Width())
	return TextRange{StartOffset: start, EndOffset: start + int(node.Width()), Length: int(node.Width())}
}

func positionedSTNodes(node STNode, position int, diagnosticsOnly bool) iter.Seq2[STNode, int] {
	return func(yield func(STNode, int) bool) {
		var visit func(STNode, int) bool
		visit = func(node STNode, position int) bool {
			if node == nil || !IsSTNodePresent(node) || diagnosticsOnly && !node.HasDiagnostics() {
				return true
			}
			if (!diagnosticsOnly || len(node.Diagnostics()) > 0) && !yield(node, position) {
				return false
			}
			if token, ok := node.(STToken); ok {
				if !visit(token.LeadingMinutiae(), position) {
					return false
				}
				return visit(token.TrailingMinutiae(), position+int(token.WidthWithLeadingMinutiae()))
			}
			for _, child := range childSTNodesForDiagnosticSearch(node) {
				if child == nil || !IsSTNodePresent(child) {
					continue
				}
				if !visit(child, position) {
					return false
				}
				position += int(child.WidthWithMinutiae())
			}
			return true
		}
		visit(node, position)
	}
}
