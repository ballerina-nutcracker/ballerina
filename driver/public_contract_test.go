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

package driver

import (
	"context"
	"testing"
	"testing/fstest"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/model/symbolpool"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestResolvePublicNodesUsesOneCompilerContext(t *testing.T) {
	module, cx := publicTestAST(t, "public function main() {}")
	creations := 0
	cx.newContextHook = func(*compilercontext.CompilerContext) { creations++ }
	if ResolvePublicNodes(cx, module) == nil {
		t.Fatalf("public resolution failed: %v", cx.Diagnostics())
	}
	if creations != 1 {
		t.Fatalf("compiler context creations = %d, want exactly 1 for ResolvePublicNodes", creations)
	}
}

func TestPublishedDependentFunctionSymbolsRoundTrip(t *testing.T) {
	module, cx := publicTestAST(t, "public function inferred(int val, typedesc retTy = <>) returns retTy = external;")
	if ResolvePublicNodes(cx, module) == nil {
		t.Fatalf("public resolution failed: %v", cx.Diagnostics())
	}
	identifier := model.PackageIdentifier{Organization: module.ID.Package.Org, Package: module.ID.Name, Version: module.ID.Package.Version}
	exported, ok := cx.ExportedSymbols()[identifier]
	if !ok {
		t.Fatalf("driver did not publish symbols for %+v", identifier)
	}
	if _, ok := exported.GetSymbol("inferred"); !ok {
		t.Fatal("compiled dependent function is absent from exported symbols")
	}
	data, err := symbolpool.Marshal(exported, cx.env.compiler)
	if err != nil {
		t.Fatal(err)
	}
	freshEnv := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	roundTripped, err := symbolpool.Unmarshal(freshEnv, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := roundTripped.GetSymbol("inferred"); !ok {
		t.Fatal("dependent function metadata did not survive exported-symbol round-trip")
	}
}

func publicTestAST(t *testing.T, source string) (*ASTModule, *Context) {
	t.Helper()
	descriptor := PackageDescriptor{Org: "testorg", Name: "publicfixture", Version: "1.0.0"}
	sources := &PackageSources{Descriptor: descriptor, Modules: []*ModuleSources{{
		ID: ModuleDescriptor{Package: descriptor, Name: descriptor.Name}, Documents: []string{"main.bal"},
	}}}
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)
	cx := NewContext(context.Background(), NewEnv(env))
	parsed, err := Parse(cx, fstest.MapFS{"main.bal": &fstest.MapFile{Data: []byte(source)}}, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	module := ToAST(cx, parsed.Modules[0])
	if module == nil {
		t.Fatalf("AST conversion failed: %v", cx.Diagnostics())
	}
	return module, cx
}
