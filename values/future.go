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

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type futureOutcome struct {
	result BalValue
	panic  any
}

type futureState uint8

const (
	pending futureState = iota
	ready
	claimed
)

// Future is a single-consumer completion state.
type Future struct {
	Type    semtypes.SemType
	ready   chan struct{}
	mu      sync.Mutex
	state   futureState
	outcome futureOutcome
}

func NewFuture(ty semtypes.SemType) *Future {
	return &Future{Type: ty, ready: make(chan struct{})}
}

func (f *Future) Complete(result BalValue, panicValue any) {
	f.mu.Lock()
	f.outcome = futureOutcome{result: result, panic: panicValue}
	f.state = ready
	f.mu.Unlock()
	close(f.ready)
}

// IsComplete is a "non blocking" lookup on whether future is completed or not. Use it determine if you should
// wait on the future or should yeild.
func (f *Future) IsComplete() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state != pending
}

// Get the completed value. This should be paired with IsComplete; Get blocks the current thread (all the strands on the thread)
// until future is ready.
func (f *Future) Get() BalValue {
	<-f.ready
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state == claimed {
		return NewErrorWithMessage("multiple waits on the same future is not allowed")
	}
	f.state = claimed
	if f.outcome.panic != nil {
		panic(f.outcome.panic)
	}
	return f.outcome.result
}
