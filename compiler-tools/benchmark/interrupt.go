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

package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// interruptExitCode is the conventional 128+SIGINT status for a signalled
// process.
const interruptExitCode = 130

// onInterrupt registers cleanup to run on SIGINT/SIGTERM before the process
// exits, covering the gap deferred cleanup leaves when the process is
// signalled. Every live registration runs before that exit, so cleanup must be
// idempotent: a signal arriving as the caller returns races its own deferred
// cleanup. The returned stop deregisters and is safe to call more than once.
func onInterrupt(cleanup func()) (stop func()) {
	return interrupts.register(cleanup)
}

// interrupts is the process-wide interrupt coordinator. Registrations overlap
// — the test binary installs one in TestMain and another inside run — and a
// single handler keeps one os.Exit from truncating another's cleanup.
var interrupts interruptCoordinator

type interruptCoordinator struct {
	mu    sync.Mutex
	ch    chan os.Signal
	done  chan struct{}
	steps []*cleanupStep
}

func (i *interruptCoordinator) register(cleanup func()) (stop func()) {
	step := newCleanupStep(cleanup)
	i.mu.Lock()
	if i.ch == nil {
		i.ch = make(chan os.Signal, 1)
		i.done = make(chan struct{})
		signal.Notify(i.ch, os.Interrupt, syscall.SIGTERM)
		go i.wait(i.ch, i.done)
	}
	i.steps = append(i.steps, step)
	i.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { i.deregister(step) }) }
}

// deregister drops step and, once nothing is left to clean up, releases the
// handler so a later signal terminates the process the default way.
func (i *interruptCoordinator) deregister(step *cleanupStep) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for n, other := range i.steps {
		if other == step {
			i.steps = append(i.steps[:n], i.steps[n+1:]...)
			break
		}
	}
	if len(i.steps) > 0 || i.ch == nil {
		return
	}
	signal.Stop(i.ch)
	close(i.done)
	i.ch, i.done = nil, nil
}

func (i *interruptCoordinator) wait(ch chan os.Signal, done chan struct{}) {
	select {
	case <-ch:
	case <-done:
		return
	}
	// Restore the previous disposition before the (potentially slow) cleanup,
	// so a second signal is acted on immediately instead of queueing behind it.
	signal.Reset(os.Interrupt, syscall.SIGTERM)
	i.mu.Lock()
	steps := i.steps
	i.steps = nil
	i.mu.Unlock()
	for n := len(steps) - 1; n >= 0; n-- {
		steps[n].run()
	}
	os.Exit(interruptExitCode)
}

// cleanupStep is one teardown step. The sync.Once makes it idempotent, so the
// deferred path and a racing signal handler cannot run it twice; the loser of
// that race blocks until the winner is done.
type cleanupStep struct {
	once sync.Once
	fn   func()
}

func newCleanupStep(fn func()) *cleanupStep {
	return &cleanupStep{fn: fn}
}

func (s *cleanupStep) run() {
	s.once.Do(s.fn)
}

// cleanups is a teardown stack shared between the deferred path and the
// interrupt handler. It is guarded because the handler reads it from another
// goroutine while the caller is still registering steps.
type cleanups struct {
	mu    sync.Mutex
	steps []*cleanupStep
}

func (c *cleanups) add(fn func()) {
	step := newCleanupStep(fn)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.steps = append(c.steps, step)
}

// run executes the registered steps in reverse order and drops them, so the
// deferred path and a racing signal cannot run the same step twice.
func (c *cleanups) run() {
	c.mu.Lock()
	steps := c.steps
	c.steps = nil
	c.mu.Unlock()
	for i := len(steps) - 1; i >= 0; i-- {
		steps[i].run()
	}
}
