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

// lexicalHandler is the seam's lowest-priority, most-general fallback: no
// more specific handler above it matched, but the cursor still sits
// somewhere with a resolvable enclosing scope -- migrated from classify()'s
// kindLexical case verbatim (its "no chain-match at all" branch).
var lexicalHandler = handler{
	name:  "lexical",
	match: matchLexical,
	build: buildLexical,
}

func matchLexical(c *cursor) (any, bool) {
	if _, ok := c.winner(); ok {
		return nil, false
	}
	if nearestScope(c.chain) == nil {
		return nil, false
	}
	return nil, true
}

func buildLexical(c *cursor, _ any) []protocol.CompletionItem {
	return lexicalItems(c, nil)
}
