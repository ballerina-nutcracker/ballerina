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

package testharness

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

func TestPrintDiagnosticsUsesRegisteredTextDocument(t *testing.T) {
	env := diagnostics.NewDiagnosticEnv()
	env.RegisterFileIdentity("dependency-identity", "main.bal", text.TextDocumentFromText("dependency source\n"))
	location := diagnostics.NewLocationForIdentity(env, "dependency-identity", 0, 10)
	code := "TEST"
	diagnostic := diagnostics.NewDefaultDiagnostic(
		diagnostics.NewDiagnosticInfo(&code, "message", diagnostics.Error), location, nil,
	)
	var output bytes.Buffer
	PrintDiagnostics(
		fstest.MapFS{"main.bal": &fstest.MapFile{Data: []byte("root source\n")}},
		&output,
		projects.NewDiagnosticResult([]diagnostics.Diagnostic{diagnostic}),
		env,
	)
	if !strings.Contains(output.String(), "dependency source") || strings.Contains(output.String(), "root source") {
		t.Fatalf("diagnostic snippet did not use registered dependency text:\n%s", output.String())
	}
}
