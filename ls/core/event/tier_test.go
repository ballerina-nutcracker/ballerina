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

func TestTieredCriticalDeliversAsync(t *testing.T) {
	bus := New()
	defer bus.Close()

	var mu sync.Mutex
	var received []uint64
	bus.SubscribeWithTier([]Kind{ProjectUpdated}, TierCritical, func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e.Generation())
	})

	bus.Publish(NewProjectUpdatedEvent("/r", 1))
	bus.Publish(NewProjectUpdatedEvent("/r", 2))
	bus.Flush()

	mu.Lock()
	defer mu.Unlock()
	want := []uint64{1, 2}
	if len(received) != len(want) {
		t.Fatalf("received %v, want %v", received, want)
	}
	for i, g := range received {
		if g != want[i] {
			t.Errorf("received[%d] = %d, want %d", i, g, want[i])
		}
	}
}

func TestTieredCriticalDropsOnFull(t *testing.T) {
	bus := New()
	defer bus.Close()

	block := make(chan struct{})
	var delivered int32
	var mu sync.Mutex
	bus.SubscribeWithTier([]Kind{ProjectUpdated}, TierCritical, func(e Event) {
		<-block
		mu.Lock()
		defer mu.Unlock()
		delivered++
	})

	for i := 0; i < 300; i++ {
		bus.Publish(NewProjectUpdatedEvent("/r", uint64(i+1)))
	}
	close(block)
	bus.Flush()

	mu.Lock()
	defer mu.Unlock()
	// Up to criticalCapacity queued events plus one in-flight event can arrive.
	if delivered >= 300 {
		t.Fatalf("expected some CRITICAL drops; delivered = %d", delivered)
	}
}

// Pending events with the same key are replaced; an in-flight event is not
// coalesced.
func TestTieredCoalesceableLastWins(t *testing.T) {
	bus := New()
	defer bus.Close()

	var mu sync.Mutex
	var received []uint64
	block := make(chan struct{})
	bus.SubscribeWithTier([]Kind{CompilationCancelled}, TierCoalesceable, func(e Event) {
		<-block
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e.Generation())
	})

	bus.Publish(NewCompilationCancelledEvent("/r", 1))
	bus.Publish(NewCompilationCancelledEvent("/r", 2))
	bus.Publish(NewCompilationCancelledEvent("/r", 3))
	close(block)
	bus.Flush()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || received[0] != 3 {
		t.Fatalf("received %v, want [3] (last-wins)", received)
	}
}

func TestTieredBestEffortHeadDrops(t *testing.T) {
	bus := New()
	defer bus.Close()

	var mu sync.Mutex
	var received []uint64
	block := make(chan struct{})
	bus.SubscribeWithTier([]Kind{CompilationDiagnosticsReady}, TierBestEffort, func(e Event) {
		<-block
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e.Generation())
	})

	for i := 0; i < 100; i++ {
		bus.Publish(NewCompilationDiagnosticsReadyEvent("/r", "pkg", uint64(i+1)))
	}
	close(block)
	bus.Flush()

	mu.Lock()
	defer mu.Unlock()
	// The ring keeps the newest bestEffortCapacity events, with at most one
	// additional event in flight.
	if len(received) >= 100 {
		t.Fatalf("expected best-effort drops; delivered = %d", len(received))
	}
	if len(received) == 0 {
		t.Fatal("expected at least one best-effort delivery")
	}
	last := received[len(received)-1]
	if last != 100 {
		t.Errorf("newest delivered = %d, want 100", last)
	}
}

func TestFlushIsNoOpForInlineSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()
	done := make(chan struct{})
	bus.Subscribe(nil, func(Event) { close(done) })
	bus.Publish(NewProjectRegisteredEvent("/r"))
	<-done
	bus.Flush()
}

func TestNewEventTypesGeneration(t *testing.T) {
	if g := NewProjectUpdatedEvent("/r", 7).Generation(); g != 7 {
		t.Fatalf("ProjectUpdated Generation = %d, want 7", g)
	}
	if g := NewProjectRegisteredEvent("/r").Generation(); g != 0 {
		t.Fatalf("08 event Generation = %d, want 0", g)
	}
	if g := NewCompilationSucceededEvent("/r", "pkg", 5).Generation(); g != 5 {
		t.Fatalf("CompilationSucceeded Generation = %d, want 5", g)
	}
	if d := NewCompilationSucceededEvent("/r", "pkg", 5).DescriptorName(); d != "pkg" {
		t.Fatalf("DescriptorName = %q, want pkg", d)
	}
	if g := NewResolutionDiagnosticsReadyEvent("/r", "pkg", 9).Generation(); g != 9 {
		t.Fatalf("ResolutionDiagnosticsReady Generation = %d, want 9", g)
	}
}
