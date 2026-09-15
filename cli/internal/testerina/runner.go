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

package testerina

import (
	"bytes"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// Outcome is the verdict of a single test.
type Outcome uint8

const (
	Passed Outcome = iota
	Failed
	Errored
	Skipped
)

// Result is the outcome of one scheduled test.
type Result struct {
	Step     Step
	Outcome  Outcome
	Messages []string
	Stdout   []byte
	Duration time.Duration
}

// Summary is the outcome of a whole run.
type Summary struct {
	Results []Result
	// RunFailures are Fail hook calls made outside any test, e.g. from `main`,
	// a `$start` hook, or a strand that outlived the test that started it.
	RunFailures []string
}

func (s Summary) Failed() bool {
	if len(s.RunFailures) > 0 {
		return true
	}
	for _, result := range s.Results {
		if result.Outcome == Failed || result.Outcome == Errored {
			return true
		}
	}
	return false
}

// Runner executes a Schedule against a live runtime. It owns the PAL overrides
// the tests need: per-test stdout capture, the shutdown signal channel, and the
// Testing.Fail hook.
type Runner struct {
	platform pal.Platform
	// signals is the channel the runtime reads. Only forwardSignals sends on
	// it and closes it, so Shutdown asks for a signal through requests rather
	// than writing to it directly.
	signals  chan pal.Signal
	requests chan pal.Signal
	stopOnce sync.Once
	stop     chan struct{}
	rt       *runtime.Runtime

	// mu guards the state below, which is touched from arbitrary strands:
	// workers, $start hooks and service resource goroutines all reach fail and
	// the stdout hook concurrently with the runner goroutine.
	mu          sync.Mutex
	current     *Step
	stdout      bytes.Buffer
	failures    []string
	runFailures []string
}

// NewRunner builds the runner and its platform overrides. It must be called
// before runtime.NewRuntime, which copies pal.Platform by value. tyEnv is the
// environment the caller hands to runtime.NewRuntime; the runner takes it so
// callers construct both from one place, but only the runtime consults it.
func NewRunner(platform pal.Platform, osSignals <-chan pal.Signal, _ semtypes.Env) *Runner {
	r := &Runner{
		platform: platform,
		signals:  make(chan pal.Signal, 1),
		requests: make(chan pal.Signal, 1),
		stop:     make(chan struct{}),
	}
	baseStdout := platform.IO.Stdout
	r.platform.IO.Stdout = r.writeStdout(baseStdout)
	r.platform.Signals = pal.SignalSource{Signals: r.signals}
	r.platform.Testing = pal.Testing{Fail: r.fail}
	go r.forwardSignals(osSignals)
	return r
}

// Platform returns the platform the runtime must be created with.
func (r *Runner) Platform() pal.Platform {
	return r.platform
}

// Start records the runtime the schedule runs against; call it after Init and
// Listen have completed.
func (r *Runner) Start(rt *runtime.Runtime) {
	r.rt = rt
}

func (r *Runner) writeStdout(base func([]byte) (int, error)) func([]byte) (int, error) {
	return func(p []byte) (int, error) {
		r.mu.Lock()
		if r.current != nil {
			defer r.mu.Unlock()
			return r.stdout.Write(p)
		}
		r.mu.Unlock()
		return base(p)
	}
}

// forwardSignals relays OS signals and Shutdown's own stop request into the
// channel the runtime reads. It is the sole sender and closer of that channel;
// the runtime requires the sender to close it once the exit code is out.
func (r *Runner) forwardSignals(osSignals <-chan pal.Signal) {
	defer close(r.signals)
	for {
		var signal pal.Signal
		select {
		case <-r.stop:
			return
		case signal = <-r.requests:
		case received, ok := <-osSignals:
			if !ok {
				// A nil channel blocks forever, so the closed source drops out
				// of the select instead of spinning it.
				osSignals = nil
				continue
			}
			signal = received
		}
		select {
		case r.signals <- signal:
		case <-r.stop:
			return
		}
	}
}

// fail records a failure against the running test, or against the run itself
// when no test is running.
func (r *Runner) fail(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		r.runFailures = append(r.runFailures, message)
		return
	}
	r.failures = append(r.failures, message)
}

// Run executes every step sequentially and returns the run summary.
func (r *Runner) Run(schedule Schedule, reporter Reporter) Summary {
	reporter.Begin(len(schedule.Steps))
	summary := Summary{}
	for _, step := range schedule.Steps {
		result := r.runStep(step)
		summary.Results = append(summary.Results, result)
		reporter.Report(result)
	}
	summary.RunFailures = r.RunFailures()
	return summary
}

// RunFailures returns the failures reported outside any test so far. Callers
// re-read it after Shutdown, which is when lifecycle handlers get to run.
func (r *Runner) RunFailures() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.runFailures...)
}

func (r *Runner) runStep(step Step) Result {
	if !step.Test.Enabled {
		return Result{Step: step, Outcome: Skipped}
	}
	r.beginStep(step)
	start := time.Now()
	messages, errored := r.runStepBody(step)
	duration := time.Since(start)
	failures, stdout := r.endStep()

	result := Result{Step: step, Stdout: stdout, Duration: duration}
	result.Messages = append(append([]string{}, failures...), newMessages(failures, messages)...)
	switch {
	case len(failures) > 0:
		result.Outcome = Failed
	case errored:
		result.Outcome = Errored
	default:
		result.Outcome = Passed
	}
	return result
}

// runStepBody runs before, the test body and after. A failing before skips both
// the body and after; a failing after errors the step even if the body passed.
func (r *Runner) runStepBody(step Step) (messages []string, errored bool) {
	if !step.Test.Before.IsZero() {
		if message, failed := r.invoke(step.Test.Before); failed {
			return []string{message}, true
		}
	}
	if message, failed := r.invoke(bodyRef(step)); failed {
		messages, errored = append(messages, message), true
	}
	if !step.Test.After.IsZero() {
		if message, failed := r.invoke(step.Test.After); failed {
			messages, errored = append(messages, message), true
		}
	}
	return messages, errored
}

// newMessages drops panic and error messages the Fail hook already recorded.
// assertFail always produces both: it calls the hook and then panics with the
// same message.
func newMessages(failures, messages []string) []string {
	var result []string
	for _, message := range messages {
		if !slices.Contains(failures, message) {
			result = append(result, message)
		}
	}
	return result
}

func bodyRef(step Step) FunctionRef {
	return FunctionRef{Org: step.Module.Org, Module: step.Module.Module, Name: step.Test.Name}
}

func (r *Runner) beginStep(step Step) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := step
	r.current = &current
	r.stdout.Reset()
	r.failures = nil
}

func (r *Runner) endStep() (failures []string, stdout []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current = nil
	failures, r.failures = r.failures, nil
	stdout = append([]byte{}, r.stdout.Bytes()...)
	r.stdout.Reset()
	return failures, stdout
}

// invoke calls one Ballerina function, reporting a panic, a returned error
// value or a Go-level error as a failure message.
func (r *Runner) invoke(ref FunctionRef) (message string, failed bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message, failed = panicMessage(recovered), true
		}
	}()
	fn, ok := runtime.LookupFunction(r.rt, ref.Org, ref.Module, ref.Name)
	if !ok {
		return fmt.Sprintf("function %s/%s:%s not found", ref.Org, ref.Module, ref.Name), true
	}
	result, err := runtime.InvokeFunction(r.rt, fn, nil)
	if err != nil {
		return err.Error(), true
	}
	if returned, ok := result.(*values.Error); ok && returned != nil {
		return returned.Message, true
	}
	return "", false
}

func panicMessage(recovered any) string {
	if err, ok := recovered.(*values.Error); ok && err != nil {
		return err.Message
	}
	return fmt.Sprint(recovered)
}

// Shutdown stops the runtime and the signal forwarder. When no listeners
// existed, Listen already drove the runtime to Stopped and the exit-status
// channel holds its code; signalling again would be an illegal transition.
func (r *Runner) Shutdown(rt *runtime.Runtime) {
	select {
	case <-rt.ExitStatus:
	default:
		select {
		case r.requests <- pal.GracefulStop:
		default:
		}
		<-rt.ExitStatus
	}
	r.stopOnce.Do(func() { close(r.stop) })
}
