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
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/workermessagesimulation"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// workerMessageGroup collects resolved actions of the workers of one block
// function body: its default worker and its named workers.
type workerMessageGroup struct {
	// order is the default worker followed by the named workers in
	// declaration order.
	order         []model.SymbolRef
	actions       map[model.SymbolRef][]ast.BLangAction
	defaultWorker model.SymbolRef
	namedWorkers  map[string]model.SymbolRef
	sendMessages  map[model.SymbolRef]*[]model.WorkerMessageRef
	// invalid is set when a message action of the group failed peer or
	// placement validation, so pairing doesn't report on its counterpart.
	invalid bool
	// nestedWaits occur inside nested blocks, where pairing cannot place
	// waits on the group's workers.
	nestedWaits []ast.BLangAction
	// withModuleLevelNodes is set when the body is type resolved with
	// module-level nodes, where its workers are resolved one after another.
	withModuleLevelNodes bool
}

func newWorkerMessageGroup(region *ast.WorkerRegion, withModuleLevelNodes bool) *workerMessageGroup {
	group := &workerMessageGroup{
		actions:              make(map[model.SymbolRef][]ast.BLangAction),
		namedWorkers:         make(map[string]model.SymbolRef),
		sendMessages:         make(map[model.SymbolRef]*[]model.WorkerMessageRef),
		withModuleLevelNodes: withModuleLevelNodes,
	}
	if defaultWorker := region.DefaultWorker; defaultWorker != nil {
		group.defaultWorker = defaultWorker.Symbol()
		group.addWorker(defaultWorker.Symbol(), &defaultWorker.SendMessages)
	}
	for _, worker := range region.Workers {
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

func (g *workerMessageGroup) hasDefaultWorker() bool {
	return !g.defaultWorker.IsEmpty()
}

func (g *workerMessageGroup) addWorker(worker model.SymbolRef, sendMessages *[]model.WorkerMessageRef) {
	g.order = append(g.order, worker)
	g.actions[worker] = nil
	g.sendMessages[worker] = sendMessages
}

// append adds action to worker's ordered actions. A worker whose declaration
// was rejected has no entry; its group is already invalid.
func (g *workerMessageGroup) append(worker model.SymbolRef, action ast.BLangAction) {
	if actions, ok := g.actions[worker]; ok {
		g.actions[worker] = append(actions, action)
	}
}

func (g *workerMessageGroup) addSend(worker model.SymbolRef, message model.WorkerMessageRef) {
	if sendMessages, ok := g.sendMessages[worker]; ok {
		*sendMessages = append(*sendMessages, message)
	}
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
	bs, _ := resolver.(*blockSymbolResolver)
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
	for current := bs; current != nil; current, _ = current.parent.(*blockSymbolResolver) {
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
	for current := bs; current != nil; current, _ = current.parent.(*blockSymbolResolver) {
		if group := current.messageGroup; group != nil {
			if _, ok := group.actions[worker]; ok {
				return group
			}
		}
	}
	return nil
}

func resolveAsyncSendAction(resolver symbolResolver, n *ast.BLangWorkerAsyncSendAction) bool {
	ok := resolveOptionalExpr(resolver, n.Expr)
	group, worker, peerOK := resolveMessagePeer(resolver, &n.Peer, n.GetPosition())
	if !peerOK {
		return false
	}
	n.Message = resolver.GetCtx().NewWorkerMessage()
	group.append(worker, n)
	group.addSend(worker, n.Message)
	return ok
}

func resolveSyncSendAction(resolver symbolResolver, n *ast.BLangWorkerSyncSendAction) bool {
	ok := resolveOptionalExpr(resolver, n.Expr)
	group, worker, peerOK := resolveMessagePeer(resolver, &n.Peer, n.GetPosition())
	if !peerOK {
		return false
	}
	n.Message = resolver.GetCtx().NewWorkerMessage()
	group.append(worker, n)
	group.addSend(worker, n.Message)
	return ok
}

func resolveReceiveAction(resolver symbolResolver, n *ast.BLangWorkerReceiveAction) bool {
	group, worker, ok := resolveMessagePeer(resolver, &n.Peer, n.GetPosition())
	if !ok {
		return false
	}
	group.append(worker, n)
	return true
}

func resolveMultipleReceiveAction(resolver symbolResolver, n *ast.BLangWorkerMultipleReceiveAction) bool {
	var group *workerMessageGroup
	var worker model.SymbolRef
	seen := make(map[model.SymbolRef]struct{}, len(n.Fields))
	fieldNames := make(map[string]struct{}, len(n.Fields))
	for i := range n.Fields {
		peer := &n.Fields[i].Peer
		if _, duplicate := fieldNames[n.Fields[i].FieldName]; duplicate {
			resolver.GetCtx().SemanticError("duplicate field '"+n.Fields[i].FieldName+"' in multiple receive", peer.Pos)
			invalidate(group)
			return false
		}
		fieldNames[n.Fields[i].FieldName] = struct{}{}
		fieldGroup, fieldWorker, ok := resolveMessagePeer(resolver, peer, n.GetPosition())
		if !ok {
			return false
		}
		group, worker = fieldGroup, fieldWorker
		if _, duplicate := seen[peer.Symbol]; duplicate {
			resolver.GetCtx().SemanticError("duplicate worker '"+peer.SourceName()+"' in multiple receive", peer.Pos)
			invalidate(group)
			return false
		}
		seen[peer.Symbol] = struct{}{}
	}
	if group != nil {
		group.append(worker, n)
	}
	return true
}

func resolveFlushAction(resolver symbolResolver, n *ast.BLangWorkerFlushAction) bool {
	if n.Peer != nil {
		group, worker, ok := resolveMessagePeer(resolver, n.Peer, n.GetPosition())
		if !ok {
			return false
		}
		group.append(worker, n)
		return true
	}
	owner, ok := messageOwner(resolver, n.GetPosition())
	if !ok {
		return false
	}
	if owner.messageGroup == nil {
		return true
	}
	owner.messageGroup.append(owner.messageWorker, n)
	return true
}

// resolveWaitAction resolves the operands of a wait. A wait on its group's
// workers joins the actions of the worker whose statements contain it. Such a
// wait inside a nested block of those statements is recorded on the group
// instead.
func resolveWaitAction(resolver symbolResolver, action ast.BLangAction, futures []ast.BLangExpression) bool {
	ok := resolveExprs(resolver, futures)
	bs, _ := resolver.(*blockSymbolResolver)
	owner := bs
	for owner != nil && !isMessageOwnerResolver(owner) {
		owner, _ = owner.parent.(*blockSymbolResolver)
	}
	if owner == nil || owner.messageGroup == nil {
		return ok
	}
	group := owner.messageGroup
	if owner == bs {
		group.append(owner.messageWorker, action)
	} else {
		group.nestedWaits = append(group.nestedWaits, action)
	}
	return ok
}

// pairWorkerMessages matches every receive of a group with the send it gets,
// and reports the first message action that can't line up.
func pairWorkerMessages(resolver *blockSymbolResolver, group *workerMessageGroup) bool {
	if group.invalid {
		return true
	}
	workers := make([]workermessagesimulation.Worker, len(group.order))
	for i, worker := range group.order {
		workers[i] = workermessagesimulation.Worker{Symbol: worker, Actions: group.actions[worker]}
	}
	return workermessagesimulation.PairWorkerMessages(resolver.GetCtx(), workers, group.nestedWaits)
}
