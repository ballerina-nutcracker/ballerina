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
	"regexp"
	"strings"

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
	callerType, err := exportedType(compilerCtx, exported, "Caller")
	if err != nil {
		return nil, err
	}
	serviceConfig, ok := exported.GetAnnotationSymbol("ServiceConfig")
	if !ok {
		return nil, fmt.Errorf("ballerina/http compiler plugin: exported annotation ServiceConfig not found")
	}
	typeCtx := semtypes.ContextFrom(compilerCtx.GetTypeEnv())
	for _, service := range pkg.Services {
		if !isHTTPService(typeCtx, listenerType, service) {
			continue
		}
		for i := range service.AnnAttachments {
			if annotation := &service.AnnAttachments[i]; annotation.Symbol() == serviceConfig {
				validateServiceConfig(compilerCtx, annotation)
			}
		}
		for _, resource := range service.ResourceMethods {
			validateResource(compilerCtx, typeCtx, requestType, callerType, resource)
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
		attachedType := semtypes.Diff(attached.Type(), semtypes.Error)
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
	callerType semtypes.SemType,
	resource *ast.BLangResourceMethod,
) {
	requestParamCount := 0
	hasCaller := false
	for i := range resource.RequiredParams {
		param := &resource.RequiredParams[i]
		paramType := compilerCtx.SymbolType(param.Symbol())
		if semtypes.IsSubtype(typeCtx, paramType, requestType) {
			requestParamCount++
		}
		if semtypes.IsSubtype(typeCtx, paramType, callerType) {
			if hasCaller {
				compilerCtx.SemanticError(fmt.Sprintf("invalid multiple 'http:Caller' parameter: '%s'", param.Name.GetValue()),
					param.Name.GetPosition())
			}
			hasCaller = true
		}
	}
	if requestParamCount > 1 {
		compilerCtx.SemanticError("http resource method must have at most one http:Request parameter", resource.ParamListPos)
	}
	if hasCaller {
		validateCallerReturnType(compilerCtx, typeCtx, resource)
	}
	if resource.RestParam != nil {
		compilerCtx.Unimplemented("http resource method rest parameters", resource.RestParam.GetPosition())
	}
}

// validateCallerReturnType rejects a resource that responds through an http:Caller and can
// also return a value other than an error or nil.
func validateCallerReturnType(compilerCtx *context.CompilerContext, typeCtx semtypes.Context, resource *ast.BLangResourceMethod) {
	returnType := resource.GetReturnTypeDescriptor()
	if returnType == nil || returnType.TypeDescriptor == nil {
		return
	}
	ty := returnType.TypeDescriptor.Type()
	if semtypes.IsSubtype(typeCtx, ty, semtypes.Union(semtypes.Error, semtypes.Nil)) {
		return
	}
	compilerCtx.SemanticError(fmt.Sprintf("invalid resource method return type: can not use 'http:Caller' and return "+
		"'%s' from a resource : expected 'error' or nil", semtypes.ToString(typeCtx, ty)), returnType.TypeDescriptor.GetPosition())
}

// mediaTypeSubtypePrefix is jBallerina's MEDIA_TYPE_SUBTYPE_REGEX, matched against the whole prefix.
var mediaTypeSubtypePrefix = regexp.MustCompile(`^\w+(\s*\.\s*\w+)*(\s*\+\s*\w+)*$`)

// validateServiceConfig checks the @http:ServiceConfig fields jBallerina's compiler plugin
// validates. A service declaration is never a service contract type, so it cannot set basePath.
func validateServiceConfig(compilerCtx *context.CompilerContext, annotation *ast.BLangAnnotationAttachment) {
	mapping, ok := annotation.Expr.(*ast.BLangMappingConstructorExpr)
	if !ok {
		return
	}
	for _, field := range mapping.Fields {
		keyValue, ok := field.(*ast.BLangMappingKeyValueField)
		if !ok {
			continue
		}
		switch fieldName(keyValue.Key) {
		case "mediaTypeSubtypePrefix":
			literal, ok := keyValue.ValueExpr.(*ast.BLangLiteral)
			if !ok {
				continue
			}
			prefix, ok := literal.Value.(string)
			if !ok {
				continue
			}
			prefix = strings.TrimSpace(prefix)
			if !mediaTypeSubtypePrefix.MatchString(prefix) {
				compilerCtx.SemanticError(fmt.Sprintf("invalid media-type subtype '%s'", prefix), keyValue.GetPosition())
			} else if _, suffix, found := strings.Cut(prefix, "+"); found {
				compilerCtx.SemanticError(fmt.Sprintf("invalid media-type subtype prefix: subtype prefix should not have suffix '%s'",
					strings.TrimSpace(suffix)), keyValue.GetPosition())
			}
		case "basePath":
			compilerCtx.SemanticError("configuring base path in the 'http:ServiceConfig' annotation is not allowed for non service contract types",
				keyValue.GetPosition())
		}
	}
}

func fieldName(key *ast.BLangMappingKey) string {
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		name, _ := expr.Value.(string)
		return name
	case *ast.BLangVarRef:
		return expr.VariableName.GetValue()
	}
	return ""
}
