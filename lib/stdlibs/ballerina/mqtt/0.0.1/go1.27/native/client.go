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
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
	"github.com/eclipse/paho.golang/paho"
)

const (
	clientStateField = "$client"
	queueField       = "$queue"
)

type clientState struct {
	conn *connection
	// Every receive() stream of a client shares this queue, as in the Java implementation.
	responses *blockingQueue[receivedMessage]
}

func clientOf(self *values.Object) *clientState {
	v, _ := self.Get(clientStateField)
	cs, _ := v.(*clientState)
	return cs
}

func (m *mqttModule) registerClient() {
	reg := func(name string, fn extern.NativeFunc) {
		runtime.RegisterExternFunction(m.rt, orgName, moduleName, name, fn)
	}
	reg("Client.externInit", m.clientInit)
	reg("Client.externPublish", m.clientPublish)
	reg("Client.externSubscribe", m.clientSubscribe)
	reg("Client."+model.RemoteMethodName("receive"), m.clientReceive)
	reg("Client.externClose", m.clientClose)
	reg("Client.externIsConnected", m.clientIsConnected)
	reg("Client.externDisconnect", m.clientDisconnect)
	reg("Client.externReconnect", m.clientReconnect)
	reg("StreamIterator.nextResult", m.streamNextResult)
	reg("StreamIterator.closeStream", m.streamClose)
}

func (m *mqttModule) clientInit(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	serverURI, _ := args[1].(string)
	clientID, _ := args[2].(string)
	config, _ := args[3].(*values.Map)
	opts, mErr := parseConnectionOptions(m.rt.Platform(), serverURI, clientID, config)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	responses := newBlockingQueue[receivedMessage]()
	conn := newConnection(m.rt.Platform(), opts, connectionHandlers{onMessage: responses.put})
	if mErr := conn.connect(); mErr != nil {
		return toBalError(ctx, mErr)
	}
	self.Put(clientStateField, &clientState{conn: conn, responses: responses})
	return nil, nil
}

func (m *mqttModule) clientPublish(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cs := clientOf(args[0].(*values.Object))
	topic, _ := args[1].(string)
	message, _ := args[2].(*values.Map)
	msg, mErr := parseOutgoingMessage(message)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	if mErr := validateTopic(topic, false); mErr != nil {
		return toBalError(ctx, mErr)
	}
	packetID, mErr := cs.conn.publish(topic, msg)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	return deliveryToken(ctx, packetID, topic)
}

func (m *mqttModule) clientSubscribe(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cs := clientOf(args[0].(*values.Object))
	subscriptions, _ := args[1].(*values.List)
	options, mErr := subscribeOptions(subscriptions)
	if mErr != nil {
		return toBalError(ctx, mErr)
	}
	return result(ctx, cs.conn.subscribe(options))
}

func subscribeOptions(subscriptions *values.List) ([]paho.SubscribeOptions, *mqttError) {
	options := make([]paho.SubscribeOptions, 0, subscriptions.Len())
	for i := 0; i < subscriptions.Len(); i++ {
		sub, _ := subscriptions.Get(i).(*values.Map)
		topic, _ := mapString(sub, "topic")
		qos, _ := mapInt(sub, "qos")
		if mErr := validateTopic(topic, true); mErr != nil {
			return nil, mErr
		}
		if mErr := validateQoS(qos); mErr != nil {
			return nil, mErr
		}
		options = append(options, paho.SubscribeOptions{Topic: topic, QoS: byte(qos)})
	}
	return options, nil
}

// clientReceive returns a stream backed by a StreamIterator object. The stream's next/close
// run the iterator's Ballerina methods on a context of their own so the stream can be
// consumed from any strand.
func (m *mqttModule) clientReceive(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cs := clientOf(args[0].(*values.Object))
	td, _ := args[1].(*values.TypeDesc)
	v, err := callHelper(ctx, "createStreamIterator")
	if err != nil {
		return nil, err
	}
	iterator := v.(*values.Object)
	iterator.Put(queueField, cs.responses)
	next := func() values.BalValue { return m.invokeIteratorMethod(iterator, "next") }
	closeFn := func() values.BalValue { return m.invokeIteratorMethod(iterator, "close") }
	return values.NewStream(td.Constraint(), next, closeFn), nil
}

func (m *mqttModule) invokeIteratorMethod(iterator *values.Object, name string) values.BalValue {
	ctx := m.rt.AcquirePooledContext()
	defer m.rt.ReleasePooledContext(ctx)
	handle, _ := ctx.LookupObjectMethod(iterator, name)
	v, err := ctx.InvokeMethod(handle, []values.BalValue{iterator})
	if err != nil {
		panic(values.NewErrorWithMessage(err.Error()))
	}
	return v
}

func (m *mqttModule) streamNextResult(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	v, _ := self.Get(queueField)
	queue, ok := v.(*blockingQueue[receivedMessage])
	if !ok {
		return nil, nil
	}
	msg, ok := queue.take()
	if !ok {
		return nil, nil
	}
	return m.toBalMessage(ctx, msg)
}

func (m *mqttModule) streamClose(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if v, _ := self.Get(queueField); v != nil {
		if queue, ok := v.(*blockingQueue[receivedMessage]); ok {
			queue.clear()
		}
	}
	self.Put(queueField, nil)
	return nil, nil
}

func (m *mqttModule) clientClose(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return result(ctx, clientOf(args[0].(*values.Object)).conn.close())
}

func (m *mqttModule) clientIsConnected(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return clientOf(args[0].(*values.Object)).conn.isConnected(), nil
}

func (m *mqttModule) clientDisconnect(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return result(ctx, clientOf(args[0].(*values.Object)).conn.disconnect())
}

func (m *mqttModule) clientReconnect(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return result(ctx, clientOf(args[0].(*values.Object)).conn.reconnect())
}
