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
		result, err := strand.InvokeFunctionValue(fn, callArgs)
		if err != nil {
			result = values.NewErrorWithMessage(err.Error())
		}
		inst.result = result
		close(inst.done)
	}()
	return inst.id, nil
}

// workflowArgs binds each workflow parameter by its type: the Context object, the events
// record of futures, or the input.
func (e *engine) workflowArgs(ctx *extern.Context, inst *instance, sig extern.FunctionSignature,
	input values.BalValue) ([]values.BalValue, *values.Error) {
	cx := ctx.TypeCtx()
	callArgs := make([]values.BalValue, len(sig.Params))
	for i, p := range sig.Params {
		switch {
		case semtypes.IsSubtype(cx, p.Type, semtypes.Object):
			wfCtx, errVal := newContextObject(ctx, inst)
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

func newContextObject(ctx *extern.Context, inst *instance) (values.BalValue, *values.Error) {
	factory, ok := ctx.LookupFunction(orgName, moduleName, "newContext")
	if !ok {
		return nil, values.NewErrorWithMessage("workflow:newContext not found")
	}
	obj, err := ctx.InvokeFunction(factory, []values.BalValue{inst})
	if err != nil {
		return nil, values.NewErrorWithMessage(err.Error())
	}
	return obj, nil
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

func (e *engine) lookup(id string) (*instance, *values.Error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	inst, ok := e.instances[id]
	if !ok {
		return nil, values.NewErrorWithMessage("workflow instance not found: " + id)
	}
	return inst, nil
}

func (e *engine) sendData(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(args[1].(string))
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

func (e *engine) getStatus(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(args[0].(string))
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
	inst, errVal := e.lookup(args[0].(string))
	if errVal != nil {
		return errVal, nil
	}
	select {
	case <-inst.done:
		return convertResult(ctx, inst.result, args[1].(*values.TypeDesc)), nil
	default:
		return inProgressError(inst), nil
	}
}

func inProgressError(inst *instance) *values.Error {
	return values.NewErrorWithMessage("workflow is still running: " + inst.id)
}

func (e *engine) waitForResult(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	inst, errVal := e.lookup(args[0].(string))
	if errVal != nil {
		return errVal, nil
	}
	timeout, errVal := durationMillis(args[2])
	if errVal != nil {
		return errVal, nil
	}
	if timeout < 0 {
		<-inst.done
	} else {
		select {
		case <-inst.done:
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			return inProgressError(inst), nil
		}
	}
	return convertResult(ctx, inst.result, args[1].(*values.TypeDesc)), nil
}

// durationMillis converts a workflow:Duration record to milliseconds; nil means no bound.
func durationMillis(duration values.BalValue) (int64, *values.Error) {
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
		return 0, values.NewErrorWithMessage("a timeout does not support 'years' or 'months': they have no fixed length")
	}
	seconds := intField("weeks")*604800 + intField("days")*86400 + intField("hours")*3600 + intField("minutes")*60
	millis := seconds * 1000
	if v, ok := fields.Get("seconds"); ok {
		millis += int64(v.(*decimal.Decimal).Float64() * 1000)
	}
	return millis, nil
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
func (e *engine) callActivity(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
	callArgs := make([]values.BalValue, len(sig.Params))
	paramNames := make(map[string]bool, len(sig.Params))
	for i, p := range sig.Params {
		paramNames[p.Name] = true
		v, present := named.Get(p.Name)
		if !present {
			return values.NewErrorWithMessage("missing activity argument: " + p.Name), nil
		}
		callArgs[i] = v
	}
	for _, key := range named.Keys() {
		if !paramNames[key] {
			return values.NewErrorWithMessage("unknown activity argument: " + key), nil
		}
	}
	result, err := ctx.InvokeFunctionValue(fn, callArgs)
	if err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}
	return convertResult(ctx, result, td), nil
}
