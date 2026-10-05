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

import "testing"

func TestRecoveredTypeDefinitionItem(t *testing.T) {
	req := Request{Text: "public type TEST_TYPE \n\nfunction testFunction() {}", Offset: len("public type TEST_TYPE ")}
	tokens := typeDefinitionTokens(req.Text)
	keyword := lastTypeDefinitionKeyword(tokens)
	name := nextTopLevelWord(tokens, keyword+1)
	next := nextTopLevelWord(tokens, name+1)
	if keyword < 0 || name < 0 || tokens[name].text != "TEST_TYPE" || next < 0 || tokens[next].start < req.Offset || typeDefinitionNextWord(tokens, name+1) != "function" {
		t.Fatalf("recovery tokens = %#v, keyword=%d name=%d", tokens, keyword, name)
	}
}

func TestTypeDefinitionContextAt(t *testing.T) {
	tests := []struct {
		name string
		text string
		want typeDefinitionContext
		ok   bool
	}{
		{
			name: "missing name",
			text: "public type ",
			want: typeDefinitionContext{position: typeDefinitionName},
			ok:   true,
		},
		{
			name: "incomplete name",
			text: "public type T",
			want: typeDefinitionContext{position: typeDefinitionName},
			ok:   true,
		},
		{
			name: "descriptor",
			text: "public type T ",
			want: typeDefinitionContext{position: typeDefinitionDescriptor},
			ok:   true,
		},
		{
			name: "qualified descriptor",
			text: "public type T module1:",
			want: typeDefinitionContext{position: typeDefinitionQualifiedDescriptor, alias: "module1"},
			ok:   true,
		},
		{
			name: "qualified descriptor prefix",
			text: "public type T module1:T",
			want: typeDefinitionContext{position: typeDefinitionQualifiedDescriptor, alias: "module1"},
			ok:   true,
		},
		{
			name: "union continuation",
			text: "type Cloneable readonly|xml|Cloneable[]|",
			want: typeDefinitionContext{position: typeDefinitionDescriptor},
			ok:   true,
		},
		{
			name: "comments and strings",
			text: "// type ignored\nstring note = \"type ignored\";\ntype T ",
			want: typeDefinitionContext{position: typeDefinitionDescriptor},
			ok:   true,
		},
		{
			name: "no declaration",
			text: "// type ignored\nstring note = \"type ignored\";",
			ok:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := typeDefinitionContextAt(Request{Text: test.text, Offset: len(test.text)})
			if ok != test.ok || got != test.want {
				t.Fatalf("typeDefinitionContextAt() = (%+v, %t), want (%+v, %t)", got, ok, test.want, test.ok)
			}
		})
	}
}
