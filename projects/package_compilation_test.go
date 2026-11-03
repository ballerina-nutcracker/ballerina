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

package projects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// TestCollectModuleDiagnostics_DropsBIRDiagsOnlyWhenRequested covers the
// dropBIRDiags truncation directly: a sibling failing desugaring (the only
// way a module can reach BIR generation and still have the package end up
// errored, now that an analysis failure stops the package at the barrier
// before desugar/BIR run at all) must not surface noise recorded after a
// successful module's own BIR generation. Reaching a real desugar failure
// from Ballerina source isn't practical (desugar only ever reports internal
// errors), so this exercises collectModuleDiagnostics directly instead of
// through a corpus fixture.
func TestCollectModuleDiagnostics_DropsBIRDiagsOnlyWhenRequested(t *testing.T) {
	t.Parallel()
	require := test_util.NewRequire(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "single-file", "main.bal"))
	require.NoError(err)

	ballerinaEnvPath := os.Getenv(BallerinaEnvVar)
	if ballerinaEnvPath == "" {
		userHome, err := os.UserHomeDir()
		require.NoError(err)
		ballerinaEnvPath = filepath.Join(userHome, UserHomeDirName)
	}

	buildOptions := NewBuildOptionsBuilder().WithOffline(true).Build()
	fsys := os.DirFS(filepath.Dir(absPath))
	result, err := Load(fsys, filepath.Base(absPath), ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOptions,
	})
	require.NoError(err)

	compilation := result.Project().CurrentPackage().Compilation()
	if compilation.DiagnosticResult().HasErrors() {
		t.Fatalf("fixture failed to compile: %v", compilation.DiagnosticResult().Diagnostics())
	}

	modules := compilation.packageResolution.topologicallySortedModuleList
	var moduleCtx *moduleContext
	for _, m := range modules {
		if m.isDefault() {
			moduleCtx = m
			break
		}
	}
	require.NotNil(moduleCtx)
	if !moduleCtx.birGenAttempted {
		t.Fatalf("expected the default module to have attempted BIR generation")
	}

	// Stand in for a diagnostic from this module's own BIR generation, which
	// a sibling failing desugaring makes collectModuleDiagnostics drop.
	moduleCtx.compilerCtx.InternalError("synthetic noise after BIR generation", diagnostics.NewBuiltinLocation())

	var dropped []diagnostics.Diagnostic
	compilation.collectModuleDiagnostics(&dropped, true)
	for _, diag := range dropped {
		if diag.Message() == "synthetic noise after BIR generation" {
			t.Fatalf("dropBIRDiags=true: synthetic post-BIR-gen diagnostic was not dropped")
		}
	}

	var kept []diagnostics.Diagnostic
	compilation.collectModuleDiagnostics(&kept, false)
	found := false
	for _, diag := range kept {
		if diag.Message() == "synthetic noise after BIR generation" {
			found = true
		}
	}
	if !found {
		t.Fatalf("dropBIRDiags=false: synthetic post-BIR-gen diagnostic should still be present")
	}
}
