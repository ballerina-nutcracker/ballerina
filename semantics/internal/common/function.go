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

package common

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type FunctionDecl interface {
	ast.InvokableNode
	ast.FunctionSignature
	ast.BNodeWithSymbol
	Scope() model.Scope
	IsIsolated() bool
	IsTransactional() bool
	FuncSymbolFlags() model.FuncSymbolFlags
}

func PackageFunctionDecls(pkg *ast.BLangPackage) []FunctionDecl {
	var fns []FunctionDecl
	for i := range pkg.Functions {
		fns = append(fns, pkg.Functions[i])
	}
	if pkg.InitFunction != nil {
		fns = append(fns, pkg.InitFunction)
	}
	for i := range pkg.ClassDefinitions {
		classDef := pkg.ClassDefinitions[i]
		fns = appendMethods(fns, classDef.InitFunction, classDef.Methods, classDef.ResourceMethods)
	}
	for i := range pkg.Services {
		service := pkg.Services[i]
		fns = appendMethods(fns, service.InitFunction, service.Methods, service.ResourceMethods)
	}
	return fns
}

func appendMethods(fns []FunctionDecl, initFn *ast.BLangFunction, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod) []FunctionDecl {
	if initFn != nil {
		fns = append(fns, initFn)
	}
	for _, method := range methods {
		fns = append(fns, method)
	}
	for _, method := range resourceMethods {
		fns = append(fns, method)
	}
	return fns
}
