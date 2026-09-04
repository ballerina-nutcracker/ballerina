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

// Package compilerplugin validates HTTP service resource signatures.
package compilerplugin

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// ValidateService validates services attached to an http:Listener.
func ValidateService(
	compilerCtx *context.CompilerContext,
	exported model.ExportedSymbolSpace,
	pkg *ast.BLangPackage,
) (*ast.BLangPackage, error) {
	listenerType, err := exportedType(compilerCtx, exported, "Listener")
	if err != nil {
		return nil, err
	}
	requestType, err := exportedType(compilerCtx, exported, "Request")
	if err != nil {
		return nil, err
	}
	typeCtx := semtypes.ContextFrom(compilerCtx.GetTypeEnv())
	for _, service := range pkg.Services {
		if !isHTTPService(typeCtx, listenerType, service) {
			continue
		}
		for _, resource := range service.ResourceMethods {
			validateResource(compilerCtx, typeCtx, requestType, resource)
		}
	}
	return pkg, nil
}

func exportedType(compilerCtx *context.CompilerContext, exported model.ExportedSymbolSpace, name string) (semtypes.SemType, error) {
	ref, ok := exported.GetSymbol(name)
	if !ok || compilerCtx.SymbolKind(ref) != model.SymbolKindType {
		return semtypes.SemType{}, fmt.Errorf("ballerina/http compiler plugin: exported type %s not found", name)
	}
	return compilerCtx.SymbolType(ref), nil
}

func isHTTPService(typeCtx semtypes.Context, listenerType semtypes.SemType, service *ast.BLangService) bool {
	for _, attached := range service.AttachedExprs {
		determinedType := attached.GetDeterminedType()
		if semtypes.IsNever(determinedType) {
			continue
		}
		attachedType := semtypes.Diff(determinedType, semtypes.Error)
		if semtypes.IsNever(attachedType) {
			continue
		}
		if semtypes.IsSubtype(typeCtx, attachedType, listenerType) {
			return true
		}
	}
	return false
}

func validateResource(
	compilerCtx *context.CompilerContext,
	typeCtx semtypes.Context,
	requestType semtypes.SemType,
	resource *ast.BLangResourceMethod,
) {
	if resource.RestParam != nil || len(resource.RequiredParams) > 1 {
		compilerCtx.SemanticError("http resource method must have at most one parameter", resource.ParamListPos)
		return
	}
	if len(resource.RequiredParams) == 0 {
		return
	}
	param := &resource.RequiredParams[0]
	if !semtypes.IsSubtype(typeCtx, compilerCtx.SymbolType(param.Symbol()), requestType) {
		compilerCtx.SemanticError(
			"http resource method parameter must be a subtype of http:Request",
			param.GetPosition(),
		)
	}
}
