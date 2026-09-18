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
// value is a disabled handle: it is safe to use, to start children from, and to
// end.
type TraceSpan struct {
	state  *traceState
	label  string
	start  time.Duration
	id     uint64
	parent uint64
}

// traceSpan is a completed span held by the recorder. A parent of 0 denotes a
// root span.
type traceSpan struct {
	label  string
	start  time.Duration
	end    time.Duration
	id     uint64
	parent uint64
}

// traceState is the environment-owned recorder.
type traceState struct {
	enabled bool
	nested  bool
	origin  time.Time
	nextID  atomic.Uint64
	mu      sync.Mutex
	spans   []traceSpan
}

// traceEventArgs carries the span's identity and its parentage. Both IDs are
// decimal strings so JSON readers that parse numbers as float64 cannot lose
// precision. Root spans emit no parent_id key.
type traceEventArgs struct {
	SpanID   string `json:"span_id"`
	ParentID string `json:"parent_id,omitempty"`
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

func newTraceState(traceOptions TraceOptions) traceState {
	if !traceOptions.Enabled {
		return traceState{}
	}
	return traceState{enabled: true, nested: traceOptions.Nested, origin: time.Now()}
}

func (s *traceState) startSpan(label string, parent uint64) TraceSpan {
	return TraceSpan{
		state:  s,
		label:  label,
		start:  time.Since(s.origin),
		id:     s.nextID.Add(1),
		parent: parent,
	}
}

func (s *traceState) endSpan(span TraceSpan) {
	// Read the clock before contending for the mutex so lock waits are not
	// charged to the measured invocation.
	end := time.Since(s.origin)
	completed := traceSpan{
		label:  span.label,
		start:  span.start,
		end:    end,
		id:     span.id,
		parent: span.parent,
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

	parents := make(map[uint64]uint64, len(spans))
	for _, span := range spans {
		parents[span.id] = span.parent
	}

	events := make([]traceEvent, 0, len(spans))
	var lanes []laneStack
	for _, span := range spans {
		events = append(events, newTraceEvent(span, assignTraceLane(&lanes, span, parents)))
	}
	return json.Marshal(traceDocument{TraceEvents: events})
}

func compareTraceSpans(a, b traceSpan) int {
	if c := cmp.Compare(a.start, b.start); c != 0 {
		return c
	}
	// Longest first, so a parent sorts before a child that starts with it and
	// the lane allocator sees the parent already open.
	if c := cmp.Compare(b.end, a.end); c != 0 {
		return c
	}
	return cmp.Compare(a.id, b.id)
}

// laneStack is one viewer lane's stack of the spans still open at the current
// position of the ordered scan, innermost last.
type laneStack struct {
	open []traceSpan
}

// closeEndedAt drops the spans that no longer contain a span starting at start.
func (l *laneStack) closeEndedAt(start time.Duration) {
	for len(l.open) > 0 && l.open[len(l.open)-1].end <= start {
		l.open = l.open[:len(l.open)-1]
	}
}

// accepts reports whether span can be drawn on this lane: either nothing is
// open on it, or the innermost open span is an ancestor that contains it. The
// ancestor test is what forbids false nesting.
func (l *laneStack) accepts(span traceSpan, parents map[uint64]uint64) bool {
	if len(l.open) == 0 {
		return true
	}
	top := l.open[len(l.open)-1]
	return span.end <= top.end && isTraceAncestor(top.id, span, parents)
}

func (l *laneStack) push(span traceSpan) {
	l.open = append(l.open, span)
}

// topID reports the innermost open span's id, or 0 when the lane is free.
func (l *laneStack) topID() uint64 {
	if len(l.open) == 0 {
		return 0
	}
	return l.open[len(l.open)-1].id
}

// isTraceAncestor walks span's recorded parentage looking for candidate. Ids
// are allocated by a single increment, so a parent id is always smaller than
// its child's and the walk always terminates.
func isTraceAncestor(candidate uint64, span traceSpan, parents map[uint64]uint64) bool {
	for id := span.parent; id != 0; id = parents[id] {
		if id == candidate {
			return true
		}
	}
	return false
}

// assignTraceLane places span on a lane that renders it correctly: its parent's
// lane when that lane still has the parent innermost, otherwise the lowest lane
// that accepts it, otherwise a fresh lane. Serial children therefore share
// their parent's lane and draw inside it, while concurrent siblings spread onto
// their own lanes.
func assignTraceLane(lanes *[]laneStack, span traceSpan, parents map[uint64]uint64) int {
	for lane := range *lanes {
		(*lanes)[lane].closeEndedAt(span.start)
	}
	for lane := range *lanes {
		if (*lanes)[lane].topID() == span.parent && (*lanes)[lane].accepts(span, parents) {
			(*lanes)[lane].push(span)
			return lane
		}
	}
	for lane := range *lanes {
		if (*lanes)[lane].accepts(span, parents) {
			(*lanes)[lane].push(span)
			return lane
		}
	}
	*lanes = append(*lanes, laneStack{open: []traceSpan{span}})
	return len(*lanes) - 1
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
		Args: traceEventArgs{
			SpanID:   strconv.FormatUint(span.id, 10),
			ParentID: traceSpanID(span.parent),
		},
	}
}

// traceSpanID renders a span id as a decimal string, and the root sentinel 0 as
// no id at all.
func traceSpanID(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
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
	return c.env.traceState.startSpan(spanLabel(operation, identity), 0)
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
	return c.env.traceState.startSpan(spanLabel(operation, packageIdentity(pkgID)), 0)
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

// StartChild starts a span nested under s, labeled with the operation and a
// free-form identity such as a definition name. It returns the disabled handle
// when s is disabled or the recording is not nested, so call sites need no
// enablement check and an unnested recording allocates nothing per definition.
func (s TraceSpan) StartChild(operation, identity string) TraceSpan {
	if s.state == nil || !s.state.nested {
		return TraceSpan{}
	}
	return s.state.startSpan(spanLabel(operation, identity), s.id)
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
