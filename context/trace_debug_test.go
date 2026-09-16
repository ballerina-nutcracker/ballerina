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

//go:build debug

package context

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type decodedTrace struct {
	TraceEvents []decodedEvent `json:"traceEvents"`
}

type decodedEvent struct {
	Name string  `json:"name"`
	Cat  string  `json:"cat"`
	Ph   string  `json:"ph"`
	Pid  int     `json:"pid"`
	Tid  int     `json:"tid"`
	Ts   float64 `json:"ts"`
	Dur  float64 `json:"dur"`
	Args struct {
		SpanID string `json:"span_id"`
	} `json:"args"`
}

func enabledEnv() *CompilerEnvironment {
	return NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)
}

func decodeTrace(t *testing.T, env *CompilerEnvironment) decodedTrace {
	t.Helper()
	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	var trace decodedTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	return trace
}

// recordSpan completes a span with explicitly chosen offsets so assertions do
// not depend on wall-clock timing.
func recordSpan(state *traceState, label string, start, end time.Duration) uint64 {
	span := state.startSpan(label)
	state.mu.Lock()
	state.spans = append(state.spans, traceSpan{
		label: span.label,
		start: start,
		end:   end,
		id:    span.id,
	})
	state.mu.Unlock()
	return span.id
}

func TestEmptyRecordingMarshalsEmptyTraceEvents(t *testing.T) {
	env := enabledEnv()
	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if string(data) != `{"traceEvents":[]}` {
		t.Fatalf("empty trace = %s", data)
	}
}

func TestDisabledRecordingCollectsNothing(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := NewCompilerContext(env)

	cx.StartNamedSpan("Parse", "main.bal").End()
	cx.StartPackageSpan("Desugaring", model.NewPackageID(
		model.DefaultPackageIDInterner, "myorg", []model.Name{"mymod"}, "1.0.0")).End()

	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if data != nil {
		t.Fatalf("disabled TraceJSON = %s, want nil", data)
	}
}

func TestSpanIDsAreUniqueAndNonZero(t *testing.T) {
	env := enabledEnv()
	cx := NewCompilerContext(env)

	seen := make(map[string]bool)
	for range 4 {
		span := cx.StartNamedSpan("Parse", "main.bal")
		span.End()
	}
	for _, event := range decodeTrace(t, env).TraceEvents {
		if event.Args.SpanID == "" || event.Args.SpanID == "0" {
			t.Fatalf("span id = %q, want a nonzero id", event.Args.SpanID)
		}
		if seen[event.Args.SpanID] {
			t.Fatalf("duplicate span id %s", event.Args.SpanID)
		}
		seen[event.Args.SpanID] = true
	}
	if len(seen) != 4 {
		t.Fatalf("recorded %d spans, want 4", len(seen))
	}
}

// TestSpanLabelsCombineOperationAndIdentity covers the label composition every
// instrumented stage shares, including package assembly's identity-less span.
func TestSpanLabelsCombineOperationAndIdentity(t *testing.T) {
	env := enabledEnv()
	cx := NewCompilerContext(env)
	pkgID := model.NewPackageID(
		model.DefaultPackageIDInterner, "myorg", []model.Name{"mymod"}, "1.0.0")

	cx.StartNamedSpan("Parse", "main.bal").End()
	cx.StartNamedSpan("Package Assembly", "").End()
	cx.StartPackageSpan("Desugaring", pkgID).End()
	cx.StartPackageSpan("Package Assembly", nil).End()

	var names []string
	for _, event := range decodeTrace(t, env).TraceEvents {
		names = append(names, event.Name)
	}
	want := []string{
		"Parse main.bal",
		"Package Assembly",
		"Desugaring myorg/mymod:1.0.0",
		"Package Assembly",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("labels = %q, want %q", names, want)
	}
}

func TestOffsetsKeepFractionalMicroseconds(t *testing.T) {
	env := enabledEnv()
	recordSpan(&env.traceState, "Parse main.bal", 1500*time.Nanosecond, 3250*time.Nanosecond)

	events := decodeTrace(t, env).TraceEvents
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Ts != 1.5 {
		t.Fatalf("ts = %v, want 1.5", events[0].Ts)
	}
	if events[0].Dur != 1.75 {
		t.Fatalf("dur = %v, want 1.75", events[0].Dur)
	}
	if events[0].Cat != "frontend" || events[0].Ph != "X" || events[0].Pid != tracePid {
		t.Fatalf("unexpected event envelope: %+v", events[0])
	}
}

func TestSpansAreOrderedAndLanedWithoutFalseNesting(t *testing.T) {
	env := enabledEnv()
	// Two overlapping spans plus one that starts after the first ends.
	recordSpan(&env.traceState, "A", 0, 10*time.Microsecond)
	recordSpan(&env.traceState, "B", 5*time.Microsecond, 20*time.Microsecond)
	recordSpan(&env.traceState, "C", 12*time.Microsecond, 15*time.Microsecond)

	events := decodeTrace(t, env).TraceEvents
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}
	for i, want := range []string{"A", "B", "C"} {
		if events[i].Name != want {
			t.Fatalf("event %d = %s, want %s", i, events[i].Name, want)
		}
	}
	if events[0].Tid != 0 {
		t.Fatalf("A lane = %d, want 0", events[0].Tid)
	}
	if events[1].Tid == events[0].Tid {
		t.Fatalf("overlapping spans A and B share lane %d", events[1].Tid)
	}
	if events[2].Tid != events[0].Tid {
		t.Fatalf("C lane = %d, want the freed lane %d", events[2].Tid, events[0].Tid)
	}
}

// TestSpansWithEqualStartsAreOrderedDeterministically covers the sort
// tiebreaks: spans that begin at the same offset fall back to end offset and
// then to span id, so a recording never reorders between marshals.
func TestSpansWithEqualStartsAreOrderedDeterministically(t *testing.T) {
	env := enabledEnv()
	// Recorded longest-first so only the tiebreaks can produce the wanted order.
	recordSpan(&env.traceState, "long", 0, 20*time.Microsecond)
	recordSpan(&env.traceState, "short", 0, 5*time.Microsecond)
	recordSpan(&env.traceState, "same-extent-second", 0, 20*time.Microsecond)

	var names []string
	for _, event := range decodeTrace(t, env).TraceEvents {
		names = append(names, event.Name)
	}
	want := []string{"short", "long", "same-extent-second"}
	if !slices.Equal(names, want) {
		t.Fatalf("order = %q, want %q", names, want)
	}
}

func TestLabelsAreJSONEscaped(t *testing.T) {
	env := enabledEnv()
	cx := NewCompilerContext(env)
	label := `Parse "quoted"\path.bal`

	span := cx.StartNamedSpan(label, "")
	span.End()

	events := decodeTrace(t, env).TraceEvents
	if len(events) != 1 || events[0].Name != label {
		t.Fatalf("round-tripped label = %+v, want %q", events, label)
	}
}

func TestConcurrentSpansAreRecordedLosslessly(t *testing.T) {
	env := enabledEnv()
	cx := NewCompilerContext(env)

	const goroutines = 32
	const perGoroutine = 16
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				span := cx.StartNamedSpan("Parse", "main.bal")
				span.End()
			}
		}()
	}
	wg.Wait()

	events := decodeTrace(t, env).TraceEvents
	if len(events) != goroutines*perGoroutine {
		t.Fatalf("events = %d, want %d", len(events), goroutines*perGoroutine)
	}
}

func TestMarshalTraceIsRepeatableAndDeterministic(t *testing.T) {
	env := enabledEnv()
	recordSpan(&env.traceState, "A", 0, 10*time.Microsecond)
	recordSpan(&env.traceState, "B", 5*time.Microsecond, 20*time.Microsecond)

	first, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	second, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("trace is not deterministic:\n%s\n%s", first, second)
	}
}
