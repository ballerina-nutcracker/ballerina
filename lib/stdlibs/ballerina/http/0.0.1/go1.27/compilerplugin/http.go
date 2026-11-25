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
	"slices"
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
	contextType, err := exportedType(compilerCtx, exported, "RequestContext")
	if err != nil {
		return nil, err
	}
	serviceConfig, ok := exported.GetAnnotationSymbol("ServiceConfig")
	if !ok {
		return nil, fmt.Errorf("ballerina/http compiler plugin: exported annotation ServiceConfig not found")
	}
	typeCtx := semtypes.ContextFrom(compilerCtx.GetTypeEnv())
	params := paramTypes{request: requestType, caller: callerType, context: contextType}
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
			validateResource(compilerCtx, typeCtx, params, resource)
			if params.hasCaller(compilerCtx, typeCtx, resource) {
				validateCallerReturnType(compilerCtx, typeCtx, resource)
			}
		}
	}
	interceptors, err := newInterceptorValidator(compilerCtx, exported, typeCtx, params)
	if err != nil {
		return nil, err
	}
	for _, class := range pkg.ClassDefinitions {
		interceptors.validate(class)
	}
	return pkg, nil
}

// paramTypes are the types of the resource parameters the dispatcher binds itself.
type paramTypes struct {
	request semtypes.SemType
	caller  semtypes.SemType
	context semtypes.SemType
}

func (p paramTypes) hasCaller(compilerCtx *context.CompilerContext, typeCtx semtypes.Context,
	resource *ast.BLangResourceMethod) bool {
	return slices.ContainsFunc(resource.RequiredParams, func(param ast.BLangVariable) bool {
		return semtypes.IsSubtype(typeCtx, compilerCtx.SymbolType(param.Symbol()), p.caller)
	})
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

// validateResource checks the parameters of resource and reports whether it has an error
// parameter.
func validateResource(
	compilerCtx *context.CompilerContext,
	typeCtx semtypes.Context,
	types paramTypes,
	resource *ast.BLangResourceMethod,
) bool {
	requestParamCount := 0
	hasCaller, hasContext, hasError := false, false, false
	for i := range resource.RequiredParams {
		param := &resource.RequiredParams[i]
		paramType := compilerCtx.SymbolType(param.Symbol())
		if semtypes.IsSubtype(typeCtx, paramType, semtypes.Error) {
			if !isErrorTypeDescriptor(param) {
				compilerCtx.SemanticError(fmt.Sprintf("invalid resource parameter type: '%s'",
					semtypes.ToString(typeCtx, paramType)), param.Name.GetPosition())
				continue
			}
			if hasError {
				compilerCtx.SemanticError(fmt.Sprintf("invalid multiple 'error' parameter: '%s'", param.Name.GetValue()),
					param.Name.GetPosition())
			}
			hasError = true
		}
		if semtypes.IsSubtype(typeCtx, paramType, types.request) {
			requestParamCount++
		}
		if semtypes.IsSubtype(typeCtx, paramType, types.caller) {
			if hasCaller {
				compilerCtx.SemanticError(fmt.Sprintf("invalid multiple 'http:Caller' parameter: '%s'", param.Name.GetValue()),
					param.Name.GetPosition())
			}
			hasCaller = true
		}
		if semtypes.IsSubtype(typeCtx, paramType, types.context) {
			if hasContext {
				compilerCtx.SemanticError(fmt.Sprintf("invalid multiple 'http:RequestContext' parameter: '%s'",
					param.Name.GetValue()), param.Name.GetPosition())
			}
			hasContext = true
		}
	}
	if requestParamCount > 1 {
		compilerCtx.SemanticError("http resource method must have at most one http:Request parameter", resource.ParamListPos)
	}
	if resource.RestParam != nil {
		compilerCtx.Unimplemented("http resource method rest parameters", resource.RestParam.GetPosition())
	}
	return hasError
}

// isErrorTypeDescriptor reports whether param is declared with an error type descriptor, the only
// form jBallerina binds an error to; a named error type is a type reference.
func isErrorTypeDescriptor(param *ast.BLangVariable) bool {
	_, ok := param.TypeNode().(*ast.BLangErrorTypeNode)
	return ok
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
