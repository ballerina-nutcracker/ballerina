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

package nodebuilder

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/st"
)

func GetCompilationUnit(cx *context.CompilerContext, syntaxTree *st.SyntaxTree) *ast.BLangCompilationUnit {
	builder := newNodeBuilder(cx)
	compilationUnit := builder.TransformModulePart(syntaxTree.RootNode.(*st.ModulePart))
	return compilationUnit.(*ast.BLangCompilationUnit)
}

// GetRecoveredCompilationUnit builds an AST while preserving malformed syntax as bad nodes.
func GetRecoveredCompilationUnit(cx *context.CompilerContext, syntaxTree *st.SyntaxTree) *ast.BLangCompilationUnit {
	builder := newRecoveringNodeBuilder(cx)
	compilationUnit := builder.TransformModulePart(syntaxTree.RootNode.(*st.ModulePart))
	return compilationUnit.(*ast.BLangCompilationUnit)
}

func ToPackageFromCompilationUnits(compilationUnits []*ast.BLangCompilationUnit) *ast.BLangPackage {
	pkg := ast.NewBLangPackage()
	for _, compilationUnit := range compilationUnits {
		packageID := compilationUnit.GetPackageID()
		if pkg.PackageID == nil {
			pkg.PackageID = packageID
		} else if packageID != nil && pkg.PackageID != packageID {
			panic("compilation units have different package IDs")
		}
		addCompilationUnitNodesToPackage(pkg, compilationUnit)
	}
	return pkg
}

func addCompilationUnitNodesToPackage(pkg *ast.BLangPackage, compilationUnit *ast.BLangCompilationUnit) {
	for _, node := range compilationUnit.GetTopLevelNodes() {
		switch node := node.(type) {
		case *ast.BLangImportPackage:
			pkg.Imports = append(pkg.Imports, *node)
		case *ast.BLangConstant:
			pkg.Constants = append(pkg.Constants, *node)
		case *ast.BLangService:
			pkg.Services = append(pkg.Services, *node)
		case *ast.BLangSimpleVariable:
			pkg.GlobalVars = append(pkg.GlobalVars, *node)
		case *ast.BLangFunction:
			if node.Name.GetValue() == "init" {
				pkg.InitFunction = node
			} else {
				pkg.Functions = append(pkg.Functions, *node)
			}
		case *ast.BLangTypeDefinition:
			pkg.TypeDefinitions = append(pkg.TypeDefinitions, *node)
		case *ast.BLangAnnotation:
			pkg.Annotations = append(pkg.Annotations, *node)
		case *ast.BLangXMLNS:
			pkg.XmlnsList = append(pkg.XmlnsList, *node)
		case *ast.BLangClassDefinition:
			pkg.ClassDefinitions = append(pkg.ClassDefinitions, *node)
		default:
			panic(fmt.Sprintf("unexpected top-level node type: %T", node))
		}
	}
}
