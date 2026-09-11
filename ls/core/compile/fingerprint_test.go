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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package compile

import "testing"

// buildFingerprint drives a single module (default module of a fresh
// single-file project) through symbol resolution + top-level type resolution
// and returns its computed public-API fingerprint.
func buildFingerprint(t *testing.T, source string) fingerprint {
	t.Helper()
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/fp.bal")
	applyOpen(t, projSvc, u, source)
	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, input)
	if d.diagnosticContext().HasErrors() {
		t.Fatalf("unexpected errors: %v", d.diagnosticContext().Diagnostics())
	}
	return computeFingerprint(d.diagnosticContext(), d.exported)
}

func TestComputeFingerprint_SameSignature_SameFingerprint(t *testing.T) {
	fp1 := buildFingerprint(t, "public function greeting() returns string {\n    return \"a\";\n}\n")
	fp2 := buildFingerprint(t, "public function greeting() returns string {\n    return \"b\";\n}\n")
	if !fp1.valid || !fp2.valid {
		t.Fatalf("expected both fingerprints valid, got %+v %+v", fp1, fp2)
	}
	if !fp1.equal(fp2) {
		t.Errorf("body-only edit changed the fingerprint: %+v != %+v", fp1, fp2)
	}
}

func TestComputeFingerprint_DifferentSignature_DifferentFingerprint(t *testing.T) {
	fp1 := buildFingerprint(t, "public function greeting() returns string {\n    return \"a\";\n}\n")
	fp2 := buildFingerprint(t, "public function greeting() returns int {\n    return 1;\n}\n")
	if !fp1.valid || !fp2.valid {
		t.Fatalf("expected both fingerprints valid, got %+v %+v", fp1, fp2)
	}
	if fp1.equal(fp2) {
		t.Errorf("changed return type produced an equal fingerprint: %+v == %+v", fp1, fp2)
	}
}

func TestComputeFingerprint_ErroredContext_Invalid(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/fp-err.bal")
	applyOpen(t, projSvc, u, "public function greeting() returns UndefinedType {\n    return 1;\n}\n")
	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, input)

	fp := computeFingerprint(d.diagnosticContext(), d.exported)
	if fp.valid {
		t.Errorf("expected an invalid fingerprint for an errored context, got %+v", fp)
	}
	// Two invalid fingerprints must never compare equal — an invalid
	// fingerprint carries no information, so "unknown == unknown" would
	// incorrectly prune a dependent that may genuinely need resetting.
	if fp.equal(fp) {
		t.Error("an invalid fingerprint must never equal anything, including itself")
	}
}
