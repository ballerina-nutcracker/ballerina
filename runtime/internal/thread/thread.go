// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package thread

import (
	"container/list"
	"runtime"
	"sync"
)

type continuation chan struct{}

type Thread struct {
	mu    sync.Mutex
	queue *list.List
}

func New() *Thread {
	return &Thread{queue: list.New()}
}

func (t *Thread) Schedule() <-chan struct{} {
	ch := make(continuation, 1)
	t.mu.Lock()
	t.queue.PushBack(ch)
	t.mu.Unlock()
	return ch
}

func (t *Thread) Yield() <-chan struct{} {
	ch := make(continuation, 1)
	t.mu.Lock()
	t.queue.PushBack(ch)
	t.signalHead()
	t.mu.Unlock()
	// A lone strand immediately schedules itself again. Yield to Go as well so
	// isolated strands can run on single-threaded targets such as WASM.
	runtime.Gosched()
	return ch
}

func (t *Thread) Complete() {
	t.mu.Lock()
	t.signalHead()
	t.mu.Unlock()
}

func (t *Thread) signalHead() {
	head := t.queue.Front()
	if head == nil {
		return
	}
	t.queue.Remove(head)
	head.Value.(continuation) <- struct{}{}
}
