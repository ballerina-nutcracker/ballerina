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

package corpus

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/bir"
	bircodec "github.com/ballerina-nutcracker/ballerina/bir/codec"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/model/symbolpool"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type serializationFixture struct {
	birPkg          *bir.BIRPackage
	exportedSymbols model.ExportedSymbolSpace
	compilerEnv     *context.CompilerEnvironment
	tyEnv           semtypes.Env
}

func compileForSerializationBench(b *testing.B, tc test_util.TestCase) *serializationFixture {
	b.Helper()

	fsys := os.DirFS(filepath.Dir(tc.InputPath))
	entry := filepath.Base(tc.InputPath)
	if tc.IsProject {
		fsys = os.DirFS(tc.InputPath)
		entry = "."
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		b.Fatalf("getBallerinaEnvPath: %v", err)
	}
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)

	packageDirName := ""
	if tc.IsProject {
		packageDirName = filepath.Base(tc.InputPath)
	}
	compiled, err := testharness.CompileWithDriver(fsys, entry, packageDirName, projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
	})
	if err != nil {
		b.Fatalf("driver compilation (%s): %v", tc.InputPath, err)
	}
	compilerEnv := compiled.Resolver.CompilerEnvironment()
	tyEnv := compilerEnv.GetTypeEnv()

	var stderrBuf bytes.Buffer
	diagnosticResult := projects.NewDiagnosticResult(compiled.Context.Diagnostics())
	testharness.PrintDiagnostics(fsys, &stderrBuf, diagnosticResult, compiled.Context.DiagnosticEnv())
	if diagnosticResult.HasErrors() {
		b.Fatalf("compile errors for %s:\n%s", tc.InputPath, stderrBuf.String())
	}

	if len(compiled.BIRPackages) == 0 {
		b.Fatalf("nil BIR for %s", tc.InputPath)
	}
	birPkg := compiled.BIRPackages[len(compiled.BIRPackages)-1]
	if birPkg == nil {
		b.Fatalf("nil BIR for %s", tc.InputPath)
		return nil
	}
	pkgID := birPkg.PackageID
	if pkgID == nil || pkgID.OrgName == nil || pkgID.PkgName == nil {
		b.Fatalf("BIR package has incomplete package ID for %s", tc.InputPath)
		return nil
	}

	exported := nonEmptySerializationSymbols(compilerEnv)
	return &serializationFixture{birPkg: birPkg, exportedSymbols: exported, compilerEnv: compilerEnv, tyEnv: tyEnv}
}

func nonEmptySerializationSymbols(env *context.CompilerEnvironment) model.ExportedSymbolSpace {
	cx := context.NewCompilerContext(env)
	pkgID := cx.NewPackageID("benchmark", []model.Name{"symbols"}, "1.0.0")
	space := cx.NewSymbolSpace(*pkgID)
	symbol := model.NewVariableSymbol("value", true, false, false, diagnostics.NewBuiltinLocation())
	symbol.SetType(semtypes.Int)
	space.AddSymbol("value", &symbol)
	return model.NewExportedSymbolSpaces([]*model.SymbolSpace{space}, nil)
}

func benchTestPairs(b *testing.B) []test_util.TestCase {
	return test_util.GetTests(b, test_util.Bench, func(path string) bool { return true })
}

func BenchmarkBIRMarshal(b *testing.B) {
	for _, tc := range benchTestPairs(b) {
		fixture := compileForSerializationBench(b, tc)
		b.Run(tc.Name, func(b *testing.B) {
			for b.Loop() {
				if _, err := bircodec.Marshal(fixture.tyEnv, fixture.birPkg); err != nil {
					b.Fatalf("BIR Marshal: %v", err)
				}
			}
		})
	}
}

func BenchmarkBIRUnmarshal(b *testing.B) {
	for _, tc := range benchTestPairs(b) {
		fixture := compileForSerializationBench(b, tc)
		data, err := bircodec.Marshal(fixture.tyEnv, fixture.birPkg)
		if err != nil {
			b.Fatalf("BIR Marshal setup: %v", err)
		}
		b.Run(tc.Name, func(b *testing.B) {
			for b.Loop() {
				freshEnv := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
				freshCtx := context.NewCompilerContext(freshEnv)
				if _, err := bircodec.Unmarshal(freshCtx, data); err != nil {
					b.Fatalf("BIR Unmarshal: %v", err)
				}
			}
		})
	}
}

func BenchmarkSymbolMarshal(b *testing.B) {
	for _, tc := range benchTestPairs(b) {
		fixture := compileForSerializationBench(b, tc)
		b.Run(tc.Name, func(b *testing.B) {
			for b.Loop() {
				if _, err := symbolpool.Marshal(fixture.exportedSymbols, fixture.compilerEnv); err != nil {
					b.Fatalf("Symbol Marshal: %v", err)
				}
			}
		})
	}
}

func BenchmarkSymbolUnmarshal(b *testing.B) {
	for _, tc := range benchTestPairs(b) {
		fixture := compileForSerializationBench(b, tc)
		data, err := symbolpool.Marshal(fixture.exportedSymbols, fixture.compilerEnv)
		if err != nil {
			b.Fatalf("Symbol Marshal setup: %v", err)
		}
		b.Run(tc.Name, func(b *testing.B) {
			for b.Loop() {
				freshEnv := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
				if _, err := symbolpool.Unmarshal(freshEnv, data); err != nil {
					b.Fatalf("Symbol Unmarshal: %v", err)
				}
			}
		})
	}
}
