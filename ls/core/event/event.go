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

// Package event provides project and compilation lifecycle events for the
// language server core. Subscribers receive events inline on the publishing
// goroutine or asynchronously through per-subscriber tiered queues.
package event

import (
	"fmt"
	"sync"
)

// Logf reports subscriber panics and dropped events. Hooks may be called
// concurrently; New uses a no-op unless WithLogf supplies a hook.
type Logf func(msg string, args ...any)

func noopLogf(string, ...any) {}

// Kind discriminates event types on the bus. Subscribers filter by kind.
type Kind uint8

const (
	// ProjectRegistered reports a project load or reload attempt.
	ProjectRegistered Kind = iota
	// ProjectEvicted is published when a project is removed from the index,
	// either by LRU eviction or a kind transition.
	ProjectEvicted
	// ProjectKindTransitioned reports a manifest create/delete reload of a
	// previously indexed root.
	ProjectKindTransitioned
	// ProjectUpdated is published after a content republish.
	ProjectUpdated
	// CompilationSucceeded is published when a stable snapshot is stored.
	CompilationSucceeded
	// CompilationFailed is published when a compile panics.
	CompilationFailed
	// CompilationCancelled is published when a cycle is superseded before
	// compilation or stable publication, not interrupted mid-compile.
	CompilationCancelled
	// ResolutionDiagnosticsReady is published with each stable snapshot,
	// even when its resolution diagnostics are empty.
	ResolutionDiagnosticsReady
	// CompilationDiagnosticsReady is published after a cycle with a clean
	// resolution, carrying compilation diagnostics.
	CompilationDiagnosticsReady
)

// Event is the interface every bus event implements. SourceRoot is the
// core-internal source-root key, not a DocumentURI. Generation identifies a
// source-root revision; registration, eviction, and kind-transition events
// return zero.
type Event interface {
	Kind() Kind
	SourceRoot() string
	Generation() uint64
}

// Tier selects a subscriber's asynchronous delivery semantics.
type Tier uint8

const (
	// TierCritical drops and logs incoming events when its bounded channel is full.
	TierCritical Tier = iota
	// TierCoalesceable replaces pending events of the same kind and source root.
	TierCoalesceable
	// TierBestEffort drops the oldest queued event when its bounded ring is full.
	TierBestEffort
)

// CoalesceKey identifies a coalesce group within a TierCoalesceable subscriber.
type CoalesceKey struct {
	Kind        Kind
	SourceRoot  string
	DocumentURI string // not populated by the bus
}

const criticalCapacity = 256

const bestEffortCapacity = 64

// EvictionReason describes why a project was evicted.
type EvictionReason uint8

const (
	// EvictionLRU is the count-bounded background-preferred eviction.
	EvictionLRU EvictionReason = iota
	// EvictionKindTransition is invalidation during manifest rediscovery.
	EvictionKindTransition
)

// eventBase holds the source root and generation shared by every event.
type eventBase struct {
	root       string
	generation uint64
}

func (e eventBase) SourceRoot() string { return e.root }
func (e eventBase) Generation() uint64 { return e.generation }

// ProjectRegisteredEvent reports a project load or reload attempt.
type ProjectRegisteredEvent struct {
	eventBase
}

// NewProjectRegisteredEvent constructs a ProjectRegisteredEvent.
func NewProjectRegisteredEvent(sourceRoot string) ProjectRegisteredEvent {
	return ProjectRegisteredEvent{eventBase: eventBase{root: sourceRoot}}
}

func (e ProjectRegisteredEvent) Kind() Kind { return ProjectRegistered }

// ProjectEvictedEvent is published when a project leaves the index.
type ProjectEvictedEvent struct {
	eventBase
	reason EvictionReason
}

// NewProjectEvictedEvent constructs a ProjectEvictedEvent.
func NewProjectEvictedEvent(sourceRoot string, reason EvictionReason) ProjectEvictedEvent {
	return ProjectEvictedEvent{eventBase: eventBase{root: sourceRoot}, reason: reason}
}

// Reason returns why the project was evicted.
func (e ProjectEvictedEvent) Reason() EvictionReason { return e.reason }

func (e ProjectEvictedEvent) Kind() Kind { return ProjectEvicted }

// ProjectKindTransitionedEvent reports a manifest create/delete reload of a
// previously indexed root. The workspace publisher supplies the same root
// as both oldRoot and newRoot.
type ProjectKindTransitionedEvent struct {
	eventBase
	oldRoot string
}

// NewProjectKindTransitionedEvent constructs a ProjectKindTransitionedEvent.
func NewProjectKindTransitionedEvent(oldRoot, newRoot string) ProjectKindTransitionedEvent {
	return ProjectKindTransitionedEvent{eventBase: eventBase{root: newRoot}, oldRoot: oldRoot}
}

// OldRoot returns the source root before the transition.
func (e ProjectKindTransitionedEvent) OldRoot() string { return e.oldRoot }

// NewRoot returns the source root after the transition.
func (e ProjectKindTransitionedEvent) NewRoot() string { return e.root }

func (e ProjectKindTransitionedEvent) Kind() Kind { return ProjectKindTransitioned }

// ProjectUpdatedEvent carries the source root's new generation after a content
// republish.
type ProjectUpdatedEvent struct {
	eventBase
}

// NewProjectUpdatedEvent constructs a ProjectUpdatedEvent.
func NewProjectUpdatedEvent(sourceRoot string, generation uint64) ProjectUpdatedEvent {
	return ProjectUpdatedEvent{eventBase: eventBase{root: sourceRoot, generation: generation}}
}

func (e ProjectUpdatedEvent) Kind() Kind { return ProjectUpdated }

// CompilationSucceededEvent is published when a stable snapshot is stored.
type CompilationSucceededEvent struct {
	eventBase
	descriptor string
}

// NewCompilationSucceededEvent constructs a CompilationSucceededEvent.
func NewCompilationSucceededEvent(sourceRoot, descriptor string, generation uint64) CompilationSucceededEvent {
	return CompilationSucceededEvent{eventBase: eventBase{root: sourceRoot, generation: generation}, descriptor: descriptor}
}

// DescriptorName returns the published package's descriptor name.
func (e CompilationSucceededEvent) DescriptorName() string { return e.descriptor }

func (e CompilationSucceededEvent) Kind() Kind { return CompilationSucceeded }

// CompilationFailedEvent is published when a compile panics.
type CompilationFailedEvent struct {
	eventBase
}

// NewCompilationFailedEvent constructs a CompilationFailedEvent.
func NewCompilationFailedEvent(sourceRoot string, generation uint64) CompilationFailedEvent {
	return CompilationFailedEvent{eventBase: eventBase{root: sourceRoot, generation: generation}}
}

func (e CompilationFailedEvent) Kind() Kind { return CompilationFailed }

// CompilationCancelledEvent is published when a cycle is superseded before
// compilation or stable publication.
type CompilationCancelledEvent struct {
	eventBase
}

// NewCompilationCancelledEvent constructs a CompilationCancelledEvent.
func NewCompilationCancelledEvent(sourceRoot string, generation uint64) CompilationCancelledEvent {
	return CompilationCancelledEvent{eventBase: eventBase{root: sourceRoot, generation: generation}}
}

func (e CompilationCancelledEvent) Kind() Kind { return CompilationCancelled }

// ResolutionDiagnosticsReadyEvent is published with each stable snapshot,
// even when its resolution diagnostics are empty.
type ResolutionDiagnosticsReadyEvent struct {
	eventBase
	descriptor string
}

// NewResolutionDiagnosticsReadyEvent constructs a ResolutionDiagnosticsReadyEvent.
func NewResolutionDiagnosticsReadyEvent(sourceRoot, descriptor string, generation uint64) ResolutionDiagnosticsReadyEvent {
	return ResolutionDiagnosticsReadyEvent{eventBase: eventBase{root: sourceRoot, generation: generation}, descriptor: descriptor}
}

// DescriptorName returns the published package's descriptor name.
func (e ResolutionDiagnosticsReadyEvent) DescriptorName() string { return e.descriptor }

func (e ResolutionDiagnosticsReadyEvent) Kind() Kind { return ResolutionDiagnosticsReady }

// CompilationDiagnosticsReadyEvent is published after a cycle with a clean
// resolution, carrying compilation diagnostics.
type CompilationDiagnosticsReadyEvent struct {
	eventBase
	descriptor string
}

// NewCompilationDiagnosticsReadyEvent constructs a CompilationDiagnosticsReadyEvent.
func NewCompilationDiagnosticsReadyEvent(sourceRoot, descriptor string, generation uint64) CompilationDiagnosticsReadyEvent {
	return CompilationDiagnosticsReadyEvent{eventBase: eventBase{root: sourceRoot, generation: generation}, descriptor: descriptor}
}

// DescriptorName returns the published package's descriptor name.
func (e CompilationDiagnosticsReadyEvent) DescriptorName() string { return e.descriptor }

func (e CompilationDiagnosticsReadyEvent) Kind() Kind { return CompilationDiagnosticsReady }

// Bus dispatches events inline on the publishing goroutine or asynchronously
// on per-subscriber drainer goroutines. Handler panics are recovered and logged
// without stopping subsequent deliveries.
type Bus struct {
	mu          sync.Mutex
	closed      bool
	subscribers []busSubscriber
	logf        Logf
}

// Option configures a Bus at construction time.
type Option func(*Bus)

// WithLogf sets the panic/drop logging hook. A nil hook leaves the default unchanged.
func WithLogf(logf Logf) Option {
	return func(b *Bus) {
		if logf != nil {
			b.logf = logf
		}
	}
}

type busSubscriber struct {
	kinds   []Kind
	handler func(Event)
	tier    *tierDelivery // nil for inline subscribers
}

type tierDelivery struct {
	tier     Tier
	logf     Logf
	events   chan Event            // CRITICAL only
	signal   chan struct{}         // COALESCEABLE/BEST_EFFORT: wake signal
	pending  map[CoalesceKey]Event // COALESCEABLE only (under coaMu)
	ring     []Event               // BEST_EFFORT only (under ringMu)
	ringMu   sync.Mutex
	ringHead int // BEST_EFFORT: oldest event's slot
	ringLen  int // BEST_EFFORT: current count
	coaMu    sync.Mutex
	flushReq chan chan struct{} // flush sends a chan; drainer closes it when idle
	done     chan struct{}      // closed when the drainer exits
}

// New creates an empty event bus. By default panic/drop conditions are
// reported through a no-op Logf; pass WithLogf to observe them.
func New(opts ...Option) *Bus {
	b := &Bus{logf: noopLogf}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Subscribe registers an inline handler; empty kinds matches all events.
// Handlers run in registration order on the publishing goroutine and may run
// concurrently when Publish calls overlap. Subscribe after Close is a no-op.
func (b *Bus) Subscribe(kinds []Kind, handler func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.subscribers = append(b.subscribers, busSubscriber{kinds: kinds, handler: handler})
}

// SubscribeWithTier registers a queued handler; empty kinds matches all events.
// Each registration has a dedicated drainer goroutine using the given tier.
// SubscribeWithTier after Close is a no-op.
func (b *Bus) SubscribeWithTier(kinds []Kind, tier Tier, handler func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	d := newTierDelivery(tier, b.logf)
	b.subscribers = append(b.subscribers, busSubscriber{kinds: kinds, handler: handler, tier: d})
	go d.run(handler)
}

// Publish dispatches e to every matching subscriber in registration order:
// inline handlers run on the caller's goroutine; tiered events are queued for
// asynchronous delivery. Publish after Close is a no-op.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	subs := make([]busSubscriber, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.Unlock()

	for _, sub := range subs {
		if !matches(sub.kinds, e.Kind()) {
			continue
		}
		if sub.tier == nil {
			b.deliver(sub.handler, e)
		} else {
			sub.tier.enqueue(e)
		}
	}
}

// Flush waits for tiered subscribers to drain their queues and finish handler
// calls. Publishers must be quiescent to guarantee all subscribers are idle
// when it returns. Do not call Flush from a tiered handler.
// Inline handlers finish before their Publish call returns.
func (b *Bus) Flush() {
	b.mu.Lock()
	subs := make([]busSubscriber, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.Unlock()
	for _, sub := range subs {
		if sub.tier == nil {
			continue
		}
		sub.tier.flush()
	}
}

// Close stops accepting subscribers and publishes, then waits for tiered drainers
// to exit. Critical queues are drained; other tiers may discard pending events.
// Close must not overlap Publish or Flush or run in a tiered handler.
// Repeated calls are no-ops.
func (b *Bus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	subs := b.subscribers
	b.subscribers = nil
	b.mu.Unlock()
	for _, sub := range subs {
		if sub.tier != nil {
			sub.tier.close()
		}
	}
}

func matches(kinds []Kind, k Kind) bool {
	if len(kinds) == 0 {
		return true
	}
	for _, want := range kinds {
		if want == k {
			return true
		}
	}
	return false
}

func (b *Bus) deliver(handler func(Event), e Event) {
	defer func() {
		if r := recover(); r != nil {
			b.logf("event: subscriber panic", "event_type", fmt.Sprintf("%T", e), "recovered", r)
		}
	}()
	handler(e)
}

func newTierDelivery(tier Tier, logf Logf) *tierDelivery {
	d := &tierDelivery{tier: tier, logf: logf, done: make(chan struct{}), flushReq: make(chan chan struct{}, 1)}
	switch tier {
	case TierCritical:
		d.events = make(chan Event, criticalCapacity)
	case TierCoalesceable:
		d.pending = make(map[CoalesceKey]Event)
		d.signal = make(chan struct{}, 1)
	case TierBestEffort:
		d.ring = make([]Event, bestEffortCapacity)
		d.signal = make(chan struct{}, 1)
	}
	return d
}

func (d *tierDelivery) enqueue(e Event) {
	switch d.tier {
	case TierCritical:
		select {
		case d.events <- e:
		default:
			d.logf("event: CRITICAL drop", "event_type", fmt.Sprintf("%T", e), "reason", "channel full")
		}
	case TierCoalesceable:
		key := CoalesceKey{Kind: e.Kind(), SourceRoot: e.SourceRoot()}
		d.coaMu.Lock()
		d.pending[key] = e
		d.coaMu.Unlock()
		select {
		case d.signal <- struct{}{}:
		default:
		}
	case TierBestEffort:
		d.ringMu.Lock()
		if d.ringLen == bestEffortCapacity {
			d.ring[d.ringHead] = e
			d.ringHead = (d.ringHead + 1) % bestEffortCapacity
		} else {
			slot := (d.ringHead + d.ringLen) % bestEffortCapacity
			d.ring[slot] = e
			d.ringLen++
		}
		d.ringMu.Unlock()
		select {
		case d.signal <- struct{}{}:
		default:
		}
	}
}

func (d *tierDelivery) run(handler func(Event)) {
	defer close(d.done)
	for {
		if d.tier == TierCritical {
			select {
			case e, ok := <-d.events:
				if !ok {
					return
				}
				d.dispatch(handler, e)
			case req := <-d.flushReq:
				d.drainCritical(handler)
				close(req)
			}
			continue
		}
		select {
		case _, ok := <-d.signal:
			if !ok {
				return
			}
			d.drainSignal(handler)
		case req := <-d.flushReq:
			d.drainSignal(handler)
			close(req)
		}
	}
}

func (d *tierDelivery) drainCritical(handler func(Event)) {
	for {
		select {
		case e := <-d.events:
			d.dispatch(handler, e)
		default:
			return
		}
	}
}

func (d *tierDelivery) drainSignal(handler func(Event)) {
	for {
		var e Event
		var ok bool
		if d.tier == TierCoalesceable {
			e, ok = d.takeCoalesced()
		} else {
			e, ok = d.takeRing()
		}
		if !ok {
			return
		}
		d.dispatch(handler, e)
	}
}

func (d *tierDelivery) dispatch(handler func(Event), e Event) {
	defer func() {
		if r := recover(); r != nil {
			d.logf("event: tiered subscriber panic", "event_type", fmt.Sprintf("%T", e), "recovered", r)
		}
	}()
	handler(e)
}

func (d *tierDelivery) takeCoalesced() (Event, bool) {
	d.coaMu.Lock()
	defer d.coaMu.Unlock()
	for k, e := range d.pending {
		delete(d.pending, k)
		return e, true
	}
	return nil, false
}

func (d *tierDelivery) takeRing() (Event, bool) {
	d.ringMu.Lock()
	defer d.ringMu.Unlock()
	if d.ringLen == 0 {
		return nil, false
	}
	e := d.ring[d.ringHead]
	d.ring[d.ringHead] = nil
	d.ringHead = (d.ringHead + 1) % bestEffortCapacity
	d.ringLen--
	return e, true
}

// flush requests a drain, waits for its handlers to finish, and retries if
// more events are queued. Concurrent publishers can enqueue again after the
// idle check.
func (d *tierDelivery) flush() {
	for {
		req := make(chan struct{})
		d.flushReq <- req
		<-req
		if d.idle() {
			return
		}
	}
}

func (d *tierDelivery) idle() bool {
	if d.tier == TierCritical {
		return len(d.events) == 0
	}
	d.coaMu.Lock()
	coa := len(d.pending)
	d.coaMu.Unlock()
	if coa != 0 {
		return false
	}
	d.ringMu.Lock()
	defer d.ringMu.Unlock()
	return d.ringLen == 0
}

func (d *tierDelivery) close() {
	if d.tier == TierCritical {
		close(d.events)
	} else {
		close(d.signal)
	}
	<-d.done
}
