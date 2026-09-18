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
		SpanID   string `json:"span_id"`
		ParentID string `json:"parent_id"`
	} `json:"args"`
}

// enabledEnv records the whole span tree; the recorder tests are about the
// tree, so every helper below assumes nesting is on.
func enabledEnv() *CompilerEnvironment {
	return NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{Enabled: true, Nested: true})
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

// recordSpan completes a root span with explicitly chosen offsets so assertions
// do not depend on wall-clock timing.
func recordSpan(state *traceState, label string, start, end time.Duration) uint64 {
	return recordChild(state, label, 0, start, end)
}

// recordChild completes a span parented to parent with explicitly chosen
// offsets. A parent of 0 records a root span.
func recordChild(state *traceState, label string, parent uint64, start, end time.Duration) uint64 {
	span := state.startSpan(label, parent)
	state.mu.Lock()
	state.spans = append(state.spans, traceSpan{
		label:  span.label,
		start:  start,
		end:    end,
		id:     span.id,
		parent: span.parent,
	})
	state.mu.Unlock()
	return span.id
}

func eventsByLabel(trace decodedTrace) map[string]decodedEvent {
	byLabel := make(map[string]decodedEvent, len(trace.TraceEvents))
	for _, event := range trace.TraceEvents {
		byLabel[event.Name] = event
	}
	return byLabel
}

func parentIndex(trace decodedTrace) map[string]string {
	parents := make(map[string]string, len(trace.TraceEvents))
	for _, event := range trace.TraceEvents {
		parents[event.Args.SpanID] = event.Args.ParentID
	}
	return parents
}

func isDecodedAncestor(parents map[string]string, ancestor, descendant string) bool {
	for id := parents[descendant]; id != ""; id = parents[id] {
		if id == ancestor {
			return true
		}
	}
	return false
}

// traceEpsilon absorbs the float rounding of recomputing an end offset as
// ts + dur; it is far below the one-nanosecond resolution of the recording.
const traceEpsilon = 1e-6

// assertLaneInvariants checks the two rendering guarantees on every lane: any
// two slices are disjoint or properly nested, and a nested pair is always a
// genuine ancestor-descendant pair.
func assertLaneInvariants(t *testing.T, trace decodedTrace) {
	t.Helper()
	parents := parentIndex(trace)
	byLane := make(map[int][]decodedEvent)
	for _, event := range trace.TraceEvents {
		byLane[event.Tid] = append(byLane[event.Tid], event)
	}
	for lane, events := range byLane {
		for i, outer := range events {
			for _, inner := range events[i+1:] {
				assertLanePairIsDisjointOrNested(t, lane, parents, outer, inner)
			}
		}
	}
}

func assertLanePairIsDisjointOrNested(
	t *testing.T, lane int, parents map[string]string, a, b decodedEvent) {
	t.Helper()
	aEnd, bEnd := a.Ts+a.Dur, b.Ts+b.Dur
	switch {
	case aEnd <= b.Ts+traceEpsilon || bEnd <= a.Ts+traceEpsilon:
		return
	case a.Ts <= b.Ts+traceEpsilon && bEnd <= aEnd+traceEpsilon:
		if !isDecodedAncestor(parents, a.Args.SpanID, b.Args.SpanID) {
			t.Fatalf("lane %d nests %q inside unrelated %q", lane, b.Name, a.Name)
		}
	case b.Ts <= a.Ts+traceEpsilon && aEnd <= bEnd+traceEpsilon:
		if !isDecodedAncestor(parents, b.Args.SpanID, a.Args.SpanID) {
			t.Fatalf("lane %d nests %q inside unrelated %q", lane, a.Name, b.Name)
		}
	default:
		t.Fatalf("lane %d overlaps %q and %q partially", lane, a.Name, b.Name)
	}
}

// assertContainment checks every child is time-contained in its parent.
func assertContainment(t *testing.T, trace decodedTrace) {
	t.Helper()
	byID := make(map[string]decodedEvent, len(trace.TraceEvents))
	for _, event := range trace.TraceEvents {
		byID[event.Args.SpanID] = event
	}
	for _, event := range trace.TraceEvents {
		if event.Args.ParentID == "" {
			continue
		}
		parent, ok := byID[event.Args.ParentID]
		if !ok {
			t.Fatalf("%q names parent %s which is not in the document", event.Name, event.Args.ParentID)
		}
		if event.Ts+traceEpsilon < parent.Ts ||
			event.Ts+event.Dur > parent.Ts+parent.Dur+traceEpsilon {
			t.Fatalf("%q is not contained in parent %q", event.Name, parent.Name)
		}
	}
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
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{})
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
// tiebreaks: spans that begin at the same offset order the longest first so a
// parent precedes a child, then fall back to span id, so a recording never
// reorders between marshals.
func TestSpansWithEqualStartsAreOrderedDeterministically(t *testing.T) {
	env := enabledEnv()
	// Recorded shortest-first so only the tiebreaks can produce the wanted order.
	recordSpan(&env.traceState, "short", 0, 5*time.Microsecond)
	recordSpan(&env.traceState, "long", 0, 20*time.Microsecond)
	recordSpan(&env.traceState, "same-extent-second", 0, 20*time.Microsecond)

	var names []string
	for _, event := range decodeTrace(t, env).TraceEvents {
		names = append(names, event.Name)
	}
	want := []string{"long", "same-extent-second", "short"}
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

func TestRootSpansCarryNoParentAndChildrenNameTheirParent(t *testing.T) {
	env := enabledEnv()
	cx := NewCompilerContext(env)

	root := cx.StartNamedSpan("Desugaring", "main.bal")
	child := root.StartChild("Function", "main")
	child.End()
	root.End()

	byLabel := eventsByLabel(decodeTrace(t, env))
	rootEvent, childEvent := byLabel["Desugaring main.bal"], byLabel["Function main"]
	if rootEvent.Args.ParentID != "" {
		t.Fatalf("root parent_id = %q, want no key", rootEvent.Args.ParentID)
	}
	if childEvent.Args.ParentID != rootEvent.Args.SpanID {
		t.Fatalf("child parent_id = %q, want %q", childEvent.Args.ParentID, rootEvent.Args.SpanID)
	}
}

func TestDisabledSpanChildrenRecordNothing(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{})
	cx := NewCompilerContext(env)

	span := cx.StartNamedSpan("Desugaring", "main.bal")
	child := span.StartChild("Function", "main")
	if child != (TraceSpan{}) {
		t.Fatalf("child of a disabled span = %+v, want the zero handle", child)
	}
	child.End()
	span.End()

	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if data != nil {
		t.Fatalf("disabled TraceJSON = %s, want nil", data)
	}
}

func TestSerialChildSharesItsParentLane(t *testing.T) {
	env := enabledEnv()
	parent := recordSpan(&env.traceState, "parent", 0, 20*time.Microsecond)
	recordChild(&env.traceState, "child", parent, 5*time.Microsecond, 15*time.Microsecond)

	trace := decodeTrace(t, env)
	byLabel := eventsByLabel(trace)
	if byLabel["child"].Tid != byLabel["parent"].Tid {
		t.Fatalf("child lane = %d, want the parent lane %d", byLabel["child"].Tid, byLabel["parent"].Tid)
	}
	assertContainment(t, trace)
	assertLaneInvariants(t, trace)
}

func TestOverlappingSiblingsTakeDistinctLanesAndKeepTheirParent(t *testing.T) {
	env := enabledEnv()
	parent := recordSpan(&env.traceState, "parent", 0, 30*time.Microsecond)
	recordChild(&env.traceState, "first", parent, 2*time.Microsecond, 20*time.Microsecond)
	recordChild(&env.traceState, "second", parent, 5*time.Microsecond, 25*time.Microsecond)

	trace := decodeTrace(t, env)
	byLabel := eventsByLabel(trace)
	if byLabel["first"].Tid == byLabel["second"].Tid {
		t.Fatalf("overlapping siblings share lane %d", byLabel["first"].Tid)
	}
	parentID := byLabel["parent"].Args.SpanID
	for _, label := range []string{"first", "second"} {
		if byLabel[label].Args.ParentID != parentID {
			t.Fatalf("%s parent_id = %q, want %q", label, byLabel[label].Args.ParentID, parentID)
		}
	}
	assertContainment(t, trace)
	assertLaneInvariants(t, trace)
}

func TestChildAfterASiblingEndsRejoinsTheParentLane(t *testing.T) {
	env := enabledEnv()
	parent := recordSpan(&env.traceState, "parent", 0, 40*time.Microsecond)
	recordChild(&env.traceState, "first", parent, 2*time.Microsecond, 10*time.Microsecond)
	recordChild(&env.traceState, "second", parent, 5*time.Microsecond, 12*time.Microsecond)
	recordChild(&env.traceState, "third", parent, 15*time.Microsecond, 20*time.Microsecond)

	trace := decodeTrace(t, env)
	byLabel := eventsByLabel(trace)
	if byLabel["third"].Tid != byLabel["parent"].Tid {
		t.Fatalf("third lane = %d, want the parent lane %d", byLabel["third"].Tid, byLabel["parent"].Tid)
	}
	assertContainment(t, trace)
	assertLaneInvariants(t, trace)
}

func TestSpanInsideAnUnrelatedSpanIsNeverNestedUnderIt(t *testing.T) {
	env := enabledEnv()
	recordSpan(&env.traceState, "unrelated", 0, 30*time.Microsecond)
	recordSpan(&env.traceState, "inner", 5*time.Microsecond, 10*time.Microsecond)

	trace := decodeTrace(t, env)
	byLabel := eventsByLabel(trace)
	if byLabel["inner"].Tid == byLabel["unrelated"].Tid {
		t.Fatalf("unrelated spans share lane %d", byLabel["inner"].Tid)
	}
	assertLaneInvariants(t, trace)
}

func TestDeepChainsPreserveContainmentAndLaneInvariants(t *testing.T) {
	env := enabledEnv()
	root := recordSpan(&env.traceState, "root", 0, 100*time.Microsecond)
	phase := recordChild(&env.traceState, "phase", root, 5*time.Microsecond, 80*time.Microsecond)
	analysis := recordChild(&env.traceState, "analysis", phase, 10*time.Microsecond, 60*time.Microsecond)
	recordChild(&env.traceState, "fn1", analysis, 12*time.Microsecond, 30*time.Microsecond)
	recordChild(&env.traceState, "fn2", analysis, 20*time.Microsecond, 40*time.Microsecond)
	recordChild(&env.traceState, "fn3", analysis, 45*time.Microsecond, 50*time.Microsecond)
	recordSpan(&env.traceState, "other root", 110*time.Microsecond, 120*time.Microsecond)

	trace := decodeTrace(t, env)
	byLabel := eventsByLabel(trace)
	for _, label := range []string{"phase", "analysis", "fn1", "fn3"} {
		if byLabel[label].Tid != byLabel["root"].Tid {
			t.Fatalf("%s lane = %d, want the serial lane %d", label, byLabel[label].Tid, byLabel["root"].Tid)
		}
	}
	if byLabel["fn2"].Tid == byLabel["fn1"].Tid {
		t.Fatalf("overlapping fn1 and fn2 share lane %d", byLabel["fn2"].Tid)
	}
	assertContainment(t, trace)
	assertLaneInvariants(t, trace)
}

func TestConcurrentChildrenOfOneParentAreRecordedLosslessly(t *testing.T) {
	env := enabledEnv()
	cx := NewCompilerContext(env)
	root := cx.StartNamedSpan("Desugaring", "main.bal")

	const goroutines = 16
	const perGoroutine = 8
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				root.StartChild("Function", "main").End()
			}
		}()
	}
	wg.Wait()
	root.End()

	trace := decodeTrace(t, env)
	if len(trace.TraceEvents) != goroutines*perGoroutine+1 {
		t.Fatalf("events = %d, want %d", len(trace.TraceEvents), goroutines*perGoroutine+1)
	}
	var rootID string
	children := 0
	for _, event := range trace.TraceEvents {
		if event.Args.ParentID == "" {
			rootID = event.Args.SpanID
		}
	}
	for _, event := range trace.TraceEvents {
		if event.Args.SpanID == rootID {
			continue
		}
		if event.Args.ParentID != rootID {
			t.Fatalf("child parent_id = %q, want %q", event.Args.ParentID, rootID)
		}
		children++
	}
	if children != goroutines*perGoroutine {
		t.Fatalf("children = %d, want %d", children, goroutines*perGoroutine)
	}
	assertContainment(t, trace)
	assertLaneInvariants(t, trace)
}

// TestUnnestedRecordingKeepsOnlyRootSpans covers the default recording: phases
// are recorded, the children they start are not, and nothing carries parentage.
func TestUnnestedRecordingKeepsOnlyRootSpans(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{Enabled: true})
	cx := NewCompilerContext(env)

	root := cx.StartNamedSpan("Desugaring", "main.bal")
	child := root.StartChild("Class", "Counter")
	if child != (TraceSpan{}) {
		t.Fatalf("child of an unnested recording = %+v, want the zero handle", child)
	}
	// A grandchild of a disabled handle stays disabled, so a whole subtree
	// costs nothing once its root child is refused.
	child.StartChild("Method", "add").End()
	child.End()
	root.End()

	events := decodeTrace(t, env).TraceEvents
	if len(events) != 1 {
		t.Fatalf("events = %d, want only the root span", len(events))
	}
	if events[0].Name != "Desugaring main.bal" {
		t.Fatalf("recorded %q, want the root span", events[0].Name)
	}
	if events[0].Args.ParentID != "" {
		t.Fatalf("root parent_id = %q, want no key", events[0].Args.ParentID)
	}
}

// TestUnnestedRecordingLanesRootsWithoutNesting covers the rendering of a
// recording with no parentage: overlapping roots take their own lanes and a
// freed lane is reused, exactly as before children existed.
func TestUnnestedRecordingLanesRootsWithoutNesting(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{Enabled: true})
	recordSpan(&env.traceState, "A", 0, 10*time.Microsecond)
	recordSpan(&env.traceState, "B", 5*time.Microsecond, 20*time.Microsecond)
	recordSpan(&env.traceState, "C", 12*time.Microsecond, 15*time.Microsecond)

	byLabel := eventsByLabel(decodeTrace(t, env))
	if byLabel["B"].Tid == byLabel["A"].Tid {
		t.Fatalf("overlapping roots share lane %d", byLabel["B"].Tid)
	}
	if byLabel["C"].Tid != byLabel["A"].Tid {
		t.Fatalf("C lane = %d, want the freed lane %d", byLabel["C"].Tid, byLabel["A"].Tid)
	}
}
