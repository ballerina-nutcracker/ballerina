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

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const requestInterceptor = "RequestInterceptor"

var interceptorKinds = []string{requestInterceptor, "ResponseInterceptor", "RequestErrorInterceptor", "ResponseErrorInterceptor"}

// interceptorValidator checks interceptor service classes.
type interceptorValidator struct {
	compilerCtx *context.CompilerContext
	typeCtx     semtypes.Context
	params      paramTypes
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
	return &interceptorValidator{compilerCtx: compilerCtx, typeCtx: typeCtx, params: params, kinds: kinds,
		returnType: returnType}, nil
}

// validate checks a service class that includes a request interceptor type: it must have one
// resource method and no remote method. A class may include only one interceptor type.
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
	if kind != requestInterceptor {
		return
	}
	for _, name := range slices.Sorted(maps.Keys(class.Methods)) {
		if method := class.Methods[name]; method.IsRemote() {
			v.compilerCtx.SemanticError(fmt.Sprintf("remote function is not allowed in %s", kind), method.GetPosition())
		}
	}
	if len(class.ResourceMethods) == 0 {
		v.compilerCtx.SemanticError(fmt.Sprintf("%s must have a resource method", kind), class.GetPosition())
		return
	}
	for _, extra := range class.ResourceMethods[1:] {
		v.compilerCtx.SemanticError("invalid multiple interceptor resource functions", extra.GetPosition())
	}
	v.validateResource(class.ResourceMethods[0])
}

func (v *interceptorValidator) validateResource(resource *ast.BLangResourceMethod) {
	if len(resource.AnnAttachments) > 0 {
		annotation := &resource.AnnAttachments[0]
		name := annotation.AnnotationName.GetValue()
		if alias := annotation.PkgAlias.GetValue(); alias != "" {
			name = alias + ":" + name
		}
		v.compilerCtx.SemanticError(fmt.Sprintf("invalid annotation '%s': annotations are not supported for "+
			"interceptor resource functions", name), annotation.GetPosition())
	}
	validateResource(v.compilerCtx, v.typeCtx, v.params, resource)
	returnType := resource.GetReturnTypeDescriptor()
	if returnType == nil || returnType.TypeDescriptor == nil {
		return
	}
	if len(returnType.AnnAttachments) > 0 {
		v.compilerCtx.SemanticError("return type annotation is not supported in interceptor service",
			returnType.AnnAttachments[0].GetPosition())
	}
	if ty := returnType.TypeDescriptor.Type(); !semtypes.IsSubtype(v.typeCtx, ty, v.returnType) {
		v.compilerCtx.SemanticError(fmt.Sprintf("invalid interceptor resource method return type: expected "+
			"'anydata|http:Response|http:StatusCodeResponse|http:NextService|error?', but found '%s'",
			semtypes.ToString(v.typeCtx, ty)), returnType.TypeDescriptor.GetPosition())
	}
}
