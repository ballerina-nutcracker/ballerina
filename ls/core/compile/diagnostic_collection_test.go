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

package compile

import (
	"os"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func TestReplacedDiagnosticEnvironmentsAreCollected(t *testing.T) {
	t.Parallel()
	replaced := compileImportChangingEdits(t)
	for attempt := range 20 {
		runtime.GC()
		remaining := 0
		for _, env := range replaced {
			if env.Value() != nil {
				remaining++
			}
		}
		if remaining == 0 {
			t.Logf("all %d replaced environments collected after %d GC attempts", len(replaced), attempt+1)
			return
		}
		if attempt == 19 {
			t.Fatalf("%d/%d replaced diagnostic environments remain reachable after 20 GC attempts", remaining, len(replaced))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Return only weak pointers after normal shutdown, without test cleanup closures
// retaining the project service or compilation snapshots through the GC check.
func compileImportChangingEdits(t *testing.T) []weak.Pointer[diagnostics.DiagnosticEnv] {
	t.Helper()
	bus := event.New()
	defer bus.Close()
	ps := workspace.New(os.DirFS, bus)
	svc := New(ps, bus, WithDebounce(0))
	defer svc.Shutdown()
	u := tempFileURI(t, "main.bal")
	withoutImport := "public function main() {\n    io:println(1);\n}\n"
	applyOpen(t, ps, u, withoutImport)
	if messages := compileMessages(t, svc, u); !hasUnknown(messages) {
		t.Fatalf("diagnostics without import = %q, want unknown symbol", messages)
	}

	replaced := make([]weak.Pointer[diagnostics.DiagnosticEnv], 0, 50)
	for edit := range 50 {
		project, _ := currentPackage(t, ps, u)
		oldEnv := project.Environment().CompilerEnvironment().DiagnosticEnv()
		replaced = append(replaced, weak.Make(oldEnv))
		source := ioImportSource
		if edit%2 != 0 {
			source = withoutImport
		}
		updateDoc(t, ps, u.String(), source, int32(edit+2))
		messages := compileMessages(t, svc, u)
		if edit%2 == 0 {
			requireNoUnknown(t, messages)
		} else if !hasUnknown(messages) {
			t.Fatalf("edit %d diagnostics = %q, want unknown symbol after removing import", edit+1, messages)
		}
		current, _ := currentPackage(t, ps, u)
		if oldEnv == current.Environment().CompilerEnvironment().DiagnosticEnv() {
			t.Fatalf("edit %d did not replace the diagnostic environment", edit+1)
		}
	}
	return replaced
}
