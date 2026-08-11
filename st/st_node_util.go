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

package st

import "reflect"

func replaceInnerFallback(current STNode, target STNode, replacement STNode) (bool, STNode) {
	switch n := current.(type) {
	case *STNodeList:
		if n.IsEmpty() {
			return false, current
		}
		modified := false
		var newSTNodes []STNode
		for _, each := range n.children {
			m, e := replaceInner(each, target, replacement)
			modified = modified || m
			newSTNodes = append(newSTNodes, e)
		}
		if modified {
			return true, CreateNodeList(newSTNodes...)
		}
		return false, current

	case *STAmbiguousCollectionNode:

		modifiedCollectionStartToken, collectionStartTokenNode := replaceInner(n.CollectionStartToken, target, replacement)

		modifiedMembers, membersNode := replaceAll(n.Members, target, replacement)

		modifiedCollectionEndToken, collectionEndTokenNode := replaceInner(n.CollectionEndToken, target, replacement)

		modified := modifiedCollectionStartToken || modifiedMembers || modifiedCollectionEndToken
		if modified {
			children := []STNode{collectionStartTokenNode}
			children = append(children, membersNode...)
			children = append(children, collectionEndTokenNode)
			return true, createNodeAndAddChildren(&STAmbiguousCollectionNode{
				STNodeBase: n.STNodeBase,

				CollectionStartToken: collectionStartTokenNode,

				Members: membersNode,

				CollectionEndToken: collectionEndTokenNode,
			}, children...)
		}
		return false, current

	default:
		panic("unsupported node type: " + reflect.TypeOf(current).Name())
	}
}
