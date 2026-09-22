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

package values

import (
	"sync"
	"sync/atomic"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

var futureCompletionOrder atomic.Uint64

type futureOutcome struct {
	result BalValue
	panic  any
}

type futureState uint8

const (
	futureReady futureState = 1 << iota
	futureClaimed
)

// Future is a single-consumer completion state.
type Future struct {
	Type    semtypes.SemType
	ready   chan struct{}
	mu      sync.Mutex
	state   futureState
	ord     uint64
	outcome futureOutcome
}

func NewFuture(ty semtypes.SemType) *Future {
	return &Future{Type: ty, ready: make(chan struct{})}
}

func (f *Future) Complete(result BalValue, panicValue any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state&futureReady != 0 {
		return
	}
	f.outcome = futureOutcome{result: result, panic: panicValue}
	f.state |= futureReady
	f.ord = futureCompletionOrder.Add(1)
	close(f.ready)
}

// IsComplete is a "non blocking" lookup on whether future is completed or not. Use it determine if you should
// wait on the future or should yeild.
func (f *Future) IsComplete() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state&futureReady != 0
}

// Claim marks the future as waited and reports whether this wait claimed it.
func (f *Future) Claim() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state&futureClaimed != 0 {
		return false
	}
	f.state |= futureClaimed
	return true
}

// Get the completed value. This should be paired with IsComplete; Get blocks the current thread (all the strands on the thread)
// until future is ready.
func (f *Future) Get() BalValue {
	if !f.Claim() {
		return NewErrorWithMessage("multiple waits on the same future is not allowed")
	}
	return f.GetClaimed()
}

// GetClaimed returns the completed outcome of a future already claimed by this wait.
func (f *Future) GetClaimed() BalValue {
	<-f.ready
	if f.outcome.panic != nil {
		panic(f.outcome.panic)
	}
	return f.outcome.result
}

// IsAfter checks if the future was completed after other. This is best effort check. Caller must make sure both futures
// are complet (IsComplete) before calling this.
func (f *Future) IsAfter(other *Future) bool {
	// NOTE: this don't handle the overflowing case. In general you can always overflow with enough load and time.
	// Instead we assume both futures were completed close enough to each such that we can compare ord.
	return f.ord > other.ord
}
