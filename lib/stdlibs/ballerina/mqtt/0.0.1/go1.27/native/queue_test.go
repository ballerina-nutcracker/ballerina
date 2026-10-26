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
	"testing"
	"time"
)

type takeResult struct {
	item string
	ok   bool
}

func startTake(r *queueReader[string]) <-chan takeResult {
	ch := make(chan takeResult, 1)
	go func() {
		item, ok := r.take()
		ch <- takeResult{item, ok}
	}()
	return ch
}

// A Ballerina program cannot close a stream that another strand is blocked on: a stream is not an
// isolated value, so only strands sharing main's thread can hold it, and a blocked take holds that
// thread. The release of the closed reader's take is therefore checked here.
func TestQueueReaderCloseReleasesOnlyItsTakers(t *testing.T) {
	q := newBlockingQueue[string]()
	closed, open := q.newReader(), q.newReader()
	fromClosed, fromOpen := startTake(closed), startTake(open)
	time.Sleep(50 * time.Millisecond)

	closed.close()
	select {
	case got := <-fromClosed:
		if got.ok {
			t.Fatalf("closed reader got %+v, want end of stream", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("closing a reader did not release its blocked take")
	}
	select {
	case got := <-fromOpen:
		t.Fatalf("closing one reader released another reader's take with %+v", got)
	case <-time.After(100 * time.Millisecond):
	}

	q.put("after")
	select {
	case got := <-fromOpen:
		if got != (takeResult{"after", true}) {
			t.Fatalf("open reader got %+v, want after", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("open reader did not receive the later item")
	}
	if _, ok := closed.take(); ok {
		t.Fatal("take on a closed reader returned an item")
	}
}
