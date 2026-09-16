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
	"cmp"
	"encoding/json"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ballerina-nutcracker/ballerina/model"
)

// traceCategory is the Chrome Trace Event category shared by every span.
const traceCategory = "frontend"

// tracePid is the single logical process every span is emitted under; tracks
// are logical viewer lanes rather than OS threads.
const tracePid = 1

// TraceSpan is a handle to an in-flight frontend invocation span. The zero
// value is a disabled handle: it is safe to use and to end.
type TraceSpan struct {
	state *traceState
	label string
	start time.Duration
	id    uint64
}

// traceSpan is a completed span held by the recorder.
type traceSpan struct {
	label string
	start time.Duration
	end   time.Duration
	id    uint64
}

// traceState is the environment-owned recorder.
type traceState struct {
	enabled bool
	origin  time.Time
	nextID  atomic.Uint64
	mu      sync.Mutex
	spans   []traceSpan
}

// traceEventArgs carries the span's identity. The ID is a decimal string so
// JSON readers that parse numbers as float64 cannot lose precision.
type traceEventArgs struct {
	SpanID string `json:"span_id"`
}

// traceEvent is a Chrome Trace Event complete-duration event.
type traceEvent struct {
	Name string         `json:"name"`
	Cat  string         `json:"cat"`
	Ph   string         `json:"ph"`
	Pid  int            `json:"pid"`
	Tid  int            `json:"tid"`
	Ts   float64        `json:"ts"`
	Dur  float64        `json:"dur"`
	Args traceEventArgs `json:"args"`
}

// traceDocument is the Chrome Trace Event JSON object.
type traceDocument struct {
	TraceEvents []traceEvent `json:"traceEvents"`
}

func newTraceState(traceEnabled bool) traceState {
	if !traceEnabled {
		return traceState{}
	}
	return traceState{enabled: true, origin: time.Now()}
}

func (s *traceState) startSpan(label string) TraceSpan {
	return TraceSpan{
		state: s,
		label: label,
		start: time.Since(s.origin),
		id:    s.nextID.Add(1),
	}
}

func (s *traceState) endSpan(span TraceSpan) {
	// Read the clock before contending for the mutex so lock waits are not
	// charged to the measured invocation.
	end := time.Since(s.origin)
	completed := traceSpan{
		label: span.label,
		start: span.start,
		end:   end,
		id:    span.id,
	}
	s.mu.Lock()
	s.spans = append(s.spans, completed)
	s.mu.Unlock()
}

// marshalTrace sorts the completed recording, assigns viewer lanes, and encodes
// it as a Chrome Trace Event document. It performs no filesystem access.
func (s *traceState) marshalTrace() ([]byte, error) {
	if !s.enabled {
		return nil, nil
	}
	s.mu.Lock()
	spans := slices.Clone(s.spans)
	s.mu.Unlock()

	slices.SortFunc(spans, compareTraceSpans)

	events := make([]traceEvent, 0, len(spans))
	var laneEnds []time.Duration
	for _, span := range spans {
		events = append(events, newTraceEvent(span, assignTraceLane(&laneEnds, span)))
	}
	return json.Marshal(traceDocument{TraceEvents: events})
}

func compareTraceSpans(a, b traceSpan) int {
	if c := cmp.Compare(a.start, b.start); c != 0 {
		return c
	}
	if c := cmp.Compare(a.end, b.end); c != 0 {
		return c
	}
	return cmp.Compare(a.id, b.id)
}

// assignTraceLane returns the lowest lane whose preceding span ends at or
// before this span's start, allocating a new lane when none is free. Overlapping
// spans therefore land on distinct lanes instead of implying false nesting.
func assignTraceLane(laneEnds *[]time.Duration, span traceSpan) int {
	for lane, end := range *laneEnds {
		if end <= span.start {
			(*laneEnds)[lane] = span.end
			return lane
		}
	}
	*laneEnds = append(*laneEnds, span.end)
	return len(*laneEnds) - 1
}

func newTraceEvent(span traceSpan, lane int) traceEvent {
	return traceEvent{
		Name: span.label,
		Cat:  traceCategory,
		Ph:   "X",
		Pid:  tracePid,
		Tid:  lane,
		Ts:   microseconds(span.start),
		Dur:  microseconds(span.end - span.start),
		Args: traceEventArgs{SpanID: strconv.FormatUint(span.id, 10)},
	}
}

// microseconds keeps fractional microseconds rather than rounding to whole ones.
func microseconds(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000
}

// StartNamedSpan starts a root span for a frontend invocation, labeled with the
// operation and a free-form identity such as a source file path. An empty
// identity labels the span with the operation alone.
func (c *CompilerContext) StartNamedSpan(operation, identity string) TraceSpan {
	if !c.env.traceState.enabled {
		return TraceSpan{}
	}
	return c.env.traceState.startSpan(spanLabel(operation, identity))
}

// StartPackageSpan starts a root span for a frontend invocation over one
// package, labeled with the operation and the package's identity.
//
// pkgID may be nil: package assembly runs before any package identity is
// established, so it is the one invocation that can have none.
func (c *CompilerContext) StartPackageSpan(operation string, pkgID *model.PackageID) TraceSpan {
	if !c.env.traceState.enabled {
		return TraceSpan{}
	}
	return c.env.traceState.startSpan(spanLabel(operation, packageIdentity(pkgID)))
}

// packageIdentity renders a package's identity as org/name:version. Every
// PackageID comes from model.NewPackageID, which always sets all three.
func packageIdentity(pkgID *model.PackageID) string {
	if pkgID == nil {
		return ""
	}
	return pkgID.OrgName.Value() + "/" + pkgID.Name.Value() + ":" + pkgID.Version.Value()
}

// spanLabel joins the operation with the identity it applies to.
func spanLabel(operation, identity string) string {
	if identity == "" {
		return operation
	}
	return operation + " " + identity
}

// End completes the span. Each active handle is ended exactly once by its owner.
func (s TraceSpan) End() {
	if s.state == nil {
		return
	}
	s.state.endSpan(s)
}

// TraceJSON returns the completed recording as Chrome Trace Event JSON bytes.
// It is called after frontend work has stopped; the driver owns output.
func (c *CompilerEnvironment) TraceJSON() ([]byte, error) {
	return c.traceState.marshalTrace()
}
