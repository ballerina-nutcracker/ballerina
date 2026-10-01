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

package workermessagesimulation

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// Worker supplies a worker's resolved message and wait actions in source order.
type Worker struct {
	Symbol  model.SymbolRef
	Actions []ast.BLangAction
}

// messageQueueElement is one entry in a worker's queue: a message action, or
// an unconditional wait on workers of its group.
//
//sumtype:decl
type messageQueueElement interface {
	isMessageQueueElement()
	pos() diagnostics.Location
}

type (
	asyncSendElement struct {
		node *ast.BLangWorkerAsyncSendAction
	}

	syncSendElement struct {
		node *ast.BLangWorkerSyncSendAction
	}

	singleRecvElement struct {
		node *ast.BLangWorkerReceiveAction
	}

	multipleRecvElement struct {
		node *ast.BLangWorkerMultipleReceiveAction
	}

	flushElement struct {
		node  *ast.BLangWorkerFlushAction
		peers []model.SymbolRef
	}

	waitAllElement struct {
		peers    []model.SymbolRef
		position diagnostics.Location
	}

	waitAnyElement struct {
		peers    []model.SymbolRef
		position diagnostics.Location
	}
)

func (asyncSendElement) isMessageQueueElement()    {}
func (syncSendElement) isMessageQueueElement()     {}
func (singleRecvElement) isMessageQueueElement()   {}
func (multipleRecvElement) isMessageQueueElement() {}
func (flushElement) isMessageQueueElement()        {}
func (waitAllElement) isMessageQueueElement()      {}
func (waitAnyElement) isMessageQueueElement()      {}

func (e asyncSendElement) pos() diagnostics.Location    { return e.node.GetPosition() }
func (e syncSendElement) pos() diagnostics.Location     { return e.node.GetPosition() }
func (e singleRecvElement) pos() diagnostics.Location   { return e.node.GetPosition() }
func (e multipleRecvElement) pos() diagnostics.Location { return e.node.GetPosition() }
func (e flushElement) pos() diagnostics.Location        { return e.node.GetPosition() }
func (e waitAllElement) pos() diagnostics.Location      { return e.position }
func (e waitAnyElement) pos() diagnostics.Location      { return e.position }

func interactionPeers(e messageQueueElement) []model.SymbolRef {
	switch e := e.(type) {
	case asyncSendElement:
		return []model.SymbolRef{e.node.Peer.Symbol}
	case syncSendElement:
		return []model.SymbolRef{e.node.Peer.Symbol}
	case singleRecvElement:
		return []model.SymbolRef{e.node.Peer.Symbol}
	case multipleRecvElement:
		peers := make([]model.SymbolRef, len(e.node.Fields))
		for i, field := range e.node.Fields {
			peers[i] = field.Peer.Symbol
		}
		return peers
	case flushElement, waitAllElement, waitAnyElement:
		return nil
	}
	return nil
}

type workerMessageState struct {
	index          int
	queue          []messageQueueElement
	asyncSentQueue map[model.SymbolRef][]asyncSendElement
	terminated     bool
}

func newWorkerMessageState() *workerMessageState {
	return &workerMessageState{asyncSentQueue: make(map[model.SymbolRef][]asyncSendElement)}
}

func (s *workerMessageState) next() (messageQueueElement, bool) {
	if s.done() {
		return nil, false
	}
	return s.queue[s.index], true
}

func (s *workerMessageState) proceed() {
	s.index++
}

// done reports whether the worker has no action left to run.
func (s *workerMessageState) done() bool {
	return s.index >= len(s.queue)
}

func (s *workerMessageState) allAsyncQueuesEmpty() bool {
	for _, queue := range s.asyncSentQueue {
		if len(queue) > 0 {
			return false
		}
	}
	return true
}

type workerStateMap map[model.SymbolRef]*workerMessageState

type workerMessageGroup struct {
	order  []model.SymbolRef
	states workerStateMap
}

func newWorkerMessageGroup(workers []Worker) *workerMessageGroup {
	group := &workerMessageGroup{states: make(workerStateMap)}
	for _, worker := range workers {
		group.order = append(group.order, worker.Symbol)
		group.states[worker.Symbol] = newWorkerMessageState()
	}
	for _, worker := range workers {
		state := group.states[worker.Symbol]
		for _, action := range worker.Actions {
			var element messageQueueElement
			switch action := action.(type) {
			case *ast.BLangWorkerAsyncSendAction:
				element = asyncSendElement{node: action}
			case *ast.BLangWorkerSyncSendAction:
				element = syncSendElement{node: action}
			case *ast.BLangWorkerReceiveAction:
				element = singleRecvElement{node: action}
			case *ast.BLangWorkerMultipleReceiveAction:
				element = multipleRecvElement{node: action}
			case *ast.BLangWorkerFlushAction:
				var peers []model.SymbolRef
				if action.Peer != nil {
					peers = []model.SymbolRef{action.Peer.Symbol}
				} else {
					for _, peer := range group.order {
						if peer != worker.Symbol {
							peers = append(peers, peer)
						}
					}
				}
				element = flushElement{node: action, peers: peers}
			case *ast.BLangSingleWaitAction, *ast.BLangAlternateWaitAction, *ast.BLangMultipleWaitAction:
				element, _ = groupWaitElement(group.order, action)
			}
			if element != nil {
				state.queue = append(state.queue, element)
			}
		}
	}
	return group
}

func groupWaitElement(workers []model.SymbolRef, action ast.BLangAction) (messageQueueElement, bool) {
	var futures []ast.BLangExpression
	waitsForAll := true
	switch action := action.(type) {
	case *ast.BLangSingleWaitAction:
		futures = []ast.BLangExpression{action.FutureExpr}
	case *ast.BLangAlternateWaitAction:
		futures, waitsForAll = action.FutureExprs, false
	case *ast.BLangMultipleWaitAction:
		futures = action.FutureExprs
	default:
		return nil, false
	}
	peers := make([]model.SymbolRef, 0, len(futures))
	for _, future := range futures {
		for {
			paren, isParen := future.(*ast.BLangGroupExpr)
			if !isParen {
				break
			}
			future = paren.Expression
		}
		ref, ok := future.(*ast.BLangVarRef)
		if !ok || !ast.SymbolIsSet(ref) || (ref.PkgAlias != nil && ref.PkgAlias.GetValue() != "") {
			continue
		}
		for _, worker := range workers {
			if ref.Symbol() == worker {
				peers = append(peers, worker)
				break
			}
		}
	}
	switch {
	case waitsForAll && len(peers) > 0:
		return waitAllElement{peers: peers, position: action.GetPosition()}, true
	case !waitsForAll && len(peers) == len(futures):
		return waitAnyElement{peers: peers, position: action.GetPosition()}, true
	default:
		return nil, false
	}
}

// PairWorkerMessages validates and pairs the resolved actions of workers in
// declaration order, with the default worker first when one exists. Nested
// waits on these workers are unsupported when the group exchanges messages.
func PairWorkerMessages(ctx *context.CompilerContext, workers []Worker, nestedWaits []ast.BLangAction) bool {
	group := newWorkerMessageGroup(workers)
	if hasMessageActions(group) {
		invalid := false
		for _, wait := range nestedWaits {
			if _, ok := groupWaitElement(group.order, wait); ok {
				ctx.Unimplemented("wait on a worker is not supported here when the workers exchange messages", wait.GetPosition())
				invalid = true
			}
		}
		if invalid {
			return false
		}
	}
	computeFlushCoverage(group)
	if pos, msg, ok := checkMessageCounts(group); ok {
		ctx.SemanticError(msg, pos)
		return false
	}
	if pos, ok := checkInteractionAfterWait(group); ok {
		ctx.SemanticError("worker interaction after wait action", pos)
		return false
	}
	if pos, msg, ok := simulateWorkerMessages(ctx, group); ok {
		ctx.SemanticError(msg, pos)
		return false
	}
	return true
}

func hasMessageActions(group *workerMessageGroup) bool {
	for _, state := range group.states {
		for _, element := range state.queue {
			switch element.(type) {
			case asyncSendElement, syncSendElement, singleRecvElement, multipleRecvElement, flushElement:
				return true
			case waitAllElement, waitAnyElement:
			}
		}
	}
	return false
}

// computeFlushCoverage marks every async send followed by a sync send or flush
// to the same peer, and records on each flush the messages it covers.
func computeFlushCoverage(group *workerMessageGroup) {
	for _, worker := range group.order {
		pending := make(map[model.SymbolRef][]*ast.BLangWorkerAsyncSendAction)
		cover := func(peer model.SymbolRef) {
			for _, send := range pending[peer] {
				send.Covered = true
			}
			delete(pending, peer)
		}
		for _, e := range group.states[worker].queue {
			switch e := e.(type) {
			case asyncSendElement:
				peer := e.node.Peer.Symbol
				pending[peer] = append(pending[peer], e.node)
			case syncSendElement:
				cover(e.node.Peer.Symbol)
			case flushElement:
				for _, peer := range e.peers {
					if sends := pending[peer]; len(sends) > 0 {
						messages := make([]model.WorkerMessageRef, len(sends))
						for i, send := range sends {
							messages[i] = send.Message
						}
						e.node.Covered = append(e.node.Covered, ast.BLangWorkerFlushCoverage{Peer: peer, Messages: messages})
					}
					cover(peer)
				}
			case singleRecvElement, multipleRecvElement, waitAllElement, waitAnyElement:
			}
		}
	}
}

// checkMessageCounts compares, for every ordered pair of workers, the sends
// from one with the receives at the other, so a missing send or receive isn't
// reported as a deadlock.
func checkMessageCounts(group *workerMessageGroup) (diagnostics.Location, string, bool) {
	for _, sender := range group.order {
		for _, receiver := range group.order {
			if sender == receiver {
				continue
			}
			sends := sendsTo(group.states[sender], receiver)
			receives := receivesFrom(group.states[receiver], sender)
			if len(sends) > len(receives) {
				return sends[len(receives)], "no matching receive", true
			}
			if len(receives) > len(sends) {
				return receives[len(sends)], "no matching send", true
			}
		}
	}
	return diagnostics.Location{}, "", false
}

func sendsTo(state *workerMessageState, receiver model.SymbolRef) []diagnostics.Location {
	var positions []diagnostics.Location
	for _, e := range state.queue {
		switch e := e.(type) {
		case asyncSendElement:
			if e.node.Peer.Symbol == receiver {
				positions = append(positions, e.pos())
			}
		case syncSendElement:
			if e.node.Peer.Symbol == receiver {
				positions = append(positions, e.pos())
			}
		case singleRecvElement, multipleRecvElement, flushElement, waitAllElement, waitAnyElement:
		}
	}
	return positions
}

func receivesFrom(state *workerMessageState, sender model.SymbolRef) []diagnostics.Location {
	var positions []diagnostics.Location
	for _, e := range state.queue {
		switch e := e.(type) {
		case singleRecvElement:
			if e.node.Peer.Symbol == sender {
				positions = append(positions, e.pos())
			}
		case multipleRecvElement:
			for _, field := range e.node.Fields {
				if field.Peer.Symbol == sender {
					positions = append(positions, e.pos())
				}
			}
		case asyncSendElement, syncSendElement, flushElement, waitAllElement, waitAnyElement:
		}
	}
	return positions
}

// checkInteractionAfterWait reports a send or receive with a worker the same
// worker already waited for.
func checkInteractionAfterWait(group *workerMessageGroup) (diagnostics.Location, bool) {
	for _, worker := range group.order {
		waited := make(map[model.SymbolRef]struct{})
		for _, e := range group.states[worker].queue {
			if wait, ok := e.(waitAllElement); ok {
				for _, peer := range wait.peers {
					waited[peer] = struct{}{}
				}
				continue
			}
			for _, peer := range interactionPeers(e) {
				if _, ok := waited[peer]; ok {
					return e.pos(), true
				}
			}
		}
	}
	return diagnostics.Location{}, false
}

// simulateWorkerMessages runs every worker's queue until all of them
// terminate, pairing each receive with the message it gets. Each condition a
// step waits on stays true once it holds, so the firing order doesn't change
// the outcome. It reports where the workers got stuck.
func simulateWorkerMessages(ctx *context.CompilerContext, group *workerMessageGroup) (diagnostics.Location, string, bool) {
	pending := group.order
	for len(pending) > 0 {
		advanced := false
		nextPending := make([]model.SymbolRef, 0, len(pending))
		for _, worker := range pending {
			state := group.states[worker]
			if step(ctx, group.states, worker, state) {
				advanced = true
			}
			if !state.terminated {
				nextPending = append(nextPending, worker)
			}
		}
		pending = nextPending
		if !advanced && len(pending) > 0 {
			return reportStuck(group, pending)
		}
	}
	return diagnostics.Location{}, "", false
}

func step(ctx *context.CompilerContext, states workerStateMap, worker model.SymbolRef, state *workerMessageState) bool {
	next, ok := state.next()
	if !ok {
		if !state.allAsyncQueuesEmpty() {
			return false
		}
		state.terminated = true
		return true
	}
	switch next := next.(type) {
	case asyncSendElement:
		peer := next.node.Peer.Symbol
		state.asyncSentQueue[peer] = append(state.asyncSentQueue[peer], next)
		state.proceed()
		return true
	case syncSendElement:
		dest := next.node.Peer.Symbol
		if len(state.asyncSentQueue[dest]) > 0 {
			return false
		}
		destState := states[dest]
		destNext, ok := destState.next()
		if !ok {
			return false
		}
		receive, ok := destNext.(singleRecvElement)
		if !ok || receive.node.Peer.Symbol != worker {
			return false
		}
		receive.node.Message = next.node.Message
		state.proceed()
		destState.proceed()
		return true
	case singleRecvElement:
		source := states[next.node.Peer.Symbol]
		queue := source.asyncSentQueue[worker]
		if len(queue) == 0 {
			return false
		}
		next.node.Message = queue[0].node.Message
		source.asyncSentQueue[worker] = queue[1:]
		state.proceed()
		return true
	case multipleRecvElement:
		for _, field := range next.node.Fields {
			if !fieldAvailable(states[field.Peer.Symbol], worker) {
				return false
			}
		}
		for i := range next.node.Fields {
			field := &next.node.Fields[i]
			source := states[field.Peer.Symbol]
			if queue := source.asyncSentQueue[worker]; len(queue) > 0 {
				field.Message = queue[0].node.Message
				source.asyncSentQueue[worker] = queue[1:]
				continue
			}
			sourceNext, _ := source.next()
			field.Message = sourceNext.(syncSendElement).node.Message
			source.proceed()
		}
		state.proceed()
		return true
	case flushElement:
		for _, peer := range next.peers {
			if len(state.asyncSentQueue[peer]) > 0 {
				return false
			}
		}
		state.proceed()
		return true
	case waitAllElement:
		for _, peer := range next.peers {
			if !states[peer].terminated {
				return false
			}
		}
		state.proceed()
		return true
	case waitAnyElement:
		for _, peer := range next.peers {
			if states[peer].terminated {
				state.proceed()
				return true
			}
		}
		return false
	}
	ctx.InternalError("invalid worker message queue element", next.pos())
	return false
}

// fieldAvailable reports whether source has a message for worker that a
// multiple receive can take now.
func fieldAvailable(source *workerMessageState, worker model.SymbolRef) bool {
	if len(source.asyncSentQueue[worker]) > 0 {
		return true
	}
	sourceNext, ok := source.next()
	if !ok {
		return false
	}
	send, ok := sourceNext.(syncSendElement)
	return ok && send.node.Peer.Symbol == worker
}

func reportStuck(group *workerMessageGroup, pending []model.SymbolRef) (diagnostics.Location, string, bool) {
	for _, worker := range pending {
		if pos, msg, ok := unmatched(group, worker); ok {
			return pos, msg, true
		}
	}
	for _, worker := range pending {
		next, _ := group.states[worker].next()
		switch next.(type) {
		case asyncSendElement, syncSendElement, singleRecvElement, multipleRecvElement:
			return next.pos(), "worker message deadlock", true
		case flushElement, waitAllElement, waitAnyElement:
		}
	}
	return blockedPos(group.states[pending[0]], group.order), "worker message deadlock", true
}

// unmatched reports a blocked action of worker whose peer has nothing left
// that could unblock it.
func unmatched(group *workerMessageGroup, worker model.SymbolRef) (diagnostics.Location, string, bool) {
	states := group.states
	state := states[worker]
	next, ok := state.next()
	if !ok {
		for _, dest := range group.order {
			if queue := state.asyncSentQueue[dest]; len(queue) > 0 && states[dest].done() {
				return queue[0].pos(), "no matching receive", true
			}
		}
		return diagnostics.Location{}, "", false
	}
	switch next := next.(type) {
	case syncSendElement:
		if states[next.node.Peer.Symbol].done() {
			return next.pos(), "no matching receive", true
		}
	case singleRecvElement, multipleRecvElement:
		for _, peer := range interactionPeers(next) {
			if states[peer].done() && !fieldAvailable(states[peer], worker) {
				return next.pos(), "no matching send", true
			}
		}
	case flushElement:
		for _, peer := range next.peers {
			if queue := state.asyncSentQueue[peer]; len(queue) > 0 && states[peer].done() {
				return queue[0].pos(), "no matching receive", true
			}
		}
	case asyncSendElement, waitAllElement, waitAnyElement:
	}
	return diagnostics.Location{}, "", false
}

// blockedPos is where a stuck worker waits: its next action, or its first
// undelivered async send when it is waiting to terminate.
func blockedPos(state *workerMessageState, order []model.SymbolRef) diagnostics.Location {
	if next, ok := state.next(); ok {
		return next.pos()
	}
	for _, dest := range order {
		if queue := state.asyncSentQueue[dest]; len(queue) > 0 {
			return queue[0].pos()
		}
	}
	return diagnostics.Location{}
}
