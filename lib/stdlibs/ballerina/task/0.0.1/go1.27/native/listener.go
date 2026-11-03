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
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	stateField          = "$state"
	exponentialStrategy = "EXPONENTIAL"
)

// listenerState is the Go-side state of a task:Listener, stored on the
// object's "$state" field.
type listenerState struct {
	mu         sync.Mutex
	trigger    *values.Map
	warmBackup bool
	names      []string
	services   map[string]*values.Object
}

type retryConfig struct {
	maxAttempts    int64
	retryInterval  int64
	backoff        string
	maxInterval    int64
	hasMaxInterval bool
}

func registerListenerExterns(m *taskManager) {
	register := func(name string, fn extern.NativeFunc) {
		runtime.RegisterExternFunction(m.rt, orgName, moduleName, "Listener."+name, fn)
	}
	register("initListener", initListenerExtern)
	register("attachService", attachServiceExtern)
	register("detachService", detachServiceExtern(m))
	register("startListener", startListenerExtern(m))
	register("stopListener", stopListenerExtern(m))
}

func initListenerExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	config := args[1].(*values.Map)
	trigger, _ := config.Get("trigger")
	warmBackup, _ := config.Get("warmBackupConfig")
	args[0].(*values.Object).Put(stateField, &listenerState{
		trigger:    trigger.(*values.Map),
		warmBackup: warmBackup != nil,
		services:   map[string]*values.Object{},
	})
	return nil, nil
}

func attachServiceExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	stateOf(args[0]).register(args[2].(string), args[1].(*values.Object))
	return nil, nil
}

func detachServiceExtern(m *taskManager) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		name, ok := stateOf(args[0]).unregister(args[1].(*values.Object))
		if !ok {
			return "Service is not found in the listener", nil
		}
		return errorResult(m.unscheduleService(name)), nil
	}
}

func startListenerExtern(m *taskManager) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return errorResult(m.startListener(ctx, stateOf(args[0]))), nil
	}
}

func stopListenerExtern(m *taskManager) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return errorResult(m.stopListener(stateOf(args[0]))), nil
	}
}

func stateOf(self values.BalValue) *listenerState {
	v, _ := self.(*values.Object).Get(stateField)
	return v.(*listenerState)
}

func (st *listenerState) register(name string, svc *values.Object) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, exists := st.services[name]; !exists {
		st.names = append(st.names, name)
	}
	st.services[name] = svc
}

func (st *listenerState) unregister(svc *values.Object) (string, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, name := range st.names {
		if st.services[name] == svc {
			st.names = slices.Delete(st.names, i, i+1)
			delete(st.services, name)
			return name, true
		}
	}
	return "", false
}

func (st *listenerState) snapshot() ([]string, map[string]*values.Object) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return slices.Clone(st.names), maps.Clone(st.services)
}

func (m *taskManager) startListener(ctx *extern.Context, st *listenerState) error {
	startMs, err := civilToMillis(ctx, st.trigger, "startTime")
	if err != nil {
		return err
	}
	endMs, err := civilToMillis(ctx, st.trigger, "endTime")
	if err != nil {
		return err
	}
	intervalVal, _ := st.trigger.Get("interval")
	interval := intervalVal.(*decimal.Decimal)
	maxCount, _ := st.trigger.Get("maxCount")
	policyVal, _ := st.trigger.Get("taskPolicy")
	policy := policyVal.(*values.Map)
	retry := retryConfigOf(st.trigger)
	sched := intervalSchedule{
		intervalMs:    toMillis(interval),
		maxCount:      maxCount.(int64),
		startMs:       startMs,
		endMs:         endMs,
		waitingPolicy: stringField(policy, "waitingPolicy"),
	}
	names, services := st.snapshot()
	return m.scheduleServices(names, func(name string) (*triggerSpec, error) {
		if st.warmBackup {
			return nil, balError("Task coordination through warmBackupConfig is not supported")
		}
		job := &jobDetail{
			obj:         services[name],
			errorPolicy: stringField(policy, "errorPolicy"),
			retry:       retry,
			intervalSec: float64(float32(interval.Float64())),
			intervalInt: truncate(interval),
			isService:   true,
		}
		return m.buildIntervalSpec(name, job, sched)
	})
}

// stopListener unschedules every service even after a failure; as in
// jBallerina, the services stay registered when any unschedule fails.
func (m *taskManager) stopListener(st *listenerState) error {
	names, _ := st.snapshot()
	var firstErr error
	for _, name := range names {
		if err := m.unscheduleService(name); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return firstErr
	}
	st.mu.Lock()
	st.names = nil
	st.services = map[string]*values.Object{}
	st.mu.Unlock()
	return nil
}

// civilToMillis converts an optional time:Civil trigger field with the
// module's getTimeInMillies, as jBallerina does when the listener starts.
func civilToMillis(ctx *extern.Context, trigger *values.Map, key string) (int64, error) {
	civil, ok := trigger.Get(key)
	if !ok || civil == nil {
		return noFireTime, nil
	}
	h, found := ctx.LookupFunction(orgName, moduleName, "getTimeInMillies")
	if !found {
		return 0, balError("Error converting civil to milliseconds: getTimeInMillies not found")
	}
	result, err := ctx.InvokeFunction(h, []values.BalValue{civil})
	if err != nil {
		return 0, balError("Error converting civil to milliseconds: " + err.Error())
	}
	if e, isErr := result.(*values.Error); isErr {
		return 0, balError("Error converting civil to milliseconds: " + e.Message)
	}
	return result.(int64), nil
}

func retryConfigOf(trigger *values.Map) *retryConfig {
	v, _ := trigger.Get("retryConfig")
	cfg, ok := v.(*values.Map)
	if !ok {
		return nil
	}
	maxAttempts, _ := cfg.Get("maxAttempts")
	retryInterval, _ := cfg.Get("retryInterval")
	rc := &retryConfig{
		maxAttempts:   maxAttempts.(int64),
		retryInterval: retryInterval.(int64),
		backoff:       stringField(cfg, "backoffStrategy"),
	}
	if maxInterval, ok := cfg.Get("maxInterval"); ok {
		rc.maxInterval, rc.hasMaxInterval = maxInterval.(int64), true
	}
	return rc
}

// executeService ports TaskServerJob.executeJob: a returned error is retried
// per the retry config before the error policy applies, while a panic is
// reported on stderr like an uncaught strand panic.
func (m *taskManager) executeService(ctx *extern.Context, s *scheduler, spec *triggerSpec) {
	result, panicked := m.invokeService(ctx, spec.job.obj)
	if panicked {
		return
	}
	if _, isErr := result.(*values.Error); !isErr {
		return
	}
	if spec.job.shouldRetry() {
		if result, panicked = m.executeWithRetry(ctx, spec.job); panicked {
			return
		}
	}
	if e, isErr := result.(*values.Error); isErr {
		m.notifyFailure(s, spec, e.Message)
	}
}

func (m *taskManager) invokeService(ctx *extern.Context, obj *values.Object) (values.BalValue, bool) {
	result, msg, panicked := invokeExecute(ctx, obj)
	if panicked {
		_, _ = m.pal.IO.Stderr([]byte("error: " + msg + "\n"))
	}
	return result, panicked
}

func (job *jobDetail) shouldRetry() bool {
	rc := job.retry
	if rc == nil || rc.retryInterval > job.intervalInt {
		return false
	}
	return rc.maxAttempts > 0
}

func (m *taskManager) executeWithRetry(ctx *extern.Context, job *jobDetail) (values.BalValue, bool) {
	rc := job.retry
	var result values.BalValue
	startMs := m.nowMs()
	current := rc.retryInterval
	for attempt := int64(0); attempt < rc.maxAttempts; attempt++ {
		m.sleep(current)
		if rc.hasMaxInterval && (current > rc.maxInterval || current >= job.intervalInt) {
			break
		}
		if float64(m.nowMs()-startMs) >= job.intervalSec*1000 {
			break
		}
		var panicked bool
		if result, panicked = m.invokeService(ctx, job.obj); panicked {
			return nil, true
		}
		current = rc.nextInterval(current)
		if _, isErr := result.(*values.Error); !isErr {
			break
		}
	}
	return result, false
}

func (rc *retryConfig) nextInterval(current int64) int64 {
	if rc.backoff != exponentialStrategy {
		return rc.retryInterval
	}
	if rc.hasMaxInterval {
		return min(current*2, rc.maxInterval)
	}
	return current * 2
}

func (m *taskManager) sleep(seconds int64) {
	done := make(chan struct{})
	m.pal.Time.AfterFunc(time.Duration(seconds)*time.Second, func() { close(done) })
	<-done
}
