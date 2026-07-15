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

import "github.com/ballerina-nutcracker/ballerina/semtypes"

type futureOutcome struct {
	result BalValue
	panic  any
}

// Future is a reusable completion state. Closing done publishes one immutable
// outcome to every current and subsequent waiter.
type Future struct {
	Type    semtypes.SemType
	done    chan struct{}
	outcome futureOutcome
}

func NewFuture(ty semtypes.SemType) *Future {
	return &Future{Type: ty, done: make(chan struct{})}
}

func (f *Future) Complete(result BalValue, panicValue any) {
	f.outcome = futureOutcome{result: result, panic: panicValue}
	close(f.done)
}

func (f *Future) Wait() BalValue {
	<-f.done
	if f.outcome.panic != nil {
		panic(f.outcome.panic)
	}
	return f.outcome.result
}
