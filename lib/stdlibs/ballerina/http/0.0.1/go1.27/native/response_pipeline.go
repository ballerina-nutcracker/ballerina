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
	"net/http"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// answer sends what a stage returned through the response interceptors from position start. A
// stage with an http:Caller parameter cannot respond once the walk is over, and its () is a 500.
func (x *exchange) answer(ctx *extern.Context, result values.BalValue, hasCaller bool, start int) {
	conn := x.conn
	if hasCaller {
		defer conn.close()
	}
	if conn.responded() {
		conn.rejectReturn(result)
		return
	}
	if err, isError := result.(*values.Error); isError {
		x.deliver(nil, err, start)
		return
	}
	nilStatus := http.StatusAccepted
	if hasCaller {
		nilStatus = http.StatusInternalServerError
	}
	resp, err := x.toResponse(ctx, result, nilStatus)
	x.deliver(resp, err, start)
}

// toResponse converts a value a stage returned into the response it is sent as, or the error
// converting it raised.
func (x *exchange) toResponse(ctx *extern.Context, result values.BalValue, nilStatus int) (*values.Object, *values.Error) {
	converted, err := invokeModuleFunction(ctx, "toResponse", result, strings.ToUpper(x.conn.accessor), int64(nilStatus))
	if err != nil {
		return nil, errorCause(err)
	}
	if convErr, isError := converted.(*values.Error); isError {
		return nil, convErr
	}
	return converted.(*values.Object), nil
}

// step is where the response walk goes after a stage: on with resp or err from position from, or
// nowhere, when the stage answered the request itself or failed with failure.
type step struct {
	resp    *values.Object
	err     *values.Error
	from    int
	done    bool
	failure values.BalValue
}

// deliver walks resp, or err, from position start towards the first stage, and returns the
// error the response failed with, if any.
func (x *exchange) deliver(resp *values.Object, err *values.Error, start int) values.BalValue {
	rt := x.conn.rt
	// The interceptors run on their own context, so a panic in one releases no lock of the
	// strand that responded.
	ctx := rt.AcquirePooledContext()
	defer rt.ReleasePooledContext(ctx)
	stages := x.conn.pipeline.stages
	for i := start; i >= 0; {
		stage := stages[i]
		switch {
		case stage.kind == defaultErrorKind && err != nil:
			converted, convErr := errorResponse(ctx, err, x.r)
			if convErr != nil {
				return x.fail(ctx, panicMessage(convErr)).failure
			}
			resp, err, i = converted, nil, -1
		case stage.kind == responseErrorKind && err != nil, stage.kind == responseKind && err == nil:
			next := x.intercept(ctx, i, stage, resp, err)
			if next.done {
				return next.failure
			}
			resp, err, i = next.resp, next.err, next.from
		default:
			i--
		}
	}
	return x.conn.write(ctx, resp)
}

// intercept runs the response-side interceptor at position index on resp, or on err and the
// response it maps to, and acts on its result.
func (x *exchange) intercept(ctx *extern.Context, index int, stage interceptor, resp *values.Object,
	err *values.Error) step {
	conn, pipeline := x.conn, x.conn.pipeline
	handle, _ := ctx.LookupRemoteMethod(stage.obj, remoteMethods[stage.kind])
	if err != nil {
		errResp, invokeErr := invokeModuleFunction(ctx, "getErrorResponse", err)
		if invokeErr != nil {
			return x.fail(ctx, invokeErr.Error())
		}
		resp = errResp.(*values.Object)
	}
	args, argErr := x.responseArgs(ctx, handle, remoteMethods[stage.kind], stage.obj, resp, err)
	if argErr != nil {
		return x.fail(ctx, argErr.Error())
	}
	outer := pipeline.enterResponse(index)
	result, failure, failed := invokeStage(ctx, conn.rt, x.r, handle, args)
	from, exhausted := pipeline.leaveResponse(index, outer)
	// Once ctx.next() passed the first stage, an error still reaches the error interceptors.
	errorFrom := from
	if exhausted {
		errorFrom = index - 1
	}
	resultErr, isError := result.(*values.Error)
	obj, isObject := result.(*values.Object)
	nextService := isObject && isOfType(ctx.TypeCtx(), obj, x.state.interceptorTypes.nextService)
	switch {
	case failed:
		conn.failPanic(failure)
		return step{done: true, failure: httpError(ctx, genericListenerError, failure, nil)}
	case conn.responded():
		return step{done: true}
	case result == nil && exhausted:
		return step{resp: resp, from: -1}
	case result == nil:
		// Unlike jBallerina, the 202 does not depend on an http:Caller parameter of the resource (#9902).
		return x.replace(ctx, nil, from, errorFrom)
	case isError:
		return step{err: resultErr, from: errorFrom}
	case nextService && !exhausted && obj == pipeline.stages[from].obj:
		return step{resp: resp, from: from}
	case nextService:
		// Unlike jBallerina, a service returned after ctx.next() passed the first stage is a
		// mismatch rather than a panic (#9588).
		return step{err: httpError(ctx, internalInterceptorReturnError, nextMismatch, nil), from: errorFrom}
	default:
		return x.replace(ctx, result, from, errorFrom)
	}
}

// replace continues the walk from position from with the response result converts to.
func (x *exchange) replace(ctx *extern.Context, result values.BalValue, from, errorFrom int) step {
	replaced, convErr := x.toResponse(ctx, result, http.StatusAccepted)
	if convErr != nil {
		return step{err: convErr, from: errorFrom}
	}
	return step{resp: replaced, from: from}
}

// fail answers with a 500 for message and ends the walk.
func (x *exchange) fail(ctx *extern.Context, message string) step {
	x.conn.fail(http.StatusInternalServerError, message)
	return step{done: true, failure: httpError(ctx, genericListenerError, message, nil)}
}

// responseArgs binds the parameters of a response-side interceptor's remote method by type.
func (x *exchange) responseArgs(ctx *extern.Context, handle extern.MethodHandle, name string, receiver *values.Object,
	resp *values.Object, err *values.Error) ([]values.BalValue, error) {
	signature, _ := ctx.MethodSignature(handle)
	tc, state := ctx.TypeCtx(), x.state
	args := make([]values.BalValue, 0, len(signature.Params)+1)
	args = append(args, receiver)
	for _, p := range signature.Params {
		var arg values.BalValue
		var bindErr error
		switch {
		case isExactType(tc, state.interceptorTypes.requestContext, p.Type):
			arg, bindErr = x.conn.requestContext(ctx, state.interceptorTypes.createRequestContext)
		case isExactType(tc, state.requestTy, p.Type):
			arg = buildRequest(x.r.Method, x.r.RequestURI, x.r.Proto, x.r.Header, nil, 0, x.r.URL.RawQuery, x.body)
		case isExactType(tc, state.interceptorTypes.response, p.Type):
			arg = resp
		case isExactType(tc, state.callerTy, p.Type):
			arg, bindErr = newCaller(ctx, state, x.conn)
		case semtypes.IsSubtype(tc, p.Type, semtypes.Error):
			arg = boundError(err)
		default:
			bindErr = fmt.Errorf("invalid parameter type: '%s' in '%s' remote method", semtypes.ToString(tc, p.Type), name)
		}
		if bindErr != nil {
			return nil, bindErr
		}
		args = append(args, arg)
	}
	return args, nil
}

// boundError is the value of an error parameter: the error being handled, or, as in jBallerina,
// a placeholder when there is none.
func boundError(err *values.Error) *values.Error {
	if err == nil {
		return values.NewErrorWithMessage("new error")
	}
	return err
}

// write sends resp as the response, unless a response was already sent.
func (c *callerConn) write(ctx *extern.Context, resp *values.Object) values.BalValue {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.has(stateResponded) {
		return httpError(ctx, genericListenerError, illegalRespond, nil)
	}
	c.state.set(stateResponded)
	err := writeResponse(c.rt, ctx, c.w, c.r, resp)
	c.markSent()
	if err != nil {
		return httpError(ctx, genericListenerError, err.Error(), nil)
	}
	return nil
}
