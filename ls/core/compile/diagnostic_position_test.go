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
	"reflect"
	"testing"
)

func TestPublishedDiagnosticPositionsSurviveEdits(t *testing.T) {
	t.Parallel()
	ps, svc := newTestServices(t)
	u := tempFileURI(t, "main.bal")
	applyOpen(t, ps, u, unusedTopLevelSource)
	compileMessages(t, svc, u)
	root, err := ps.ProjectKey(u)
	if err != nil {
		t.Fatal(err)
	}
	published, generation, ok := svc.DiagnosticsFor(root)
	if !ok || len(published[u]) != 1 {
		t.Fatalf("first published diagnostics = %v, want one diagnostic", published)
	}
	before := append([]CompilerDiagnostic(nil), published[u]...)
	firstSnapshot, ok := svc.store.Stable(root)
	if !ok {
		t.Fatal("first stable snapshot missing")
	}
	if d := before[0]; d.StartLine != 0 || d.StartChar != 0 || d.EndLine != 0 || d.EndChar != 21 {
		t.Fatalf("original diagnostic range = %+v, want (0, 0)-(0, 21)", d)
	}
	project, pkg := currentPackage(t, ps, u)

	updateDoc(t, ps, u.String(), "\n\n    "+unusedTopLevelSource, 2)
	compileMessages(t, svc, u)
	requireSamePackage(t, ps, u, project, pkg)
	edited, editedGeneration, ok := svc.DiagnosticsFor(root)
	if !ok || editedGeneration <= generation || len(edited[u]) != 1 {
		t.Fatalf("edited diagnostics = %v, generation = %d, want one diagnostic newer than %d", edited, editedGeneration, generation)
	}
	if d := edited[u][0]; d.StartLine != 2 || d.StartChar != 4 || d.EndLine != 2 || d.EndChar != 25 {
		t.Fatalf("edited diagnostic range = %+v, want (2, 4)-(2, 25)", d)
	}
	if !reflect.DeepEqual(published[u], before) {
		t.Fatalf("previously published diagnostics changed: %v -> %v", before, published[u])
	}
	if got := firstSnapshot.Diagnostics(); !reflect.DeepEqual(got, before) {
		t.Fatalf("previous snapshot diagnostics changed: %v -> %v", before, got)
	}
}
