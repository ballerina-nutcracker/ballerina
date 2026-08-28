// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 ( the "License"); you may not use this file except
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
	"testing"
)

func TestPublishCallsHandlersInRegistrationOrder(t *testing.T) {
	bus := New()
	defer bus.Close()
	var order []string
	bus.Subscribe(nil, func(Event) { order = append(order, "a") })
	bus.Subscribe(nil, func(Event) { order = append(order, "b") })
	bus.Subscribe(nil, func(Event) { order = append(order, "c") })
	bus.Publish(NewProjectRegisteredEvent("/root"))
	if got, want := join(order), "a,b,c"; got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestSubscribeFiltersByKind(t *testing.T) {
	bus := New()
	defer bus.Close()
	var got []Kind
	bus.Subscribe([]Kind{ProjectEvicted}, func(e Event) { got = append(got, e.Kind()) })
	bus.Subscribe([]Kind{ProjectRegistered, ProjectKindTransitioned}, func(e Event) { got = append(got, e.Kind()) })
	bus.Publish(NewProjectRegisteredEvent("/r"))
	bus.Publish(NewProjectEvictedEvent("/r", EvictionLRU))
	bus.Publish(NewProjectKindTransitionedEvent("/old", "/new"))
	// registered → second subscriber; evicted → first; transitioned → second
	want := []Kind{ProjectRegistered, ProjectEvicted, ProjectKindTransitioned}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d", len(got), len(want))
	}
	for i, k := range got {
		if k != want[i] {
			t.Errorf("event %d: kind %v, want %v", i, k, want[i])
		}
	}
}

func TestPanickingHandlerDoesNotStopSubsequent(t *testing.T) {
	bus := New()
	defer bus.Close()
	var after bool
	bus.Subscribe(nil, func(Event) { panic("boom") })
	bus.Subscribe(nil, func(Event) { after = true })
	bus.Publish(NewProjectRegisteredEvent("/r"))
	if !after {
		t.Fatal("handler after panic did not run")
	}
}

func TestSubscribeAllKindsOnEmptyFilter(t *testing.T) {
	bus := New()
	defer bus.Close()
	var count int
	bus.Subscribe(nil, func(Event) { count++ })
	bus.Publish(NewProjectRegisteredEvent("/r"))
	bus.Publish(NewProjectEvictedEvent("/r", EvictionLRU))
	bus.Publish(NewProjectKindTransitionedEvent("/o", "/n"))
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
}

func TestCloseStopsPublish(t *testing.T) {
	bus := New()
	bus.Close()
	var count int
	bus.Subscribe(nil, func(Event) { count++ })
	bus.Publish(NewProjectRegisteredEvent("/r"))
	if count != 0 {
		t.Fatalf("count = %d after Close, want 0", count)
	}
}

func TestEventValues(t *testing.T) {
	r := NewProjectRegisteredEvent("/root")
	if r.SourceRoot() != "/root" || r.Kind() != ProjectRegistered {
		t.Fatalf("registered event = %+v", r)
	}
	e := NewProjectEvictedEvent("/root", EvictionKindTransition)
	if e.SourceRoot() != "/root" || e.Reason() != EvictionKindTransition || e.Kind() != ProjectEvicted {
		t.Fatalf("evicted event = %+v", e)
	}
	k := NewProjectKindTransitionedEvent("/old", "/new")
	if k.OldRoot() != "/old" || k.NewRoot() != "/new" || k.SourceRoot() != "/new" || k.Kind() != ProjectKindTransitioned {
		t.Fatalf("transition event = %+v", k)
	}
}

func join(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}
