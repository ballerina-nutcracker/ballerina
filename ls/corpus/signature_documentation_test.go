// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package corpus

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestSignatureDocumentationExternalIdentity(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name  string
		count int
	}{{"docs-imported", 5}, {"docs-stdlib", 1}} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			fixturePath := "signature/testdata/" + scenario.name + ".signatureHelp.json"
			assertExternalDocumentationIdentity(t, fixturePath, scenario.count)
		})
	}
}

func assertExternalDocumentationIdentity(t *testing.T, fixturePath string, expectedCount int) {
	t.Helper()
	content, err := fs.ReadFile(os.DirFS("."), fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture transcript
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	var uri workspace.DocumentURI
	var text string
	var options []workspace.Option
	if len(fixture.Sources) > 0 {
		root := materializeSources(t, fixture.Sources)
		uri, err = workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
		text = fixture.Sources["main.bal"]
		options = append(options, workspace.WithRepositories([]projects.Repository{projects.NewFileSystemRepository(os.DirFS(filepath.Join(root, "repository")), ".")}))
	} else {
		uri, err = workspace.FileURIFromPath(filepath.Join(t.TempDir(), "signature68-identity.bal"))
		source, readErr := fs.ReadFile(os.DirFS("."), path.Join(path.Dir(fixturePath), fixture.Source))
		if readErr != nil {
			t.Fatal(readErr)
		}
		text = string(source)
	}
	if err != nil {
		t.Fatal(err)
	}
	bus := event.New()
	defer bus.Close()
	service := workspace.New(os.DirFS, bus, options...)
	compiler := compile.New(service, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	if _, err := service.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeOpen, URI: uri, Text: text, Version: 1, LanguageID: "ballerina"}); err != nil {
		t.Fatal(err)
	}
	sm, ok := compiler.SealedModuleFor(context.Background(), uri)
	if !ok {
		t.Fatal("no sealed module for documentation identity fixture")
	}
	visitor := &documentationCalls{sm: sm, refs: make([]model.SymbolRef, 0)}
	ast.Walk(visitor, sm.PackageNode())
	if len(visitor.refs) != expectedCount {
		t.Fatalf("got %d callees, want %d", len(visitor.refs), expectedCount)
	}
	for _, ref := range visitor.refs {
		if ref.IsEmpty() {
			t.Fatal("empty callee ref")
		}
		id := sm.Context().SymbolPackage(ref)
		projection, ok := sm.ExternalModuleProjection(id.Organization, id.Package)
		if !ok || projection.PackageNode == nil {
			t.Fatalf("missing external projection: %v", id)
		}
		declarations := &documentationDeclarations{functions: make(map[model.SymbolRef]*ast.BLangFunction)}
		ast.Walk(declarations, projection.PackageNode)
		function := declarations.functions[ref]
		if function == nil || function.Symbol() != ref || function.GetMarkdownDocumentationAttachment() == nil {
			t.Fatalf("callee %v has no exactly-equal documented declaration", ref)
		}
		callLocation, declarationLocation := sm.Context().SymbolLocation(ref), function.GetPosition()
		if callLocation.FileIndex() != declarationLocation.FileIndex() || callLocation.StartOffset() < declarationLocation.StartOffset() || callLocation.EndOffset() > declarationLocation.EndOffset() {
			t.Fatalf("callee %v declaration location mismatch", ref)
		}
		t.Logf("%s/%s %s: callsite=%v declaration=%v equal=true location=%v", id.Organization, id.Package, sm.Context().SymbolName(ref), ref, function.Symbol(), callLocation)
	}
}

type documentationCalls struct {
	sm   compile.SealedModule
	refs []model.SymbolRef
}

func (v *documentationCalls) Visit(node ast.BLangNode) ast.Visitor {
	var raw model.Symbol
	switch node := node.(type) {
	case *ast.BLangInvocation:
		raw = node.RawSymbol
	case *ast.BLangRemoteMethodCallAction:
		raw = node.RawSymbol
	case *ast.BLangNewExpression:
		cx := v.sm.Context()
		ty := node.Type()
		if !node.ClassSymbol.IsEmpty() {
			ty = cx.SymbolType(node.ClassSymbol)
		}
		if !semtypes.IsZero(ty) {
			atom, unique := semtypes.ObjectPositiveAtom(semtypes.ContextFrom(cx.GetTypeEnv()), ty)
			if unique {
				if table, ok := cx.ObjectMethodTable(atom); ok {
					if ref, ok := table.Methods["init"]; ok {
						v.refs = append(v.refs, ref)
					}
				}
			}
		}
	}
	if ref, ok := raw.(*model.SymbolRef); ok && ref != nil {
		v.refs = append(v.refs, *ref)
	}
	return v
}
func (v *documentationCalls) VisitTypeData(*ast.TypeData) ast.Visitor { return nil }

type documentationDeclarations struct {
	functions map[model.SymbolRef]*ast.BLangFunction
}

func (v *documentationDeclarations) Visit(node ast.BLangNode) ast.Visitor {
	if function, ok := node.(*ast.BLangFunction); ok {
		v.functions[function.Symbol()] = function
		return nil
	}
	return v
}
func (v *documentationDeclarations) VisitTypeData(*ast.TypeData) ast.Visitor { return nil }
