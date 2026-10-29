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
	"sync/atomic"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// engine runs workflow instances in process, without durability: a workflow is a strand,
// an activity is a direct call, and data events complete the futures of the events record.
type engine struct {
	rt        *runtime.Runtime
	mu        sync.Mutex
	instances map[string]*instance
	nextID    atomic.Uint64
}

type instance struct {
	id           string
	workflowType string
	events       map[string]*values.Future
	done         chan struct{}
	result       values.BalValue
}

func newEngine(rt *runtime.Runtime) *engine {
	return &engine{rt: rt, instances: make(map[string]*instance)}
}

func (e *engine) run(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	fn := args[0].(*values.Function)
	org, module, name := splitLookupKey(fn.LookupKey)
	handle, ok := ctx.LookupFunction(org, module, name)
	if !ok {
		return values.NewErrorWithMessage("workflow function not found: " + fn.LookupKey), nil
	}
	sig, ok := ctx.FunctionSignature(handle)
	if !ok {
		return values.NewErrorWithMessage("workflow function has no signature: " + fn.LookupKey), nil
	}
	inst := &instance{
		id:           fmt.Sprintf("%s-%d", name, e.nextID.Add(1)),
		workflowType: name,
		events:       make(map[string]*values.Future),
		done:         make(chan struct{}),
	}
	callArgs, errVal := e.workflowArgs(ctx, inst, sig, args[1])
	if errVal != nil {
		return errVal, nil
	}
	e.mu.Lock()
	e.instances[inst.id] = inst
	e.mu.Unlock()
	go func() {
		strand := e.rt.AcquirePooledContext()
		defer e.rt.ReleasePooledContext(strand)
		defer close(inst.done)
		defer func() {
			// A panicking workflow fails its instance instead of taking the process down
			if rec := recover(); rec != nil {
				strand.ReleaseAllHeldLocks()
				inst.result = panicAsError(rec)
			}
		}()
		result, err := strand.InvokeFunctionValue(fn, callArgs)
		if err != nil {
			result = values.NewErrorWithMessage(err.Error())
		}
		inst.result = result
	}()
	return inst.id, nil
}

// workflowArgs binds each workflow parameter by its type: the Context object, the events
// record of futures, or the input.
func (e *engine) workflowArgs(ctx *extern.Context, inst *instance, sig extern.FunctionSignature,
	input values.BalValue) ([]values.BalValue, *values.Error) {
	cx := ctx.TypeCtx()
	contextType, errVal := contextType(ctx)
	if errVal != nil {
		return nil, errVal
	}
	callArgs := make([]values.BalValue, len(sig.Params))
	for i, p := range sig.Params {
		switch {
		case !semtypes.IsNever(p.Type) && semtypes.IsSubtype(cx, p.Type, contextType):
			wfCtx, errVal := callModuleFunction(ctx, "newContext", inst)
			if errVal != nil {
				return nil, errVal
			}
			callArgs[i] = wfCtx
		case isEventsRecord(cx, p.Type):
			callArgs[i] = newEventsRecord(cx, inst, p.Type)
		default:
			converted, convErr := values.CloneWithType(cx, input, p.Type)
			if convErr != nil {
				return nil, convErr
			}
			callArgs[i] = converted
		}
	}
	return callArgs, nil
}

// contextType is the module's Context class, read off the factory that builds one.
func contextType(ctx *extern.Context) (semtypes.SemType, *values.Error) {
	factory, ok := ctx.LookupFunction(orgName, moduleName, "newContext")
	if !ok {
		return semtypes.Never, values.NewErrorWithMessage("workflow:newContext not found")
	}
	sig, ok := ctx.FunctionSignature(factory)
	if !ok {
		return semtypes.Never, values.NewErrorWithMessage("workflow:newContext has no signature")
	}
	return sig.ReturnType, nil
}

// callModuleFunction invokes a module-level Ballerina function that builds what Go cannot: the
// Context object and the module's typed errors.
func callModuleFunction(ctx *extern.Context, name string, args ...values.BalValue) (values.BalValue, *values.Error) {
	fn, ok := ctx.LookupFunction(orgName, moduleName, name)
	if !ok {
		return nil, values.NewErrorWithMessage("workflow:" + name + " not found")
	}
	result, err := ctx.InvokeFunction(fn, args)
	if err != nil {
		return nil, values.NewErrorWithMessage(err.Error())
	}
	return result, nil
}

// moduleError builds one of the module's typed errors through its Ballerina factory.
func moduleError(ctx *extern.Context, factory string, args ...values.BalValue) values.BalValue {
	errVal, failed := callModuleFunction(ctx, factory, args...)
	if failed != nil {
		return failed
	}
	return errVal
}

func panicAsError(rec any) *values.Error {
	switch v := rec.(type) {
	case *values.Error:
		return v
	case error:
		return values.NewErrorWithMessage(v.Error())
	default:
		return values.NewErrorWithMessage(fmt.Sprintf("%v", rec))
	}
}

func isEventsRecord(cx semtypes.Context, ty semtypes.SemType) bool {
	if !semtypes.IsSubtype(cx, ty, semtypes.Mapping) {
		return false
	}
	atomic := semtypes.ToMappingAtomicType(cx, ty)
	if atomic == nil || len(atomic.FieldNames()) == 0 {
		return false
	}
	for _, name := range atomic.FieldNames() {
		if !semtypes.IsSubtype(cx, atomic.FieldInnerVal(name), semtypes.Future) {
			return false
		}
	}
	return true
}

func newEventsRecord(cx semtypes.Context, inst *instance, ty semtypes.SemType) *values.Map {
	atomic := semtypes.ToMappingAtomicType(cx, ty)
	entries := make([]values.MapEntry, 0, len(atomic.FieldNames()))
	for _, name := range atomic.FieldNames() {
		future := values.NewFuture(atomic.FieldInnerVal(name))
		inst.events[name] = future
		entries = append(entries, values.MapEntry{Key: name, Value: future})
	}
	return values.NewMap(ty, atomic, false, entries)
}

func (e *engine) lookup(ctx *extern.Context, id string) (*instance, values.BalValue) {
	e.mu.Lock()
	defer e.mu.Unlock()
	inst, ok := e.instances[id]
	if !ok {
		return nil, moduleError(ctx, "newInstanceNotFoundError", id)
	}
	return inst, nil
}

func (e *engine) sendData(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(ctx, args[1].(string))
	if errVal != nil {
		return errVal, nil
	}
	dataName := args[2].(string)
	future, ok := inst.events[dataName]
	if !ok {
		return values.NewErrorWithMessage("workflow has no data event: " + dataName), nil
	}
	cx := ctx.TypeCtx()
	converted, convErr := values.CloneWithType(cx, args[3], semtypes.FutureEventualType(cx, future.Type))
	if convErr != nil {
		return convErr, nil
	}
	future.Complete(converted, nil)
	return nil, nil
}

func (e *engine) getStatus(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(ctx, args[0].(string))
	if errVal != nil {
		return errVal, nil
	}
	select {
	case <-inst.done:
		if _, failed := inst.result.(*values.Error); failed {
			return "FAILED", nil
		}
		return "COMPLETED", nil
	default:
		return "RUNNING", nil
	}
}

func (e *engine) getResult(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(ctx, args[0].(string))
	if errVal != nil {
		return errVal, nil
	}
	select {
	case <-inst.done:
		return instanceResult(ctx, inst, args[1].(*values.TypeDesc)), nil
	default:
		return moduleError(ctx, "newInProgressError", inst.id), nil
	}
}

func (e *engine) waitForResult(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(ctx, args[0].(string))
	if errVal != nil {
		return errVal, nil
	}
	timeout, boundErr := durationMillis(args[2], "a timeout")
	if boundErr != nil {
		return boundErr, nil
	}
	// Poll between yields, as runtime:sleep does: blocking on the channel would hold this
	// strand's thread, and a strand sharing it could be the one that sends the data.
	clock := ctx.Env.Platform.Time
	deadline := clock.MonotonicNow() + time.Duration(timeout)*time.Millisecond
	for {
		select {
		case <-inst.done:
			return instanceResult(ctx, inst, args[1].(*values.TypeDesc)), nil
		default:
		}
		if timeout >= 0 && clock.MonotonicNow() >= deadline {
			return moduleError(ctx, "newInProgressError", inst.id), nil
		}
		<-ctx.Yield()
	}
}

// durationMillis converts a workflow:Duration record to milliseconds; nil means no bound.
func durationMillis(duration values.BalValue, what string) (int64, *values.Error) {
	if duration == nil {
		return -1, nil
	}
	fields := duration.(*values.Map)
	intField := func(name string) int64 {
		v, ok := fields.Get(name)
		if !ok {
			return 0
		}
		return v.(int64)
	}
	if intField("years") != 0 || intField("months") != 0 {
		return 0, values.NewErrorWithMessage(what + " does not support 'years' or 'months': they have no fixed length")
	}
	seconds := intField("weeks")*604800 + intField("days")*86400 + intField("hours")*3600 + intField("minutes")*60
	millis := seconds * 1000
	if v, ok := fields.Get("seconds"); ok {
		millis += int64(v.(*decimal.Decimal).Float64() * 1000)
	}
	if millis < 0 {
		return 0, values.NewErrorWithMessage(what + " must not be negative")
	}
	return millis, nil
}

// instanceResult is what a result read answers for a closed instance: the result as T, or an
// InstanceFailedError carrying the workflow's own error.
func instanceResult(ctx *extern.Context, inst *instance, td *values.TypeDesc) values.BalValue {
	if errVal, ok := inst.result.(*values.Error); ok {
		return moduleError(ctx, "newInstanceFailedError", inst.id, errVal)
	}
	return convertResult(ctx, inst.result, td)
}

func convertResult(ctx *extern.Context, result values.BalValue, td *values.TypeDesc) values.BalValue {
	if errVal, ok := result.(*values.Error); ok {
		return errVal
	}
	converted, convErr := values.CloneWithType(ctx.TypeCtx(), result, td.Constraint())
	if convErr != nil {
		return convErr
	}
	return converted
}

// callActivity runs the activity on the workflow's strand, binding named args to parameters.
func (e *engine) callActivity(ctx *extern.Context, args []values.BalValue) (result values.BalValue, err error) {
	fn := args[1].(*values.Function)
	named := args[2].(*values.Map)
	td := args[3].(*values.TypeDesc)
	org, module, name := splitLookupKey(fn.LookupKey)
	handle, ok := ctx.LookupFunction(org, module, name)
	if !ok {
		return values.NewErrorWithMessage("activity function not found: " + fn.LookupKey), nil
	}
	sig, ok := ctx.FunctionSignature(handle)
	if !ok {
		return values.NewErrorWithMessage("activity function has no signature: " + fn.LookupKey), nil
	}
	paramNames := make(map[string]bool, len(sig.Params))
	for _, p := range sig.Params {
		paramNames[p.Name] = true
	}
	for _, key := range named.Keys() {
		if !paramNames[key] {
			return values.NewErrorWithMessage("unknown activity argument: " + key), nil
		}
	}
	callArgs := make([]values.BalValue, len(sig.Params))
	for i, p := range sig.Params {
		v, present := named.Get(p.Name)
		if !present {
			return values.NewErrorWithMessage("missing activity argument: " + p.Name), nil
		}
		callArgs[i] = v
	}
	// An activity's panic comes back to the workflow as an error, as a failed activity does
	defer func() {
		if rec := recover(); rec != nil {
			result, err = panicAsError(rec), nil
		}
	}()
	callResult, callErr := ctx.InvokeFunctionValue(fn, callArgs)
	if callErr != nil {
		return values.NewErrorWithMessage(callErr.Error()), nil
	}
	return convertResult(ctx, callResult, td), nil
}
