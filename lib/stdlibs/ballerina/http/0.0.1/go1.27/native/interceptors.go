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

package native

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	noNextService     = "no next service to be returned"
	nextMismatch      = "next interceptor service did not match with the configuration"
	targetMismatch    = "target service did not match with the configuration"
	noPipeline        = "request context object does not contain the configured interceptors"
	pipelineSlot      = "$pipeline"
	unsupportedKind   = "response and error interceptors are not yet supported"
	registrationError = "interceptor service registration failed: "
)

// interceptorTypes are the types the dispatcher checks services, interceptors and resource
// parameters against.
type interceptorTypes struct {
	interceptable        semtypes.SemType
	requestInterceptor   semtypes.SemType
	nextService          semtypes.SemType
	requestContext       semtypes.SemType
	createRequestContext extern.FunctionHandle
}

func resolveInterceptorTypes(ctx *extern.Context) (interceptorTypes, error) {
	handle, found := ctx.LookupFunction(orgName, moduleName, "interceptorTypes")
	signature, ok := ctx.FunctionSignature(handle)
	if !found || !ok || len(signature.Params) != 3 {
		return interceptorTypes{}, fmt.Errorf("http: cannot resolve the interceptor types from 'interceptorTypes'")
	}
	createContext, contextSignature, err := resolveFactory(ctx, "createRequestContext")
	if err != nil {
		return interceptorTypes{}, err
	}
	return interceptorTypes{
		interceptable:        signature.Params[0].Type,
		requestInterceptor:   signature.Params[1].Type,
		nextService:          signature.Params[2].Type,
		requestContext:       contextSignature.ReturnType,
		createRequestContext: createContext,
	}, nil
}

func isOfType(tc semtypes.Context, v values.BalValue, ty semtypes.SemType) bool {
	return semtypes.IsSubtype(tc, values.SemTypeForValue(v), ty)
}

// createInterceptors returns the request interceptors svcObj engages: those its
// createInterceptors method returns when it is an http:InterceptableService.
func createInterceptors(ctx *extern.Context, types interceptorTypes, svcObj *values.Object) ([]*values.Object, error) {
	tc := ctx.TypeCtx()
	if !isOfType(tc, svcObj, types.interceptable) {
		return nil, nil
	}
	handle, ok := ctx.LookupObjectMethod(svcObj, "createInterceptors")
	if !ok {
		return nil, nil
	}
	result, err := ctx.InvokeMethod(handle, []values.BalValue{svcObj})
	if err != nil {
		return nil, err
	}
	elements := []values.BalValue{result}
	if list, isList := result.(*values.List); isList {
		elements = make([]values.BalValue, list.Len())
		for i := range list.Len() {
			elements[i] = list.Get(i)
		}
	}
	interceptors := make([]*values.Object, len(elements))
	for i, element := range elements {
		if !isOfType(tc, element, types.requestInterceptor) {
			return nil, errors.New(unsupportedKind)
		}
		interceptors[i] = element.(*values.Object)
	}
	return interceptors, nil
}

// requestPipeline is the native state of the http:RequestContext of a dispatched request: the
// request interceptors of its service, in order, followed by the service itself.
type requestPipeline struct {
	mu           sync.Mutex
	interceptors []*values.Object
	target       *values.Object
	// index is the position of the running interceptor, advanced by each ctx.next() call.
	index int
	state pipelineState
}

type pipelineState uint8

const (
	// stageInterceptor marks an interceptor as running, rather than the service.
	stageInterceptor pipelineState = 1 << iota
	// stageNextCalled marks that the running interceptor called ctx.next().
	stageNextCalled
)

func (s pipelineState) has(flag pipelineState) bool { return s&flag != 0 }

func pipelineOf(requestCtx *values.Object) (*requestPipeline, bool) {
	p, ok := requestCtx.Get(pipelineSlot)
	if !ok {
		return nil, false
	}
	return p.(*requestPipeline), true
}

func (p *requestPipeline) enter(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.index, p.state = index, stageInterceptor
}

// leave ends the interceptor at index and returns the position the pipeline continues from and
// whether the interceptor called ctx.next().
func (p *requestPipeline) leave(index int) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	nextCalled := p.state.has(stageNextCalled)
	p.state = 0
	return max(p.index, index+1), nextCalled
}

// next returns the stage after the one ctx.next() last reached, as jBallerina's
// ExternRequestContext.next does: each call advances, and only one call may pass the service.
func (p *requestPipeline) next(ctx *extern.Context) values.BalValue {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.state.has(stageInterceptor) || p.index >= len(p.interceptors) {
		return httpError(ctx, internalInterceptorReturnError, noNextService, nil)
	}
	p.state |= stageNextCalled
	p.index++
	if p.index < len(p.interceptors) {
		return p.interceptors[p.index]
	}
	return p.target
}

// stageAt is the stage the pipeline continues with at index.
func (p *requestPipeline) stageAt(index int) *values.Object {
	if index < len(p.interceptors) {
		return p.interceptors[index]
	}
	return p.target
}

func registerRequestContextExterns(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "RequestContext.next", nativeRequestContextNext)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "RequestContext.getWithType", nativeGetWithType)
}

func nativeRequestContextNext(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	pipeline, ok := pipelineOf(args[0].(*values.Object))
	if !ok {
		return httpError(ctx, internalInterceptorReturnError, noPipeline, nil), nil
	}
	return pipeline.next(ctx), nil
}

func nativeGetWithType(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, key := args[0].(*values.Object), args[1].(string)
	handle, _ := ctx.LookupObjectMethod(self, "member")
	value, err := ctx.InvokeMethod(handle, []values.BalValue{self, key})
	if err != nil {
		return nil, err
	}
	if cause, missing := value.(*values.Error); missing {
		return httpError(ctx, genericListenerError, "no member found for key: "+key, cause), nil
	}
	converted, err := values.CastValue(ctx.TypeCtx(), value, args[2].(*values.TypeDesc).Constraint())
	if err != nil {
		return httpError(ctx, genericListenerError, "type conversion failed for value of key: "+key, errorCause(err)), nil
	}
	return converted, nil
}
