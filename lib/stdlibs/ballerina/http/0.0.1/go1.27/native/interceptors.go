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
	registrationError = "interceptor service registration failed: "
)

// interceptorKind is the interceptor service type an interceptor includes.
type interceptorKind uint8

const (
	requestKind interceptorKind = iota
	responseKind
	requestErrorKind
	responseErrorKind
	// defaultErrorKind is the default error interceptor, which has no object: the dispatcher
	// answers an error that reaches it with the error's response.
	defaultErrorKind
)

// remoteMethods are the remote methods the response-side kinds are invoked through.
var remoteMethods = map[interceptorKind]string{responseKind: "interceptResponse", responseErrorKind: "interceptResponseError"}

// interceptorTypes are the types the dispatcher checks services, interceptors and parameters against.
type interceptorTypes struct {
	interceptable        semtypes.SemType
	kinds                [defaultErrorKind]semtypes.SemType
	nextService          semtypes.SemType
	response             semtypes.SemType
	requestContext       semtypes.SemType
	createRequestContext extern.FunctionHandle
}

func resolveInterceptorTypes(ctx *extern.Context) (interceptorTypes, error) {
	handle, found := ctx.LookupFunction(orgName, moduleName, "interceptorTypes")
	signature, ok := ctx.FunctionSignature(handle)
	if !found || !ok || len(signature.Params) != 7 {
		return interceptorTypes{}, fmt.Errorf("http: cannot resolve the interceptor types from 'interceptorTypes'")
	}
	createContext, contextSignature, err := resolveFactory(ctx, "createRequestContext")
	if err != nil {
		return interceptorTypes{}, err
	}
	params := signature.Params
	return interceptorTypes{
		interceptable:        params[0].Type,
		kinds:                [...]semtypes.SemType{params[1].Type, params[2].Type, params[3].Type, params[4].Type},
		nextService:          params[5].Type,
		response:             params[6].Type,
		requestContext:       contextSignature.ReturnType,
		createRequestContext: createContext,
	}, nil
}

func isOfType(tc semtypes.Context, v values.BalValue, ty semtypes.SemType) bool {
	return semtypes.IsSubtype(tc, values.SemTypeForValue(v), ty)
}

// interceptor is a stage of the interceptor pipeline of a service.
type interceptor struct {
	obj  *values.Object
	kind interceptorKind
}

// createInterceptors returns the interceptor pipeline of svcObj, the default error interceptor
// followed by what its createInterceptors method returns, or nil when it engages no interceptor.
func createInterceptors(ctx *extern.Context, types interceptorTypes, svcObj *values.Object) ([]interceptor, error) {
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
	if len(elements) == 0 {
		return nil, nil
	}
	stages := make([]interceptor, 1, len(elements)+1)
	stages[0] = interceptor{kind: defaultErrorKind}
	for _, element := range elements {
		for kind, ty := range types.kinds {
			if isOfType(tc, element, ty) {
				stages = append(stages, interceptor{obj: element.(*values.Object), kind: interceptorKind(kind)})
				break
			}
		}
	}
	return stages, nil
}

// requestPipeline is the native state of the http:RequestContext of a dispatched request: the
// stages of its service and how far the request and response walks over them have come.
type requestPipeline struct {
	mu     sync.Mutex
	stages []interceptor
	target *values.Object
	// current is the position of the running request-side interceptor.
	current int
	// index is the request-side position ctx.next() reached.
	index int
	state pipelineState
	// walk is the state of the running response-side interceptor, kept apart from the request
	// side since a request interceptor's caller->respond runs the response walk inside it.
	walk responseWalk
}

type pipelineState uint8

const (
	// stageRequest marks a request-side interceptor as running.
	stageRequest pipelineState = 1 << iota
	// stageNextCalled marks that the running interceptor called ctx.next().
	stageNextCalled
)

func (s pipelineState) has(flag pipelineState) bool { return s&flag != 0 }

type responseWalk struct {
	running bool
	current int
	// response is the position ctx.next() reached; -1 once it passed the first stage.
	response int
}

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
	p.current, p.index, p.state = index, index, stageRequest
}

// leave ends the request-side interceptor at index and returns the position the pipeline
// continues from and whether the interceptor called ctx.next().
func (p *requestPipeline) leave(index int) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	nextCalled := p.state.has(stageNextCalled)
	p.state = 0
	return max(p.index, index+1), nextCalled
}

// enterResponse starts the response-side interceptor at index and returns the walk state it
// replaces, which leaveResponse restores.
func (p *requestPipeline) enterResponse(index int) responseWalk {
	p.mu.Lock()
	defer p.mu.Unlock()
	outer := p.walk
	p.walk = responseWalk{running: true, current: index, response: index}
	return outer
}

// leaveResponse ends the response-side interceptor at index and returns the position the
// response continues from, as jBallerina computes it, and whether ctx.next() passed the first stage.
func (p *requestPipeline) leaveResponse(index int, outer responseWalk) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	response := p.walk.response
	p.walk = outer
	return min(response, index-1), response < 0
}

// responseStart is where the walk of a response sent now starts: at the running request
// interceptor, below the running response interceptor, or else at the last stage.
func (p *requestPipeline) responseStart() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch {
	case p.walk.running:
		return p.walk.current - 1
	case p.state.has(stageRequest):
		return p.current
	}
	return len(p.stages) - 1
}

// next returns the stage after the one ctx.next() last reached, on the side of the running
// interceptor.
func (p *requestPipeline) next(ctx *extern.Context) values.BalValue {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch {
	case p.walk.running:
		for p.walk.response--; p.walk.response >= 0; p.walk.response-- {
			if p.stages[p.walk.response].kind == responseKind {
				return p.stages[p.walk.response].obj
			}
		}
		p.walk.response = -1
		return nil
	case p.state.has(stageRequest) && p.index < len(p.stages):
		p.state |= stageNextCalled
		for p.index++; p.index < len(p.stages); p.index++ {
			if p.stages[p.index].kind == requestKind {
				return p.stages[p.index].obj
			}
		}
		return p.target
	}
	return httpError(ctx, internalInterceptorReturnError, noNextService, nil)
}

// stageAt is the stage the request continues with at index.
func (p *requestPipeline) stageAt(index int) *values.Object {
	if index < len(p.stages) {
		return p.stages[index].obj
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
