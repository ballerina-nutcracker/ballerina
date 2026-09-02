// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser_test

import (
	"testing"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func TestSyntaxDiagnosticMetadataAndProperties(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := compilercontext.NewCompilerContext(env)
	tree, err := parser.GetSyntaxTree(cx, "main.bal", "public function main() { int x += 1; }")
	if err != nil {
		t.Fatal(err)
	}
	for diagnostic := range tree.Diagnostics() {
		info := diagnostic.DiagnosticInfo()
		if info.Code() == "" || info.MessageFormat() == "" || info.Severity() == 0 {
			t.Fatalf("incomplete diagnostic info: %#v", info)
		}
		properties := diagnostic.Properties()
		if len(properties) == 0 || properties[0].Kind() != diagnostics.Collection {
			t.Fatalf("unexpected diagnostic properties: %#v", properties)
		}
		return
	}
	t.Fatal("expected a syntax diagnostic")
}
