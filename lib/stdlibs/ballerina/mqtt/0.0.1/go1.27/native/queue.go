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

import "sync"

// blockingQueue is an unbounded FIFO whose take blocks until an item arrives, mirroring the
// LinkedBlockingQueue the Java implementation uses between Paho callbacks and Ballerina.
type blockingQueue[T any] struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []T
	closed bool
}

func newBlockingQueue[T any]() *blockingQueue[T] {
	q := &blockingQueue[T]{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *blockingQueue[T]) put(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.items = append(q.items, item)
	q.cond.Signal()
}

// take returns false once the queue is closed and drained.
func (q *blockingQueue[T]) take() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	item := q.items[0]
	q.items[0] = zero
	q.items = q.items[1:]
	return item, true
}

func (q *blockingQueue[T]) clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = nil
}

func (q *blockingQueue[T]) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.items = nil
	q.cond.Broadcast()
}
