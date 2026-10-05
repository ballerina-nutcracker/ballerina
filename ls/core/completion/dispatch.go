// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package completion

import "github.com/ballerina-nutcracker/ballerina/ls/protocol"

// handler pairs a chain-context predicate with the candidate builder that
// applies once matched. match inspects the cursor (typically its chain) and
// returns a typed context value for build to use, so build never has to
// redo match's own node inspection.
type handler struct {
	name  string
	match func(*cursor) (any, bool)
	build func(*cursor, any) []protocol.CompletionItem
}

// handlers is ordered highest-priority (most specific node context) first.
// dispatch walks it in order and returns the first match's build result --
// no priority tiebreak, no registry/init() auto-registration (see the
// completion-provider dispatch seam ADR). New per-family handlers land here
// as they're built; this slice is the only thing that grows -- each
// handler's own match/build logic lives in its own file.
var handlers = []handler{
	// new per-family handlers land here as they're built, ordered
	// highest-priority (most specific node context) first
	typeDefinitionHandler,
	enumDeclarationHandler,
	fieldAccessHandler,
	invocationHandler,
	importHandler,
	moduleHandler,
	blockHandler,
	lexicalHandler,
}

func dispatch(c *cursor) []protocol.CompletionItem {
	for _, h := range handlers {
		if ctx, ok := h.match(c); ok {
			return h.build(c, ctx)
		}
	}
	return emptyItems
}
