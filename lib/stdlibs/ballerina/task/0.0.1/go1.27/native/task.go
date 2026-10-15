// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName       = "ballerina"
	moduleName    = "task"
	executeMethod = "execute"
)

func initTaskModule(rt *runtime.Runtime) {
	m := newTaskManager(rt, rt.Platform())
	ld := semtypes.NewListDefinition()
	intArrayTy := ld.Define(rt.GetTypeEnv(), nil, semtypes.ListRest(semtypes.Int))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "configureThread", configureThreadExtern(m))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "scheduleJob", scheduleJobExtern(m))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "scheduleIntervalJob", scheduleIntervalJobExtern(m))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUnscheduleJob", jobIDExtern(m.unscheduleJob))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPauseJob", jobIDExtern(m.pauseJob))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externResumeJob", jobIDExtern(m.resumeJob))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPauseAllJobs", noArgExtern(m.pauseAll))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externResumeAllJobs", noArgExtern(m.resumeAll))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetRunningJobs",
		func(ctx *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			ids := m.getRunningJobs()
			items := make([]values.BalValue, len(ids))
			for i, id := range ids {
				items[i] = id
			}
			atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), intArrayTy)
			return values.NewList(intArrayTy, atomic, false, nil, 0, items), nil
		})
	registerListenerExterns(m)
}

func init() {
	runtime.RegisterModuleInitializer(initTaskModule)
}

func errorResult(err error) values.BalValue {
	if err != nil {
		return err.Error()
	}
	return nil
}

func idResult(id int64, err error) values.BalValue {
	if err != nil {
		return err.Error()
	}
	return id
}

func configureThreadExtern(m *taskManager) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		props := schedulerProps{threadCount: args[0].(int64), thresholdMs: args[1].(int64)}
		return errorResult(m.configure(props)), nil
	}
}

func scheduleJobExtern(m *taskManager) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		job := &jobDetail{obj: args[0].(*values.Object), errorPolicy: policyLogAndContinue}
		return idResult(m.scheduleOneTime(job, args[1].(int64))), nil
	}
}

func scheduleIntervalJobExtern(m *taskManager) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		policy := args[5].(*values.Map)
		job := &jobDetail{obj: args[0].(*values.Object), errorPolicy: stringField(policy, "errorPolicy")}
		sched := intervalSchedule{
			intervalMs:    toMillis(args[1].(*decimal.Decimal)),
			maxCount:      args[2].(int64),
			startMs:       optionalMillis(args[3]),
			endMs:         optionalMillis(args[4]),
			waitingPolicy: stringField(policy, "waitingPolicy"),
		}
		return idResult(m.scheduleInterval(job, sched)), nil
	}
}

func jobIDExtern(op func(id int64) error) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return errorResult(op(args[0].(int64))), nil
	}
}

func noArgExtern(op func() error) extern.NativeFunc {
	return func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
		return errorResult(op()), nil
	}
}

// execute runs one fire of spec's job on a fresh strand, as jBallerina's
// TaskJob and TaskServerJob do on a virtual thread per fire.
func (m *taskManager) execute(s *scheduler, spec *triggerSpec) {
	ctx := m.rt.AcquirePooledContext()
	defer m.rt.ReleasePooledContext(ctx)
	if spec.job.isService {
		m.executeService(ctx, s, spec)
		return
	}
	if _, msg, panicked := invokeExecute(ctx, spec.job.obj); panicked {
		m.notifyFailure(s, spec, msg)
	}
}

// invokeExecute calls obj.execute(), recovering a panic into its message.
func invokeExecute(ctx *extern.Context, obj *values.Object) (result values.BalValue, panicMsg string, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg, panicked = panicMessage(r), true
		}
	}()
	result, err := callExecute(ctx, obj)
	if err != nil {
		return nil, err.Error(), true
	}
	return result, "", false
}

func callExecute(ctx *extern.Context, obj *values.Object) (values.BalValue, error) {
	h, ok := ctx.LookupObjectMethod(obj, executeMethod)
	if !ok {
		return nil, fmt.Errorf("method '%s' not found", executeMethod)
	}
	return ctx.InvokeMethod(h, []values.BalValue{obj})
}

func panicMessage(r any) string {
	switch v := r.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r)
	}
}

func stringField(m *values.Map, key string) string {
	v, _ := m.Get(key)
	s, _ := v.(string)
	return s
}

func optionalMillis(v values.BalValue) int64 {
	if ms, ok := v.(int64); ok {
		return ms
	}
	return noFireTime
}

// toMillis converts seconds to milliseconds, truncating toward zero like
// Java's BigDecimal.longValue.
func toMillis(seconds *decimal.Decimal) int64 {
	ms, _ := seconds.Mul(decimal.FromInt64(1000))
	return truncate(ms)
}

func truncate(d *decimal.Decimal) int64 {
	var whole *decimal.Decimal
	if d.Cmp(decimal.FromInt64(0)) < 0 {
		whole, _ = d.Ceiling()
	} else {
		whole, _ = d.Floor()
	}
	n, _, _ := whole.Int64()
	return n
}
