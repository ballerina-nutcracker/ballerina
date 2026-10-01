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

package symbols

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common/constants"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

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

// interactionPeers returns the peers a send or receive exchanges a message
// with; flushes and waits exchange none.
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

// workerMessageGroup holds the message queues of the workers of one block
// function body: its default worker and its named workers.
type workerMessageGroup struct {
	// order is the default worker followed by the named workers in
	// declaration order.
	order         []model.SymbolRef
	states        workerStateMap
	defaultWorker model.SymbolRef
	namedWorkers  map[string]model.SymbolRef
	sendMessages  map[model.SymbolRef]*[]model.WorkerMessageRef
	// invalid is set when a message action of the group failed peer or
	// placement validation, so pairing doesn't report on its counterpart.
	invalid bool
	// withModuleLevelNodes is set when the body is type resolved with
	// module-level nodes, where its workers are resolved one after another.
	withModuleLevelNodes bool
}

func newWorkerMessageGroup(body *ast.BLangBlockFunctionBody, withModuleLevelNodes bool) *workerMessageGroup {
	group := &workerMessageGroup{
		states:               make(workerStateMap),
		defaultWorker:        body.DefaultWorker,
		namedWorkers:         make(map[string]model.SymbolRef),
		sendMessages:         make(map[model.SymbolRef]*[]model.WorkerMessageRef),
		withModuleLevelNodes: withModuleLevelNodes,
	}
	group.addWorker(body.DefaultWorker, &body.DefaultWorkerSendMessages)
	for _, worker := range body.Workers {
		if worker.Symbol().IsEmpty() {
			// The declaration was already rejected.
			group.invalid = true
			continue
		}
		group.namedWorkers[worker.Name] = worker.Symbol()
		group.addWorker(worker.Symbol(), &worker.SendMessages)
	}
	return group
}

func (g *workerMessageGroup) addWorker(worker model.SymbolRef, sendMessages *[]model.WorkerMessageRef) {
	g.order = append(g.order, worker)
	g.states[worker] = newWorkerMessageState()
	g.sendMessages[worker] = sendMessages
}

// append adds e to worker's queue. A worker whose declaration was rejected
// has no queue; its group is already invalid.
func (g *workerMessageGroup) append(worker model.SymbolRef, e messageQueueElement) {
	if state, ok := g.states[worker]; ok {
		state.queue = append(state.queue, e)
	}
}

func (g *workerMessageGroup) addSend(worker model.SymbolRef, message model.WorkerMessageRef) {
	if sendMessages, ok := g.sendMessages[worker]; ok {
		*sendMessages = append(*sendMessages, message)
	}
}

func (g *workerMessageGroup) othersOf(worker model.SymbolRef) []model.SymbolRef {
	others := make([]model.SymbolRef, 0, len(g.order)-1)
	for _, peer := range g.order {
		if peer != worker {
			others = append(others, peer)
		}
	}
	return others
}

func invalidate(group *workerMessageGroup) {
	if group != nil {
		group.invalid = true
	}
}

// isMessageOwnerResolver reports whether bs resolves the statements of a
// worker: a function body (its default worker) or a named worker body.
func isMessageOwnerResolver(bs *blockSymbolResolver) bool {
	switch bs.node.(type) {
	case *ast.BLangFunction, *ast.BLangResourceMethod, *ast.BLangNamedWorkerDeclaration:
		return true
	default:
		return false
	}
}

// messageOwner returns the resolver of the worker whose statements contain a
// message action. A message action must be one of those statements, not
// inside a nested block or query.
func messageOwner(resolver symbolResolver, pos diagnostics.Location) (*blockSymbolResolver, bool) {
	bs := nearestBlockResolver(resolver)
	if bs != nil && isMessageOwnerResolver(bs) {
		if bs.messageGroup != nil && bs.messageGroup.withModuleLevelNodes {
			resolver.GetCtx().Unimplemented("worker message send/receive is not supported outside a module-level function or method", pos)
			invalidate(bs.messageGroup)
			return nil, false
		}
		return bs, true
	}
	resolver.GetCtx().Unimplemented("worker message send/receive is not supported here", pos)
	// The action's peer isn't resolved, so every group it could belong to
	// skips pairing.
	for current := bs; current != nil; current = nearestBlockResolver(current.parent) {
		invalidate(current.messageGroup)
	}
	return nil, false
}

// resolveMessagePeer validates the placement of a message action and its
// peer, and sets the peer's symbol. It returns the group of the action's
// worker and that worker.
func resolveMessagePeer(resolver symbolResolver, peer *ast.BLangWorkerPeer, pos diagnostics.Location) (*workerMessageGroup, model.SymbolRef, bool) {
	owner, ok := messageOwner(resolver, pos)
	if !ok {
		return nil, model.SymbolRef{}, false
	}
	group := owner.messageGroup
	worker := owner.messageWorker
	ctx := resolver.GetCtx()
	if peer.Name == constants.DefaultWorkerName {
		if group == nil || worker == group.defaultWorker {
			ctx.SemanticError("worker can't send/receive to itself", peer.Pos)
			invalidate(group)
			return nil, model.SymbolRef{}, false
		}
		peer.Symbol = group.defaultWorker
		return group, worker, true
	}
	if group != nil {
		if symbol, ok := group.namedWorkers[peer.Name]; ok {
			if symbol == worker {
				ctx.SemanticError("worker can't send/receive to itself", peer.Pos)
				invalidate(group)
				return nil, model.SymbolRef{}, false
			}
			peer.Symbol = symbol
			return group, worker, true
		}
	}
	if lookup, found := resolver.GetSymbol(peer.Name); found && ctx.GetSymbol(lookup.ref).Kind() == model.SymbolKindWorker {
		ctx.Unimplemented("worker message send/receive is not supported here", peer.Pos)
		invalidate(enclosingGroupOf(owner, lookup.ref))
	} else {
		ctx.SemanticError("undefined worker '"+peer.Name+"'", peer.Pos)
	}
	invalidate(group)
	return nil, model.SymbolRef{}, false
}

// enclosingGroupOf returns the group of an enclosing body that worker belongs
// to, or nil.
func enclosingGroupOf(bs *blockSymbolResolver, worker model.SymbolRef) *workerMessageGroup {
	for current := bs; current != nil; current = nearestBlockResolver(current.parent) {
		if group := current.messageGroup; group != nil {
			if _, ok := group.states[worker]; ok {
				return group
			}
		}
	}
	return nil
}

func resolveAsyncSendAction[T symbolResolver](resolver T, n *ast.BLangWorkerAsyncSendAction) {
	ast.Walk(resolver, n.Expr)
	group, worker, ok := resolveMessagePeer(resolver, &n.Peer, n.GetPosition())
	if !ok {
		return
	}
	n.Message = resolver.GetCtx().NewWorkerMessage()
	group.append(worker, asyncSendElement{node: n})
	group.addSend(worker, n.Message)
}

func resolveSyncSendAction[T symbolResolver](resolver T, n *ast.BLangWorkerSyncSendAction) {
	ast.Walk(resolver, n.Expr)
	group, worker, ok := resolveMessagePeer(resolver, &n.Peer, n.GetPosition())
	if !ok {
		return
	}
	n.Message = resolver.GetCtx().NewWorkerMessage()
	group.append(worker, syncSendElement{node: n})
	group.addSend(worker, n.Message)
}

func resolveReceiveAction[T symbolResolver](resolver T, n *ast.BLangWorkerReceiveAction) {
	group, worker, ok := resolveMessagePeer(resolver, &n.Peer, n.GetPosition())
	if !ok {
		return
	}
	group.append(worker, singleRecvElement{node: n})
}

func resolveMultipleReceiveAction[T symbolResolver](resolver T, n *ast.BLangWorkerMultipleReceiveAction) {
	var group *workerMessageGroup
	var worker model.SymbolRef
	seen := make(map[model.SymbolRef]struct{}, len(n.Fields))
	fieldNames := make(map[string]struct{}, len(n.Fields))
	for i := range n.Fields {
		peer := &n.Fields[i].Peer
		if _, duplicate := fieldNames[n.Fields[i].FieldName]; duplicate {
			resolver.GetCtx().SemanticError("duplicate field '"+n.Fields[i].FieldName+"' in multiple receive", peer.Pos)
			invalidate(group)
			return
		}
		fieldNames[n.Fields[i].FieldName] = struct{}{}
		fieldGroup, fieldWorker, ok := resolveMessagePeer(resolver, peer, n.GetPosition())
		if !ok {
			return
		}
		group, worker = fieldGroup, fieldWorker
		if _, duplicate := seen[peer.Symbol]; duplicate {
			resolver.GetCtx().SemanticError("duplicate worker '"+peer.SourceName()+"' in multiple receive", peer.Pos)
			invalidate(group)
			return
		}
		seen[peer.Symbol] = struct{}{}
	}
	if group != nil {
		group.append(worker, multipleRecvElement{node: n})
	}
}

func resolveFlushAction[T symbolResolver](resolver T, n *ast.BLangWorkerFlushAction) {
	if n.Peer != nil {
		group, worker, ok := resolveMessagePeer(resolver, n.Peer, n.GetPosition())
		if !ok {
			return
		}
		group.append(worker, flushElement{node: n, peers: []model.SymbolRef{n.Peer.Symbol}})
		return
	}
	owner, ok := messageOwner(resolver, n.GetPosition())
	if !ok || owner.messageGroup == nil {
		return
	}
	group := owner.messageGroup
	group.append(owner.messageWorker, flushElement{node: n, peers: group.othersOf(owner.messageWorker)})
}

// resolveWaitAction resolves the operands of a wait. A wait in a worker's own
// statements whose operands are its group's workers joins the worker's queue:
// all of them for a multiple or single wait, any of them for an alternate
// wait.
func resolveWaitAction[T symbolResolver](resolver T, futures []ast.BLangExpression, pos diagnostics.Location, waitsForAll bool) {
	for _, future := range futures {
		ast.Walk(resolver, future)
	}
	bs := nearestBlockResolver(resolver)
	if bs == nil || !isMessageOwnerResolver(bs) || bs.messageGroup == nil {
		return
	}
	group := bs.messageGroup
	peers := make([]model.SymbolRef, 0, len(futures))
	for _, future := range futures {
		ref, ok := future.(*ast.BLangVarRef)
		if !ok || !ast.SymbolIsSet(ref) || (ref.PkgAlias != nil && ref.PkgAlias.GetValue() != "") {
			continue
		}
		if _, isGroupWorker := group.states[ref.Symbol()]; isGroupWorker {
			peers = append(peers, ref.Symbol())
		}
	}
	switch {
	case waitsForAll && len(peers) > 0:
		group.append(bs.messageWorker, waitAllElement{peers: peers, position: pos})
	case !waitsForAll && len(peers) == len(futures):
		group.append(bs.messageWorker, waitAnyElement{peers: peers, position: pos})
	}
}

// pairWorkerMessages matches every receive of a group with the send it gets,
// and reports the first message action that can't line up.
func pairWorkerMessages(resolver *blockSymbolResolver, group *workerMessageGroup) {
	if group.invalid {
		return
	}
	computeFlushCoverage(group)
	if pos, msg, ok := checkMessageCounts(group); ok {
		semanticError(resolver, msg, pos)
		return
	}
	if pos, ok := checkInteractionAfterWait(group); ok {
		semanticError(resolver, "worker interaction after wait action", pos)
		return
	}
	if pos, msg, ok := simulateWorkerMessages(resolver, group); ok {
		semanticError(resolver, msg, pos)
	}
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
func simulateWorkerMessages(resolver *blockSymbolResolver, group *workerMessageGroup) (diagnostics.Location, string, bool) {
	pending := group.order
	for len(pending) > 0 {
		advanced := false
		nextPending := make([]model.SymbolRef, 0, len(pending))
		for _, worker := range pending {
			state := group.states[worker]
			if step(resolver, group.states, worker, state) {
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

func step(resolver *blockSymbolResolver, states workerStateMap, worker model.SymbolRef, state *workerMessageState) bool {
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
	resolver.GetCtx().InternalError("invalid worker message queue element", next.pos())
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
