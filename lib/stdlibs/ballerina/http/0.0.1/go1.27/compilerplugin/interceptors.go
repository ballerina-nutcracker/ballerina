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

package compilerplugin

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const (
	requestInterceptor       = "RequestInterceptor"
	responseInterceptor      = "ResponseInterceptor"
	requestErrorInterceptor  = "RequestErrorInterceptor"
	responseErrorInterceptor = "ResponseErrorInterceptor"
)

var interceptorKinds = []string{requestInterceptor, responseInterceptor, requestErrorInterceptor, responseErrorInterceptor}

// remoteMethodNames are the remote methods of the response-side interceptor kinds.
var remoteMethodNames = map[string]string{responseInterceptor: "interceptResponse",
	responseErrorInterceptor: "interceptResponseError"}

// interceptorValidator checks interceptor service classes.
type interceptorValidator struct {
	compilerCtx *context.CompilerContext
	typeCtx     semtypes.Context
	params      paramTypes
	response    semtypes.SemType
	kinds       map[model.SymbolRef]string
	returnType  semtypes.SemType
}

func newInterceptorValidator(compilerCtx *context.CompilerContext, exported model.ExportedSymbolSpace,
	typeCtx semtypes.Context, params paramTypes) (*interceptorValidator, error) {
	kinds := make(map[model.SymbolRef]string, len(interceptorKinds))
	for _, kind := range interceptorKinds {
		ref, ok := exported.GetSymbol(kind)
		if !ok {
			return nil, fmt.Errorf("ballerina/http compiler plugin: exported type %s not found", kind)
		}
		kinds[ref] = kind
	}
	returnType := semtypes.Union(semtypes.CreateAnydata(typeCtx), semtypes.Union(semtypes.Error, semtypes.Nil))
	for _, name := range []string{"Response", "StatusCodeResponse", "NextService"} {
		ty, err := exportedType(compilerCtx, exported, name)
		if err != nil {
			return nil, err
		}
		returnType = semtypes.Union(returnType, ty)
	}
	response, err := exportedType(compilerCtx, exported, "Response")
	if err != nil {
		return nil, err
	}
	return &interceptorValidator{compilerCtx: compilerCtx, typeCtx: typeCtx, params: params, response: response,
		kinds: kinds, returnType: returnType}, nil
}

// validate checks a service class that includes an interceptor type: a request-side interceptor
// must have one resource method and no remote method, a response-side one only its remote
// method. A class may include only one interceptor type.
func (v *interceptorValidator) validate(class *ast.BLangClassDefinition) {
	if !class.IsService() {
		return
	}
	kind := ""
	for i, inclusion := range class.Inclusions {
		name, ok := v.kinds[inclusion]
		if !ok {
			continue
		}
		if kind != "" {
			v.compilerCtx.SemanticError(fmt.Sprintf("invalid multiple interceptor type reference: 'http:%s'", name),
				class.InclusionPositions[i])
			return
		}
		kind = name
	}
	if kind == "" {
		return
	}
	requestSide := kind == requestInterceptor || kind == requestErrorInterceptor
	for i, resource := range class.ResourceMethods {
		if !requestSide {
			v.compilerCtx.SemanticError(fmt.Sprintf("resource function is not allowed in %s", kind), resource.GetPosition())
		}
		if i > 0 {
			v.compilerCtx.SemanticError("invalid multiple interceptor resource functions", resource.GetPosition())
		}
	}
	var remote *ast.BLangFunction
	for _, method := range sortedMethods(class) {
		if !method.IsRemote() {
			continue
		}
		name := model.StripRemotePrefix(method.Name.GetValue())
		switch {
		case requestSide:
			v.compilerCtx.SemanticError(fmt.Sprintf("remote function is not allowed in %s", kind), method.GetPosition())
		case name != remoteMethodNames[kind]:
			v.compilerCtx.SemanticError(fmt.Sprintf("invalid remote function : '%s'. %s can have only '%s' remote function",
				name, kind, remoteMethodNames[kind]), method.GetPosition())
		case remote == nil:
			remote = method
		}
	}
	switch {
	case requestSide && len(class.ResourceMethods) == 0:
		v.compilerCtx.SemanticError(fmt.Sprintf("%s must have a resource method", kind), class.GetPosition())
	case requestSide:
		v.validateResource(class.ResourceMethods[0], kind == requestErrorInterceptor)
	case remote == nil:
		v.compilerCtx.SemanticError(fmt.Sprintf("%s must have the remote method : '%s'", kind, remoteMethodNames[kind]),
			class.GetPosition())
	default:
		v.validateRemote(remote, kind)
	}
}

// sortedMethods returns the methods of class in source order.
func sortedMethods(class *ast.BLangClassDefinition) []*ast.BLangFunction {
	methods := slices.Collect(maps.Values(class.Methods))
	slices.SortFunc(methods, func(a, b *ast.BLangFunction) int {
		posA, posB := a.GetPosition(), b.GetPosition()
		return posA.StartOffset() - posB.StartOffset()
	})
	return methods
}

func (v *interceptorValidator) validateResource(resource *ast.BLangResourceMethod, errorInterceptor bool) {
	if len(resource.AnnAttachments) > 0 {
		annotation := &resource.AnnAttachments[0]
		name := annotation.AnnotationName.GetValue()
		if alias := annotation.PkgAlias.GetValue(); alias != "" {
			name = alias + ":" + name
		}
		v.compilerCtx.SemanticError(fmt.Sprintf("invalid annotation '%s': annotations are not supported for "+
			"interceptor resource functions", name), annotation.GetPosition())
	}
	if errorInterceptor {
		v.validateDefaultResource(resource)
	}
	hasError := validateResource(v.compilerCtx, v.typeCtx, v.params, resource)
	if errorInterceptor && !hasError {
		v.compilerCtx.SemanticError("resource function should have the mandatory parameter 'error'", resource.GetPosition())
	}
	v.validateReturnType(resource.GetReturnTypeDescriptor(), "resource")
}

// validateDefaultResource checks that a request error interceptor resource is the default
// resource of the default path, '[string... path]'.
func (v *interceptorValidator) validateDefaultResource(resource *ast.BLangResourceMethod) {
	if len(resource.ResourcePath) == 0 {
		v.compilerCtx.SemanticError("invalid interceptor resource path: expected default resource path: "+
			"'[string... path]', but found '.'", resource.GetPosition())
	} else if segment := &resource.ResourcePath[0]; segment.Kind != ast.ResourcePathSegmentParamRest ||
		!semtypes.IsSameType(v.typeCtx, segment.ParamType.(ast.BLangNode).(ast.NodeWithType).Type(), semtypes.String) {
		v.compilerCtx.SemanticError(fmt.Sprintf("invalid interceptor resource path: expected default resource path: "+
			"'[string... path]', but found '%s'", v.segmentText(segment)), segment.GetPosition())
	}
	if accessor := resource.Name.GetValue(); !strings.Contains(accessor, "default") {
		v.compilerCtx.SemanticError(fmt.Sprintf("invalid interceptor resource method: expected default resource "+
			"method: 'default', but found '%s'", accessor), resource.Name.GetPosition())
	}
}

func (v *interceptorValidator) segmentText(segment *ast.BLangResourcePathSegment) string {
	if segment.Kind == ast.ResourcePathSegmentName {
		return segment.Name
	}
	ellipsis := ""
	if segment.Kind == ast.ResourcePathSegmentParamRest {
		ellipsis = "..."
	}
	return fmt.Sprintf("[%s%s %s]", semtypes.ToString(v.typeCtx, segment.ParamType.(ast.BLangNode).(ast.NodeWithType).Type()),
		ellipsis, segment.Name)
}

// validateRemote checks the parameters and return type of the remote method of a response-side
// interceptor: it binds only an http:Caller, http:Request, http:Response, http:RequestContext and,
// in a response error interceptor, the mandatory error, each at most once.
func (v *interceptorValidator) validateRemote(method *ast.BLangFunction, kind string) {
	errorInterceptor := kind == responseErrorInterceptor
	seen := map[string]bool{}
	for i := range method.RequiredParams {
		param := &method.RequiredParams[i]
		paramType := v.compilerCtx.SymbolType(param.Symbol())
		label := ""
		switch {
		case semtypes.IsSubtype(v.typeCtx, paramType, v.params.caller):
			label = "http:Caller"
		case semtypes.IsSubtype(v.typeCtx, paramType, v.params.request):
			label = "http:Request"
		case semtypes.IsSubtype(v.typeCtx, paramType, v.response):
			label = "http:Response"
		case semtypes.IsSubtype(v.typeCtx, paramType, v.params.context):
			label = "http:RequestContext"
		case errorInterceptor && isErrorTypeDescriptor(param):
			label = "error"
		default:
			v.compilerCtx.SemanticError(fmt.Sprintf("invalid parameter type: '%s' in '%s' remote method",
				semtypes.ToString(v.typeCtx, paramType), remoteMethodNames[kind]), param.Name.GetPosition())
			continue
		}
		if seen[label] {
			v.compilerCtx.SemanticError(fmt.Sprintf("invalid multiple '%s' parameter: '%s'", label, param.Name.GetValue()),
				param.Name.GetPosition())
		}
		seen[label] = true
	}
	if errorInterceptor && !seen["error"] {
		v.compilerCtx.SemanticError("remote function should have the mandatory parameter 'error'", method.GetPosition())
	}
	v.validateReturnType(method.GetReturnTypeDescriptor(), "remote")
}

func (v *interceptorValidator) validateReturnType(returnType *ast.BLangReturnTypeDescriptor, kind string) {
	if returnType == nil || returnType.TypeDescriptor == nil {
		return
	}
	if ty := returnType.TypeDescriptor.Type(); !semtypes.IsSubtype(v.typeCtx, ty, v.returnType) {
		v.compilerCtx.SemanticError(fmt.Sprintf("invalid interceptor %s method return type: expected "+
			"'anydata|http:Response|http:StatusCodeResponse|http:NextService|error?', but found '%s'",
			kind, semtypes.ToString(v.typeCtx, ty)), returnType.TypeDescriptor.GetPosition())
	}
	if len(returnType.AnnAttachments) > 0 {
		v.compilerCtx.SemanticError("return type annotation is not supported in interceptor service",
			returnType.AnnAttachments[0].GetPosition())
	}
}
