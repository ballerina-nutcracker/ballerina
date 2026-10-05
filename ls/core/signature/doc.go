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

package signature

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type parameterDocumentation struct {
	Name        string
	Type        string
	Description string
}

func documentation(sm compile.SealedModule, ref model.SymbolRef) *ast.BLangMarkdownDocumentation {
	if ref.IsEmpty() {
		return nil
	}
	cx := sm.Context()
	pkg := sm.PackageNode()
	if pkg == nil || pkg.PackageID == nil {
		return nil
	}
	id := cx.SymbolPackage(ref)
	if model.PackageIdentifierFromID(pkg.PackageID) != id {
		projection, ok := sm.ExternalModuleProjection(id.Organization, id.Package)
		if !ok || projection.PackageNode == nil || projection.PackageNode.PackageID == nil ||
			model.PackageIdentifierFromID(projection.PackageNode.PackageID) != id {
			return nil
		}
		pkg = projection.PackageNode
	}
	loc := cx.SymbolLocation(ref)
	if !locationHasUsableOffsets(loc) {
		return nil
	}
	finder := &declarationFinder{location: loc, ref: ref, declaration: nil, ambiguous: false}
	ast.Walk(finder, pkg)
	if finder.declaration == nil || finder.ambiguous {
		return nil
	}
	return finder.declaration.GetMarkdownDocumentationAttachment()
}

type declarationFinder struct {
	location    ast.Location
	ref         model.SymbolRef
	declaration *ast.BLangFunction
	ambiguous   bool
}

func (f *declarationFinder) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	loc := node.GetPosition()
	if locationHasUsableOffsets(loc) && (loc.FileIndex() != f.location.FileIndex() || !rangeWithin(f.location, loc)) {
		return nil
	}
	if function, ok := node.(*ast.BLangFunction); ok {
		if locationHasUsableOffsets(loc) && function.Symbol() == f.ref {
			if f.declaration != nil && f.declaration != function {
				f.ambiguous = true
			}
			f.declaration = function
		}
		return nil
	}
	return f
}

func (f *declarationFinder) VisitTypeData(*ast.TypeData) ast.Visitor { return nil }

func parameterDescriptions(doc *ast.BLangMarkdownDocumentation) map[string]string {
	result := make(map[string]string)
	if doc != nil {
		for name, parameter := range doc.GetParameterDocumentations() {
			result[name] = strings.TrimSpace(strings.Join(parameter.ParameterDocumentationLines, "\n"))
		}
	}
	return result
}

func description(doc *ast.BLangMarkdownDocumentation) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(doc.GetDocumentation())
}
