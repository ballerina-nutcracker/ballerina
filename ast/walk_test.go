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

package ast

import "testing"

type badNodeVisitor struct {
	bad    *BLangBadTopLevelNode
	visits int
}

func (v *badNodeVisitor) Visit(node BLangNode) Visitor {
	if node == v.bad {
		v.visits++
	}
	return v
}

func (v *badNodeVisitor) VisitTypeData(*TypeData) Visitor { return v }

// Corpus tests cannot supply a custom visitor to observe Walk's API contract;
// retained bad nodes have no executable Ballerina behavior to assert.
func TestWalkVisitsBadTopLevelNodes(t *testing.T) {
	t.Parallel()
	for _, owner := range []string{"package", "class", "service"} {
		t.Run(owner, func(t *testing.T) {
			t.Parallel()
			bad := &BLangBadTopLevelNode{}
			var root BLangNode
			switch owner {
			case "package":
				pkg := NewBLangPackage()
				pkg.BadTopLevelNodes = []*BLangBadTopLevelNode{bad}
				root = pkg
			case "class":
				class := NewBLangClassDefinition()
				class.BadTopLevelNodes = []*BLangBadTopLevelNode{bad}
				root = &class
			case "service":
				service := NewBLangService()
				service.BadTopLevelNodes = []*BLangBadTopLevelNode{bad}
				root = &service
			}
			visitor := &badNodeVisitor{bad: bad}
			Walk(visitor, root)
			if visitor.visits != 1 {
				t.Errorf("retained bad node visited %d times, want 1", visitor.visits)
			}
		})
	}
}
