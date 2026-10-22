// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"fmt"
	"sync"

	"github.com/eclipse/paho.golang/paho"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	listenerStateField = "$listener"
	callerStateField   = "$caller"
)

type listenerEventKind int

const (
	eventMessage listenerEventKind = iota
	eventComplete
	eventError
)

type listenerEvent struct {
	kind listenerEventKind
	// service is set on completion events, which go to the service that handled the request
	// even if it has since been detached.
	service  *values.Object
	message  receivedMessage
	packetID uint16
	topic    string
	err      *mqttError
}

// listenerState serialises service invocations on one dispatcher goroutine, matching the
// single Paho Java callback thread jBallerina dispatches from.
type listenerState struct {
	conn       *connection
	manualAcks bool
	events     *blockingQueue[listenerEvent]

	mu      sync.Mutex
	service *values.Object
	started bool
	// subMu serialises the broker round-trips of start, attach and detach without holding mu,
	// which the incoming-message path takes.
	subMu         sync.Mutex
	subscriptions []paho.SubscribeOptions
	subscribed    bool
}

type callerState struct {
	listener *listenerState
	service  *values.Object
	message  receivedMessage
}

func listenerOf(self *values.Object) *listenerState {
	v, _ := self.Get(listenerStateField)
	ls, _ := v.(*listenerState)
	return ls
}

func (ls *listenerState) currentService() *values.Object {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.service
}

func (ls *listenerState) setService(svc *values.Object) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.service = svc
}

func (m *mqttModule) registerListener() {
	reg := func(name string, fn extern.NativeFunc) {
		runtime.RegisterExternFunction(m.rt, orgName, moduleName, name, fn)
	}
	reg("Listener.externInit", m.listenerInit)
	reg("Listener.attach", m.listenerAttach)
	reg("Listener.detach", m.listenerDetach)
	reg("Listener.externStart", m.listenerStart)
	reg("Listener.gracefulStop", m.listenerGracefulStop)
	reg("Listener.immediateStop", m.listenerImmediateStop)
	reg("Caller."+model.RemoteMethodName("complete"), m.callerComplete)
	reg("Caller."+model.RemoteMethodName("respond"), m.callerRespond)
}

func (m *mqttModule) listenerInit(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	serverURI, _ := args[1].(string)
	clientID, _ := args[2].(string)
	config, _ := args[3].(*values.Map)
	opts, mErr := parseConnectionOptions(m.rt.Platform(), serverURI, clientID, config)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	ls := &listenerState{manualAcks: opts.manualAcks, events: newBlockingQueue[listenerEvent]()}
	// Acknowledgements are always sent by the dispatcher (or Caller->complete), after onMessage
	// returns, as Paho Java does for messages handled in its callback.
	opts.manualAcks = true
	ls.conn = newConnection(m.rt.Platform(), opts, connectionHandlers{
		onMessage: func(msg receivedMessage) {
			if ls.currentService() == nil {
				ls.conn.ack(msg)
				return
			}
			ls.events.put(listenerEvent{kind: eventMessage, message: msg})
		},
		onConnectionLost: func(err *mqttError) {
			ls.events.put(listenerEvent{kind: eventError, err: err})
		},
	})
	if mErr := ls.conn.connect(); mErr != nil {
		return toBalError(ctx, mErr)
	}
	self.Put(listenerStateField, ls)
	return nil, nil
}

// listenerAttach re-subscribes the listener's topics when a service is attached again
// after detach.
func (m *mqttModule) listenerAttach(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	ls := listenerOf(args[0].(*values.Object))
	svc, _ := args[1].(*values.Object)
	ls.subMu.Lock()
	defer ls.subMu.Unlock()
	if ls.subscriptions != nil && !ls.subscribed {
		if mErr := ls.conn.subscribe(ls.subscriptions); mErr != nil {
			return toBalError(ctx, mErr)
		}
		ls.subscribed = true
	}
	ls.mu.Lock()
	ls.service = svc
	startDispatcher := !ls.started
	ls.started = true
	ls.mu.Unlock()
	if startDispatcher {
		go m.dispatch(ls)
	}
	return nil, nil
}

// listenerDetach unsubscribes the listener's topics and drops the service but keeps the
// broker connection open, as clarified in ballerina-library#9326.
func (m *mqttModule) listenerDetach(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	ls := listenerOf(args[0].(*values.Object))
	ls.subMu.Lock()
	defer ls.subMu.Unlock()
	if ls.subscribed {
		if mErr := ls.conn.unsubscribe(ls.subscriptions); mErr != nil {
			return toBalError(ctx, mErr)
		}
		ls.subscribed = false
	}
	ls.setService(nil)
	return nil, nil
}

func (m *mqttModule) listenerStart(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	ls := listenerOf(args[0].(*values.Object))
	subscriptions, _ := args[1].(*values.List)
	options, mErr := subscribeOptions(subscriptions)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	ls.subMu.Lock()
	defer ls.subMu.Unlock()
	if mErr := ls.conn.subscribe(options); mErr != nil {
		return toBalError(ctx, mErr)
	}
	ls.subscriptions = options
	ls.subscribed = true
	return nil, nil
}

func (m *mqttModule) listenerGracefulStop(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	ls := listenerOf(args[0].(*values.Object))
	if mErr := ls.conn.disconnect(); mErr != nil {
		return toBalError(ctx, mErr)
	}
	if mErr := ls.conn.close(); mErr != nil {
		return toBalError(ctx, mErr)
	}
	ls.stop()
	return nil, nil
}

func (m *mqttModule) listenerImmediateStop(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	ls := listenerOf(args[0].(*values.Object))
	ls.conn.disconnectForcibly()
	if mErr := ls.conn.close(); mErr != nil {
		return toBalError(ctx, mErr)
	}
	ls.stop()
	return nil, nil
}

func (ls *listenerState) stop() {
	ls.setService(nil)
	ls.events.close()
}

func (m *mqttModule) dispatch(ls *listenerState) {
	for {
		event, ok := ls.events.take()
		if !ok {
			return
		}
		svc := ls.currentService()
		if event.kind == eventComplete {
			svc = event.service
		}
		if svc == nil {
			if event.kind == eventMessage {
				ls.conn.ack(event.message)
			}
			continue
		}
		m.handleEvent(ls, svc, event)
	}
}

func (m *mqttModule) handleEvent(ls *listenerState, svc *values.Object, event listenerEvent) {
	ctx := m.rt.AcquirePooledContext()
	defer m.rt.ReleasePooledContext(ctx)
	switch event.kind {
	case eventMessage:
		m.invokeOnMessage(ctx, ls, svc, event.message)
		if !ls.manualAcks {
			ls.conn.ack(event.message)
		}
	case eventComplete:
		handle, ok := ctx.LookupRemoteMethod(svc, "onComplete")
		if !ok {
			return
		}
		token, err := deliveryToken(ctx, event.packetID, event.topic)
		if err != nil {
			m.printError(nil, err)
			return
		}
		m.printPanic(m.invokeTrapped(ctx, handle, []values.BalValue{svc, token}))
	case eventError:
		m.invokeOnError(ctx, svc, event.err)
	}
}

func (m *mqttModule) invokeOnMessage(ctx *extern.Context, ls *listenerState, svc *values.Object, msg receivedMessage) {
	handle, ok := ctx.LookupRemoteMethod(svc, "onMessage")
	if !ok {
		m.invokeOnError(ctx, svc, newError("method onMessage not found"))
		return
	}
	message, err := m.toBalMessage(ctx, msg)
	if err != nil {
		m.printError(nil, err)
		return
	}
	args := []values.BalValue{svc, message}
	if sig, ok := ctx.MethodSignature(handle); ok && len(sig.Params) == 2 {
		caller, err := callHelper(ctx, "createCaller")
		if err != nil {
			m.printError(nil, err)
			return
		}
		caller.(*values.Object).Put(callerStateField, &callerState{listener: ls, service: svc, message: msg})
		args = append(args, caller)
	}
	if panicErr := m.invokeTrapped(ctx, handle, args); panicErr != nil {
		m.invokeOnError(ctx, svc, panicErr)
	}
}

func (m *mqttModule) invokeOnError(ctx *extern.Context, svc *values.Object, mErr *mqttError) {
	errValue, err := toBalError(ctx, mErr)
	if err != nil {
		m.printError(nil, err)
		return
	}
	handle, ok := ctx.LookupRemoteMethod(svc, "onError")
	if !ok {
		m.printError(errValue, nil)
		return
	}
	m.printPanic(m.invokeTrapped(ctx, handle, []values.BalValue{svc, errValue}))
}

// invokeTrapped calls a service method, prints an error it returns and returns a panic it
// raises, as jBallerina's listener callback does instead of crashing the process.
func (m *mqttModule) invokeTrapped(ctx *extern.Context, handle extern.MethodHandle, args []values.BalValue) (panicErr *mqttError) {
	defer func() {
		if rec := recover(); rec != nil {
			ctx.ReleaseAllHeldLocks()
			panicErr = newError(panicMessage(rec))
		}
	}()
	m.printError(ctx.InvokeMethod(handle, args))
	return nil
}

func (m *mqttModule) printPanic(panicErr *mqttError) {
	if panicErr != nil {
		m.printError(nil, panicErr)
	}
}

func panicMessage(r any) string {
	switch v := r.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r)
	}
}

func callerOf(self *values.Object) *callerState {
	v, _ := self.Get(callerStateField)
	cs, _ := v.(*callerState)
	return cs
}

// callerComplete acknowledges the message in manualAcks mode and reports a failure to do so,
// as clarified in ballerina-library#9326.
func (m *mqttModule) callerComplete(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cs := callerOf(args[0].(*values.Object))
	if !cs.listener.manualAcks {
		return nil, nil
	}
	return result(ctx, cs.listener.conn.complete(cs.message))
}

// callerRespond publishes the response to the request's response topic, carrying the
// request's correlation data, and reports a failed publish (ballerina-library#9326).
// onComplete is invoked once the response is delivered.
func (m *mqttModule) callerRespond(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cs := callerOf(args[0].(*values.Object))
	response, _ := args[1].(*values.Map)
	msg, mErr := parseOutgoingMessage(response)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	props := cs.message.publish.Properties
	if props == nil || props.ResponseTopic == "" {
		return toBalError(ctx, newError("Response topic is not set"))
	}
	if props.CorrelationData != nil {
		msg.correlationData = props.CorrelationData
	}
	ls := cs.listener
	topic := props.ResponseTopic
	packetID, mErr := ls.conn.publish(topic, msg)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	if !ls.events.put(listenerEvent{kind: eventComplete, service: cs.service, packetID: packetID, topic: topic}) {
		return toBalError(ctx, newReasonError(reasonClientClosed, nil))
	}
	return nil, nil
}
