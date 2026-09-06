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

//go:build debug

package driver

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type debugDependencyResolver struct{}

func (debugDependencyResolver) Resolve(_ context.Context, request DependencyRequest) (
	fs.FS, string, *PackageSources, []diagnostics.Diagnostic, error,
) {
	descriptor := request.Descriptor
	if descriptor.Version == "" {
		descriptor.Version = "1.0.0"
	}
	module := &ModuleSources{ID: ModuleDescriptor{Package: descriptor, Name: descriptor.Name}}
	files := fstest.MapFS{}
	if descriptor.Org == "acme" {
		module.Documents = []string{"dependency.bal"}
		files["dependency.bal"] = &fstest.MapFile{Data: []byte("public function dependencyOnly() {}")}
	}
	return files, ".", &PackageSources{Descriptor: descriptor, Modules: []*ModuleSources{module}}, nil, nil
}

func TestDriverDebugBuffersAreOrderedAndDependenciesAreSuppressed(t *testing.T) {
	descriptor := PackageDescriptor{Org: "testorg", Name: "debugfixture", Version: "1.0.0"}
	sources := &PackageSources{Descriptor: descriptor, Modules: []*ModuleSources{{
		ID:            ModuleDescriptor{Package: descriptor, Name: descriptor.Name},
		Documents:     []string{"first.bal", "second.bal"},
		TestDocuments: []string{"test.bal"},
	}}}
	fsys := fstest.MapFS{
		"first.bal":  &fstest.MapFile{Data: []byte("import acme/dep; function firstOnly() {}")},
		"second.bal": &fstest.MapFile{Data: []byte("function secondOnly() {}")},
		"test.bal":   &fstest.MapFile{Data: []byte("function testOnly() {}")},
	}
	cx := NewContext(context.Background(), NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)))
	parsed, err := Parse(cx, fsys, ".", sources, parser.DebugOptions{DumpTokens: true, DumpSyntaxTree: true})
	if err != nil {
		t.Fatal(err)
	}
	root := parsed.Modules[0]
	ordered := string(root.Documents[0].ParserDebugOutput) + string(root.Documents[1].ParserDebugOutput) + string(root.TestDocuments[0].ParserDebugOutput)
	first, second, testIndex := strings.Index(ordered, "firstOnly"), strings.Index(ordered, "secondOnly"), strings.Index(ordered, "testOnly")
	if first < 0 || second <= first || testIndex <= second {
		t.Fatalf("root parser buffers are not in source/source/test index order: %d %d %d", first, second, testIndex)
	}
	resolved, err := ResolveDependencies(cx, parsed, debugDependencyResolver{})
	if err != nil {
		t.Fatal(err)
	}
	for _, module := range resolved.Modules {
		if module.ID.Package.Org != "acme" {
			continue
		}
		for _, document := range append(append([]*ParsedDocument(nil), module.Documents...), module.TestDocuments...) {
			if len(document.ParserDebugOutput) != 0 {
				t.Fatalf("dependency parser emitted debug output for %s", document.SourcePath)
			}
		}
	}
}
