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

package langinternalruntime

import (
	"fmt"
	"sync/atomic"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// workerMessage is one message sent between workers: the value, and two
// one-shot latches, set once the sender stored the value and received once
// the receiver took it. The value is written before set is released, so a
// strand that sees set sees the value.
type workerMessage struct {
	value    values.BalValue
	set      atomic.Bool
	received atomic.Bool
}

func workerMessageFromHandle(arg values.BalValue) (*workerMessage, error) {
	m, ok := arg.(*workerMessage)
	if !ok {
		return nil, fmt.Errorf("handle is not a worker message")
	}
	return m, nil
}

func workerMessagesFromList(arg values.BalValue) ([]*workerMessage, error) {
	list := arg.(*values.List)
	messages := make([]*workerMessage, list.Len())
	for i := range messages {
		m, err := workerMessageFromHandle(list.Get(i))
		if err != nil {
			return nil, err
		}
		messages[i] = m
	}
	return messages, nil
}

func futuresFromList(arg values.BalValue) []*values.Future {
	list := arg.(*values.List)
	futures := make([]*values.Future, list.Len())
	for i := range futures {
		futures[i] = list.Get(i).(*values.Future)
	}
	return futures
}

// peerOutcome returns how a completed peer terminated: its result, or the
// panic it terminated with. GetClaimed doesn't claim the future, so a later
// wait on the peer still works; it re-panics with the peer's stored panic,
// which keeps the peer's stack, and that panic is returned here instead of
// raised so the caller can choose which peer's outcome to report.
func peerOutcome(peer *values.Future) (result values.BalValue, panicValue any) {
	defer func() {
		panicValue = recover()
	}()
	return peer.GetClaimed(), nil
}

// peerFailure turns the outcome of a peer that completed without doing its
// part of a message into what the message action reports.
func peerFailure(result values.BalValue, panicValue any) *values.Error {
	if panicValue != nil {
		panic(panicValue)
	}
	if err, ok := result.(*values.Error); ok {
		return err
	}
	// Compile time pairing rules out a peer that terminates with success
	// without its matching action.
	panic(values.NewErrorWithMessage("internal error: worker terminated without its message action"))
}

func createWorkerMessage(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return &workerMessage{}, nil
}

func setWorkerMessageValue(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	m, err := workerMessageFromHandle(args[0])
	if err != nil {
		return nil, err
	}
	m.value = values.Clone(args[1])
	m.set.Store(true)
	return nil, nil
}

// getWorkerMessageValue returns the value of m once its sender set it, or how
// the sender terminated if it did so first.
func getWorkerMessageValue(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	m, err := workerMessageFromHandle(args[0])
	if err != nil {
		return nil, err
	}
	sender := args[1].(*values.Future)
	for {
		if m.set.Load() {
			m.received.Store(true)
			return m.value, nil
		}
		if sender.IsComplete() && !m.set.Load() {
			m.received.Store(true)
			return peerFailure(peerOutcome(sender)), nil
		}
		<-ctx.Yield()
	}
}

// getWorkerMessageValues returns the values of every message once all their
// senders set them, or how the first sender found terminated before setting
// its message did. Either way no message is left waiting to be received.
func getWorkerMessageValues(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	messages, err := workerMessagesFromList(args[0])
	if err != nil {
		return nil, err
	}
	senders := futuresFromList(args[1])
	for {
		allSet := true
		for i, m := range messages {
			if m.set.Load() {
				continue
			}
			allSet = false
			if senders[i].IsComplete() && !m.set.Load() {
				markReceived(messages)
				return peerFailure(peerOutcome(senders[i])), nil
			}
		}
		if allSet {
			markReceived(messages)
			result := newQueryList(ctx)
			for _, m := range messages {
				result.Append(ctx.TypeCtx(), m.value)
			}
			return result, nil
		}
		<-ctx.Yield()
	}
}

func markReceived(messages []*workerMessage) {
	for _, m := range messages {
		m.received.Store(true)
	}
}

// waitWorkerMessageReceived returns once the receiver took m, or how the
// receiver terminated if it did so first.
func waitWorkerMessageReceived(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	m, err := workerMessageFromHandle(args[0])
	if err != nil {
		return nil, err
	}
	receiver := args[1].(*values.Future)
	for {
		if m.received.Load() {
			return nil, nil
		}
		if receiver.IsComplete() && !m.received.Load() {
			return peerFailure(peerOutcome(receiver)), nil
		}
		<-ctx.Yield()
	}
}

// unreceivedOutcome is how a receiver terminated while one of its messages
// was still unreceived.
type unreceivedOutcome struct {
	result     values.BalValue
	panicValue any
}

// awaitReceivers waits until every receiver either took all its messages or
// terminated, and returns the outcome of each receiver that left a message
// unreceived, in receiver order. A message that was never set is skipped:
// its sender failed before sending it.
func awaitReceivers(ctx *extern.Context, messages [][]*workerMessage, receivers []*values.Future) []unreceivedOutcome {
	for {
		var outcomes []unreceivedOutcome
		pending := false
		for i, receiver := range receivers {
			if allReceived(messages[i]) {
				continue
			}
			if !receiver.IsComplete() || allReceived(messages[i]) {
				pending = true
				continue
			}
			result, panicValue := peerOutcome(receiver)
			outcomes = append(outcomes, unreceivedOutcome{result: result, panicValue: panicValue})
		}
		if !pending {
			return outcomes
		}
		<-ctx.Yield()
	}
}

func allReceived(messages []*workerMessage) bool {
	for _, m := range messages {
		if m.set.Load() && !m.received.Load() {
			return false
		}
	}
	return true
}

// flushWorkerMessages returns once every receiver took all its messages. Of
// the receivers that terminated leaving one unreceived, a panic is re-raised
// first, else the first error is returned.
func flushWorkerMessages(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	groups := args[0].(*values.List)
	messages := make([][]*workerMessage, groups.Len())
	for i := range messages {
		group, err := workerMessagesFromList(groups.Get(i))
		if err != nil {
			return nil, err
		}
		messages[i] = group
	}
	outcomes := awaitReceivers(ctx, messages, futuresFromList(args[1]))
	for _, outcome := range outcomes {
		if outcome.panicValue != nil {
			panic(outcome.panicValue)
		}
	}
	if len(outcomes) > 0 {
		return peerFailure(outcomes[0].result, nil), nil
	}
	return nil, nil
}

// awaitWorkerMessageDelivery holds a terminating worker until every async
// message it sent is received. A receiver that terminated first leaves a
// message undelivered, and the worker panics with the receiver's termination
// value; like flush, a receiver's panic is preferred over another's error.
func awaitWorkerMessageDelivery(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	sent, err := workerMessagesFromList(args[0])
	if err != nil {
		return nil, err
	}
	messages := make([][]*workerMessage, len(sent))
	for i, m := range sent {
		messages[i] = []*workerMessage{m}
	}
	outcomes := awaitReceivers(ctx, messages, futuresFromList(args[1]))
	for _, outcome := range outcomes {
		if outcome.panicValue != nil {
			panic(outcome.panicValue)
		}
	}
	if len(outcomes) > 0 {
		panic(peerFailure(outcomes[0].result, nil))
	}
	return nil, nil
}
