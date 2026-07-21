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

package runtime

import (
	"fmt"
	"iter"
	"sync"

	"ballerina/platform/pal"
	"ballerina/runtime/extern"
	"ballerina/runtime/internal/exec"
	"ballerina/values"
)

// State is the runtime lifecycle state.
type State uint32

const (
	StateUninitialized State = iota
	StateInitializing
	StateListening
	StateGracefulStopping
	StateImmediateStopping
	StateStopped
)

const numStates = int(StateStopped) + 1

func (s State) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateInitializing:
		return "initializing"
	case StateListening:
		return "listening"
	case StateGracefulStopping:
		return "gracefulStopping"
	case StateImmediateStopping:
		return "immediateStopping"
	case StateStopped:
		return "stopped"
	default:
		panic("unexpected")
	}
}

// lifeCycle bundles the lifecycle state private to a Runtime.
// Current spec don't fully describe the how the lifecycle management happens. Therefore this implementation makes following assumptions/design decisions
// 1. Signals are triggered by humans (we are not dealing with repeat signals coming within micro-seconds)
// 2. We will not handle signals until start has been completed (IMPORTANT: we are not dropping the signal we just don't execute them until start is over)
//   - this is to simplify the design
//
// 3. We listen to signal only after starting initializing phase. Also we don't support concurrent init so calls to init should be sequential
//   - Sending a stop signal during initialization will cause runtime to panic (we don't have listeners started so nothing to stop there, winding down any user strands started by init/main is ignored)
//   - In listening phase sending a kill signal will lead to a graceful shutdown from the runtime and will write the exit code to ExitStatus channel
//
// 4. We try to do best effort escalation on repeated graceful stop signals
//   - We'll finish whatever the module we are trying to gracefully stop and then escalate to immediate stop
//
// 5 . Sending signals to a stopped runtime could trigger a panic (this is treated as undefined behavior)
// IMPORTANT: "sender" for pal.Signal must close it after we write to ExitStatus channel
type lifeCycle struct {
	mu               sync.Mutex
	state            State
	startFns         []*exec.InvokableHandle
	gracefulStopFns  []*exec.InvokableHandle
	immediateStopFns []*exec.InvokableHandle
	stopHandlers     []*exec.InvokableHandle

	exitCode     uint8
	exitCodeChan chan<- uint8
	listening    bool
}

type action func(rt *Runtime)

// transitionTable[from][to] holds the action invoked when state moves from
// `from` to `to`. A nil entry means the edge is illegal and any attempt to
// take it panics. Actions run with rt.mu released.
var transitionTable [numStates][numStates]action

func init() {
	transitionTable[StateUninitialized][StateInitializing] = initializingAction

	transitionTable[StateInitializing][StateInitializing] = initializingAction
	transitionTable[StateInitializing][StateGracefulStopping] = gracefulStopAction
	transitionTable[StateInitializing][StateImmediateStopping] = abortAction
	transitionTable[StateInitializing][StateListening] = listenAction
	transitionTable[StateInitializing][StateStopped] = stoppedAction

	transitionTable[StateListening][StateGracefulStopping] = gracefulStopAction
	transitionTable[StateListening][StateImmediateStopping] = immediateStopAction

	transitionTable[StateGracefulStopping][StateGracefulStopping] = immediateStopAction
	transitionTable[StateGracefulStopping][StateImmediateStopping] = immediateStopAction
	transitionTable[StateGracefulStopping][StateStopped] = stoppedAction

	transitionTable[StateImmediateStopping][StateStopped] = stoppedAction

	transitionTable[StateStopped][StateStopped] = stoppedAction
}

// transition is the only mutator of state. It validates the edge under
// rt.mu, swaps the state, releases the mutex, and then invokes the action
// with the mutex released so long-running stop sequences cannot block
// concurrent transitions (e.g. graceful → immediate escalation).
func (rt *Runtime) transition(target State) {
	rt.mu.Lock()
	from := rt.state
	action := transitionTable[from][target]
	rt.mu.Unlock()
	if action == nil {
		panic(fmt.Sprintf("invalid lifecycle transition from %s -> %s", from, target))
	}
	action(rt)
}

func initializingAction(rt *Runtime) {
	rt.mu.Lock()
	rt.state = StateInitializing
	rt.mu.Unlock()
	rt.env.TypeEnv.Freeze()
	rt.setupSignalListeners()
}

func abortAction(_ *Runtime) {
	panic("ABORT: aborting module initialization due to stop signal")
}

func (rt *Runtime) stopAfterInitFailure() {
	rt.mu.Lock()
	rt.exitCode = 1
	rt.mu.Unlock()
	rt.transition(StateGracefulStopping)
}

// listenAction runs $start for every registered module on the caller's
// goroutine. On the first $start failure it cascades into a graceful
// stop. Spawns the signal-watcher goroutine exactly once before any
// $start runs so a signal that arrives mid-startup is not lost.
func listenAction(rt *Runtime) {
	rt.mu.Lock()
	rt.state = StateListening
	onError := func(message string) {
		writeStderr(rt.env, message)
		rt.exitCode = 1
		rt.mu.Unlock()
		rt.transition(StateGracefulStopping)
	}
	for _, fn := range rt.startFns {
		cx := exec.CreateContext(rt.env)
		res, err := exec.Invoke(cx, fn, nil)
		if err != nil {
			onError(err.Error())
			return
		}
		if errVal, isErr := res.(*values.Error); isErr {
			onError("error: " + errVal.Message + "\n")
			return
		}
	}
	rt.mu.Unlock()
}

// gracefulStopAction call gracefulStop on all module listeners in the
// reverse order modules were initialized. Then it calls stop handlers
// registered via runtime:onGracefulStop again on the reverse order they
// registered. If we get another stop signal while performing this, or get
// an error from any of the above functinos we transition to immediate stop.
//
// NOTE: currently for listeners this stop signal escalation is granular at
// module level not listener level. This is due to how desugar generate
// module gracefulStop function
func gracefulStopAction(rt *Runtime) {
	rt.mu.Lock()
	rt.state = StateGracefulStopping
	rt.mu.Unlock()
	for fn := range rt.gracefulStopFnSeq() {
		cx := exec.CreateContext(rt.env)
		res, err := exec.Invoke(cx, fn, nil)
		if err != nil {
			writeStderr(rt.env, err.Error())
			rt.transition(StateImmediateStopping)
			return
		}
		if errVal, isErr := res.(*values.Error); isErr {
			writeStderr(rt.env, "error: "+errVal.Message+"\n")
			rt.transition(StateImmediateStopping)
			return
		}
	}
	rt.transition(StateStopped)
}

func (rt *Runtime) gracefulStopFnSeq() iter.Seq[*exec.InvokableHandle] {
	return func(yield func(*exec.InvokableHandle) bool) {
		for {
			rt.mu.Lock()
			if rt.state != StateGracefulStopping || len(rt.gracefulStopFns) == 0 {
				rt.mu.Unlock()
				break
			}
			fn := rt.gracefulStopFns[len(rt.gracefulStopFns)-1]
			rt.gracefulStopFns = rt.gracefulStopFns[:len(rt.gracefulStopFns)-1]
			rt.immediateStopFns = rt.immediateStopFns[:len(rt.immediateStopFns)-1]
			rt.mu.Unlock()
			if !yield(fn) {
				return
			}
		}
		for {
			rt.mu.Lock()
			if rt.state != StateGracefulStopping || len(rt.stopHandlers) == 0 {
				rt.mu.Unlock()
				return
			}
			fn := rt.stopHandlers[len(rt.stopHandlers)-1]
			rt.stopHandlers = rt.stopHandlers[:len(rt.stopHandlers)-1]
			rt.mu.Unlock()
			if !yield(fn) {
				return
			}
		}
	}
}

// immediateStopAction call immediateStop on all module listeners in the reverse order
// they got registered. should any of those functions return an error runtime will panic
func immediateStopAction(rt *Runtime) {
	onError := func(reason string) {
		rt.mu.Unlock()
		writeStderr(rt.env, fmt.Sprintf("panic: immediate stop failed due to %s\n", reason))
		rt.transition(StateStopped)
	}
	rt.mu.Lock()
	if rt.exitCode == 0 {
		rt.exitCode = 131 // 128  + SIGQUIT
	}
	rt.state = StateImmediateStopping
	for i := len(rt.immediateStopFns) - 1; i >= 0; i-- {
		fn := rt.immediateStopFns[i]
		cx := exec.CreateContext(rt.env)
		res, err := exec.Invoke(cx, fn, nil)
		if err != nil {
			onError(err.Error())
			return
		}
		if errVal, isErr := res.(*values.Error); isErr {
			onError(errVal.Message)
			return
		}
	}
	rt.mu.Unlock()
	rt.transition(StateStopped)
}

func stoppedAction(rt *Runtime) {
	rt.mu.Lock()
	if rt.state == StateStopped {
		rt.mu.Unlock()
		return
	}
	rt.state = StateStopped
	exitCode := rt.exitCode
	rt.mu.Unlock()
	rt.exitCodeChan <- exitCode
	close(rt.exitCodeChan)
}

// this assume no concurrent calls (fine given triggered from init)
func (rt *Runtime) setupSignalListeners() {
	if rt.listening {
		return
	}
	rt.listening = true
	ch := rt.env.Platform.Signals.Signals
	if ch == nil {
		panic("no signal channel in PAL")
	}
	go func(ch <-chan pal.Signal) {
		for {
			rt.mu.Lock()
			stopped := rt.state == StateStopped
			rt.mu.Unlock()
			if stopped {
				return
			}
			sig, ok := <-ch
			if !ok {
				return
			}
			switch sig {
			case pal.GracefulStop:
				rt.mu.Lock()
				if rt.exitCode == 0 {
					rt.exitCode = 130 // 128 + SIGINT
				}
				rt.mu.Unlock()
				go rt.transition(StateGracefulStopping)
			case pal.ImmediateStop:
				go rt.transition(StateImmediateStopping)
			default:
				panic("unknown signal")
			}
		}
	}(ch)
}

func (rt *Runtime) registerGracefulStopHandler(handler *exec.InvokableHandle) error {
	if !rt.mu.TryLock() {
		return fmt.Errorf("can't register graceful stop listeners during state transitions")
	}
	defer rt.mu.Unlock()
	if rt.state != StateInitializing {
		// Strictly speaking spec don't forbid this but the spirit of the spec https://github.com/ballerina-platform/ballerina-spec/issues/730#issuecomment-773018382
		// was not to allow this, and allowing this would add complications with reguard to reentrant locks in the current implementation, which I don't think
		// worth it to deal with (at the moment)
		return fmt.Errorf("registering graceful stop listeners outside of module init not supported")
	}
	rt.stopHandlers = append(rt.stopHandlers, handler)
	return nil
}

func writeStderr(env *extern.Env, s string) {
	if env.Platform.IO.Stderr == nil {
		panic("no stderr in PAL")
	}
	_, _ = env.Platform.IO.Stderr([]byte(s))
}
