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
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type implicitResolver struct{}

func (implicitResolver) Resolve(_ context.Context, request DependencyRequest) (
	fs.FS, string, *PackageSources, []diagnostics.Diagnostic, error,
) {
	module := &ModuleSources{ID: ModuleDescriptor{Package: request.Descriptor, Name: request.Descriptor.Name}}
	return fstest.MapFS{}, ".", &PackageSources{Descriptor: request.Descriptor, Modules: []*ModuleSources{module}}, nil, nil
}

func TestImplicitLookupKeysExcludeRuntimeAndContainNoDuplicates(t *testing.T) {
	env := NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	cx := NewContext(context.Background(), env)
	root := PackageDescriptor{Org: "testorg", Name: "root", Version: "1.0.0"}
	sources := &PackageSources{Descriptor: root, Modules: []*ModuleSources{{
		ID: ModuleDescriptor{Package: root, Name: root.Name},
	}}}
	parsed, err := Parse(cx, fstest.MapFS{}, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = ResolveDependencies(cx, parsed, implicitResolver{})
	if err != nil {
		t.Fatal(err)
	}
	for _, module := range parsed.Modules {
		astModule := ToAST(cx, module)
		if astModule == nil || ResolvePublicNodes(cx, astModule) == nil {
			t.Fatalf("failed to publish %v: %v", module.ID, cx.Diagnostics())
		}
	}
	env.mu.Lock()
	keys := make([]string, 0, len(env.implicitSymbols))
	for key := range env.implicitSymbols {
		keys = append(keys, key)
	}
	env.mu.Unlock()
	slices.Sort(keys)
	want := []string{
		"lang.__internal", "lang.array", "lang.boolean", "lang.decimal", "lang.error", "lang.float",
		"lang.int", "lang.map", "lang.object", "lang.string", "lang.value", "lang.xml",
	}
	if !slices.Equal(keys, want) {
		t.Fatalf("implicit lookup keys = %v, want %v", keys, want)
	}
}
