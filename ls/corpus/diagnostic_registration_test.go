// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package corpus

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"weak"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func TestEditedDiagnosticRegistration(t *testing.T) {
	t.Parallel()
	var released weak.Pointer[diagnostics.DiagnosticEnv]
	runTranscriptWithCheck(t, "sync/testdata/diagnostic_position.didChange.json", func(ps *workspace.ProjectService, _ *compile.CompilationService, root string) {
		u, err := workspace.FileURIFromPath(filepath.Join(root, "diagnostic_position.bal"))
		if err != nil {
			t.Fatal(err)
		}
		project, err := ps.Project(u)
		if err != nil || project == nil {
			t.Fatalf("Project: %v", err)
		}
		name, ok := ps.Paths(project).NameOf(u)
		if !ok {
			t.Fatal("edited document has no registration name")
		}
		text, ok := ps.OpenText(project, name)
		if !ok {
			t.Fatal("edited document has no open text")
		}
		offset := strings.Index(text, "int unused")
		if offset < 0 {
			t.Fatal("edited declaration missing")
		}
		env := project.Environment().CompilerEnvironment().DiagnosticEnv()
		released = weak.Make(env)
		loc := diagnostics.NewLocation(env, name, offset, offset)
		if line, column := env.StartLine(loc), env.StartColumn(loc); line != 2 || column != 6 {
			t.Fatalf("current diagnostic registration resolves edited declaration to (%d, %d), want (2, 6)", line, column)
		}
	})
	for range 20 {
		runtime.GC()
		if released.Value() == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("diagnostic environment remains reachable after the fixture session shut down")
}
