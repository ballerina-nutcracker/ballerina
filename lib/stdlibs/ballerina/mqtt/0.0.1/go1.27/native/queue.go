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

// queueReader is one consumer of a shared blockingQueue that can be closed on its own.
type queueReader[T any] struct {
	queue *blockingQueue[T]
	done  bool
}

func newBlockingQueue[T any]() *blockingQueue[T] {
	q := &blockingQueue[T]{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// put reports false when the queue is already closed and the item was dropped.
func (q *blockingQueue[T]) put(item T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	q.items = append(q.items, item)
	q.cond.Signal()
	return true
}

// take returns false once the queue is closed and drained.
func (q *blockingQueue[T]) take() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.takeLocked(nil)
}

func (q *blockingQueue[T]) takeLocked(done *bool) (T, bool) {
	stopped := func() bool { return done != nil && *done }
	for len(q.items) == 0 && !q.closed && !stopped() {
		q.cond.Wait()
	}
	var zero T
	if len(q.items) == 0 || stopped() {
		return zero, false
	}
	item := q.items[0]
	q.items[0] = zero
	q.items = q.items[1:]
	return item, true
}

func (q *blockingQueue[T]) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.cond.Broadcast()
}

func (q *blockingQueue[T]) newReader() *queueReader[T] {
	return &queueReader[T]{queue: q}
}

// take returns false once the reader or the queue is closed.
func (r *queueReader[T]) take() (T, bool) {
	r.queue.mu.Lock()
	defer r.queue.mu.Unlock()
	return r.queue.takeLocked(&r.done)
}

// close drops the queue's pending items, as the Java closeStream does, and releases only the
// takes blocked on this reader; other readers of the queue keep waiting.
func (r *queueReader[T]) close() {
	q := r.queue
	q.mu.Lock()
	defer q.mu.Unlock()
	r.done = true
	q.items = nil
	q.cond.Broadcast()
}
