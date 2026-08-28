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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package event

import (
	"sync"
	"testing"
)

// recordingLogf collects every call so tests can assert on message/attrs
// without depending on any particular logging library.
type recordingLogf struct {
	mu    sync.Mutex
	calls []logCall
}

type logCall struct {
	msg  string
	args []any
}

func (r *recordingLogf) logf(msg string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, logCall{msg: msg, args: args})
}

func (r *recordingLogf) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// TestDefaultLogfIsNoop verifies New() without WithLogf never panics on a
// subscriber panic even though nothing consumes the drop/panic message — the
// ~15 existing New() call sites elsewhere in the repo must not need to change.
func TestDefaultLogfIsNoop(t *testing.T) {
	bus := New()
	defer bus.Close()
	bus.Subscribe(nil, func(Event) { panic("boom") })
	bus.Publish(NewProjectRegisteredEvent("/r"))
}

// TestWithLogfReceivesInlineSubscriberPanic verifies an inline (Subscribe)
// subscriber panic is reported through the injected Logf, not bare log.Printf.
func TestWithLogfReceivesInlineSubscriberPanic(t *testing.T) {
	var rec recordingLogf
	bus := New(WithLogf(rec.logf))
	defer bus.Close()
	bus.Subscribe(nil, func(Event) { panic("boom") })
	bus.Publish(NewProjectRegisteredEvent("/r"))
	if rec.count() != 1 {
		t.Fatalf("Logf calls = %d, want 1", rec.count())
	}
}

// TestWithLogfReceivesCriticalDrop verifies a dropped CRITICAL event (channel
// full) is reported through the injected Logf.
func TestWithLogfReceivesCriticalDrop(t *testing.T) {
	var rec recordingLogf
	bus := New(WithLogf(rec.logf))
	defer bus.Close()

	block := make(chan struct{})
	bus.SubscribeWithTier([]Kind{ProjectUpdated}, TierCritical, func(e Event) {
		<-block
	})
	for i := 0; i < 300; i++ {
		bus.Publish(NewProjectUpdatedEvent("/r", uint64(i+1)))
	}
	close(block)
	bus.Flush()

	if rec.count() == 0 {
		t.Fatal("Logf calls = 0, want at least one CRITICAL-drop report")
	}
}

// TestWithLogfReceivesTieredSubscriberPanic verifies a tiered (async)
// subscriber panic is reported through the injected Logf.
func TestWithLogfReceivesTieredSubscriberPanic(t *testing.T) {
	var rec recordingLogf
	bus := New(WithLogf(rec.logf))
	defer bus.Close()
	bus.SubscribeWithTier([]Kind{ProjectUpdated}, TierCritical, func(Event) { panic("boom") })
	bus.Publish(NewProjectUpdatedEvent("/r", 1))
	bus.Flush()
	if rec.count() != 1 {
		t.Fatalf("Logf calls = %d, want 1", rec.count())
	}
}
