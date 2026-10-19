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
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/eclipse/paho.golang/packets"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/session"
	"github.com/eclipse/paho.golang/paho/session/state"
)

type connectionState int

const (
	stateConnecting connectionState = iota
	stateConnected
	stateDisconnecting
	stateDisconnected
	stateClosed
)

const (
	// paho.golang bounds every acknowledgement wait by PacketTimeout; Paho Java waits
	// indefinitely, so the bound is effectively removed.
	unboundedPacketTimeout = 100 * 365 * 24 * time.Hour
	disconnectQuiesce      = 30 * time.Second
	initialReconnectDelay  = time.Second
	connectionLossGrace    = time.Second
)

type receivedMessage struct {
	client  *paho.Client
	publish *paho.Publish
}

// connectionHandlers receive events from the connection. They are invoked from paho or
// monitor goroutines and must not block.
type connectionHandlers struct {
	onMessage        func(receivedMessage)
	onConnectionLost func(*mqttError)
}

// activeLink is one network connection; a new link is created on every (re)connect while the
// session state, and with it the packet-identifier sequence, is shared across links.
type activeLink struct {
	client          *paho.Client
	reasonRecorded  chan struct{}
	mu              sync.Mutex
	lostErr         *mqttError
	serverInitiated bool
}

func newActiveLink() *activeLink {
	return &activeLink{reasonRecorded: make(chan struct{})}
}

func (l *activeLink) recordLoss(err *mqttError, serverInitiated bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lostErr == nil {
		l.lostErr = err
		l.serverInitiated = serverInitiated
		close(l.reasonRecorded)
	}
}

// operationContext is cancelled when the link drops: paho does not complete a pending
// acknowledgement wait when the server closes the connection.
func (l *activeLink) operationContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-l.client.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// awaitLossReason waits briefly for paho's disconnect callbacks, which run after the
// connection is already closed, and reports whether the server initiated the disconnect.
func (l *activeLink) awaitLossReason() bool {
	select {
	case <-l.reasonRecorded:
	case <-time.After(connectionLossGrace):
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.serverInitiated
}

func (l *activeLink) loss(cause error) *mqttError {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lostErr != nil {
		return l.lostErr
	}
	return newReasonError(reasonConnectionLost, cause)
}

type connection struct {
	platform pal.Platform
	opts     *connectionOptions
	handlers connectionHandlers
	session  *packetIDSession

	mu            sync.Mutex
	state         connectionState
	link          *activeLink
	stopReconnect chan struct{}
}

func newConnection(platform pal.Platform, opts *connectionOptions, handlers connectionHandlers) *connection {
	return &connection{
		platform: platform,
		opts:     opts,
		handlers: handlers,
		session:  &packetIDSession{SessionManager: state.NewInMemory()},
		state:    stateDisconnected,
	}
}

// connect performs the initial, synchronous connection made by Client/Listener init.
func (c *connection) connect() *mqttError {
	c.mu.Lock()
	c.state = stateConnecting
	c.mu.Unlock()
	if err := c.connectToAny(); err != nil {
		c.setState(stateDisconnected)
		return err
	}
	return nil
}

func (c *connection) connectToAny() *mqttError {
	var lastErr *mqttError
	for _, uri := range c.opts.serverURIs {
		link, err := c.connectTo(uri)
		if err == nil {
			c.mu.Lock()
			c.link = link
			c.state = stateConnected
			c.mu.Unlock()
			go c.monitor(link)
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func (c *connection) connectTo(uri serverURI) (*activeLink, *mqttError) {
	tlsConfig := c.opts.tlsConfig
	if tlsConfig != nil && !uri.secure {
		return nil, newReasonError(reasonSocketFactoryMismatch, nil)
	}
	if tlsConfig == nil && uri.secure {
		tlsConfig = &pal.TLSConfig{}
	}
	ctx := context.Background()
	if c.opts.connectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.opts.connectTimeout)
		defer cancel()
	}
	conn, err := c.platform.Net.Dial(ctx, "tcp", uri.address, "", tlsConfig)
	if err != nil {
		return nil, dialError(err)
	}
	link := newActiveLink()
	link.client = paho.NewClient(paho.ClientConfig{
		ClientID: c.opts.clientID,
		// paho writes a packet in several Write calls, so concurrent writers (publishes, acks,
		// pings) must be serialised or their bytes interleave on TLS connections.
		Conn:                       packets.NewThreadSafeConn(conn),
		Session:                    c.session,
		PingHandler:                keepAlivePinger{paho.NewDefaultPinger()},
		PacketTimeout:              unboundedPacketTimeout,
		EnableManualAcknowledgment: c.opts.manualAcks,
		OnPublishReceived: []func(paho.PublishReceived) (bool, error){
			func(pr paho.PublishReceived) (bool, error) {
				c.handlers.onMessage(receivedMessage{client: pr.Client, publish: pr.Packet})
				return true, nil
			},
		},
		OnServerDisconnect: func(d *paho.Disconnect) {
			link.recordLoss(serverDisconnectedError(d.ReasonCode), true)
		},
		OnClientError: func(err error) {
			link.recordLoss(newReasonError(reasonConnectionLost, err), false)
		},
	})
	connack, err := link.client.Connect(ctx, c.connectPacket())
	if err != nil {
		if connack != nil && connack.ReasonCode >= 0x80 {
			return nil, newReasonError(int(connack.ReasonCode), nil)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, newReasonError(reasonClientTimeout, nil)
		}
		return nil, newReasonError(reasonServerConnectError, err)
	}
	return link, nil
}

func dialError(err error) *mqttError {
	if errors.Is(err, context.DeadlineExceeded) {
		return newReasonError(reasonClientTimeout, nil)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return newReasonError(reasonServerConnectError, err)
	}
	return newReasonError(reasonClientException, err)
}

func (c *connection) connectPacket() *paho.Connect {
	cp := &paho.Connect{
		ClientID:   c.opts.clientID,
		KeepAlive:  c.opts.keepAlive,
		CleanStart: c.opts.cleanStart,
	}
	if c.opts.username != nil {
		cp.Username = *c.opts.username
		cp.UsernameFlag = true
	}
	if c.opts.password != nil {
		cp.Password = c.opts.password
		cp.PasswordFlag = true
	}
	if will := c.opts.will; will != nil {
		cp.WillMessage = &paho.WillMessage{
			Topic:   will.topic,
			Payload: will.message.payload,
			QoS:     will.message.qos,
			Retain:  will.message.retained,
		}
	}
	return cp
}

// monitor waits for the link to drop and, unless the drop was requested, starts the automatic
// reconnect cycle when enabled. As observed on jBallerina, a lost network connection is
// reported to the connection-loss handler but a server-initiated DISCONNECT is not.
func (c *connection) monitor(link *activeLink) {
	<-link.client.Done()
	c.mu.Lock()
	if c.link != link || c.state != stateConnected {
		c.mu.Unlock()
		return
	}
	c.state = stateDisconnected
	c.mu.Unlock()
	if !link.awaitLossReason() && c.handlers.onConnectionLost != nil {
		c.handlers.onConnectionLost(link.loss(nil))
	}
	if c.opts.automaticReconnect {
		c.startReconnectCycle()
	}
}

func (c *connection) startReconnectCycle() {
	c.mu.Lock()
	stop := make(chan struct{})
	c.stopReconnect = stop
	c.mu.Unlock()
	go c.reconnectCycle(stop)
}

func (c *connection) reconnectCycle(stop chan struct{}) {
	delay := initialReconnectDelay
	for {
		select {
		case <-stop:
			return
		case <-time.After(delay):
		}
		c.mu.Lock()
		if c.state != stateDisconnected || c.stopReconnect != stop {
			c.mu.Unlock()
			return
		}
		c.state = stateConnecting
		c.mu.Unlock()
		if err := c.connectToAny(); err == nil {
			return
		}
		c.setState(stateDisconnected)
		delay = min(delay*2, c.opts.maxReconnectDelay)
	}
}

func (c *connection) cancelReconnectCycleLocked() {
	if c.stopReconnect != nil {
		close(c.stopReconnect)
		c.stopReconnect = nil
	}
}

func (c *connection) setState(s connectionState) {
	c.mu.Lock()
	c.state = s
	c.mu.Unlock()
}

func (c *connection) isConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == stateConnected
}

func (c *connection) connectedLink() (*activeLink, *mqttError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateConnected {
		return nil, newReasonError(reasonClientNotConnected, nil)
	}
	return c.link, nil
}

// publish sends the message and waits for its delivery to complete, returning the packet
// identifier assigned by the session (0 for QoS 0).
func (c *connection) publish(topic string, msg outgoingMessage) (uint16, *mqttError) {
	link, mErr := c.connectedLink()
	if mErr != nil {
		return 0, mErr
	}
	var packetID uint16
	ctx, cancel := link.operationContext()
	defer cancel()
	ctx = context.WithValue(ctx, packetIDSinkKey{}, &packetID)
	p := &paho.Publish{
		Topic:   topic,
		QoS:     msg.qos,
		Retain:  msg.retained,
		Payload: msg.payload,
	}
	if msg.responseTopic != "" || msg.correlationData != nil {
		p.Properties = &paho.PublishProperties{ResponseTopic: msg.responseTopic, CorrelationData: msg.correlationData}
	}
	resp, err := link.client.Publish(ctx, p)
	if err != nil {
		if resp != nil && resp.ReasonCode >= 0x80 {
			return 0, newReasonError(int(resp.ReasonCode), nil)
		}
		return 0, c.operationError(link, err)
	}
	return packetID, nil
}

func (c *connection) subscribe(subscriptions []paho.SubscribeOptions) *mqttError {
	link, mErr := c.connectedLink()
	if mErr != nil {
		return mErr
	}
	ctx, cancel := link.operationContext()
	defer cancel()
	if _, err := link.client.Subscribe(ctx, &paho.Subscribe{Subscriptions: subscriptions}); err != nil {
		return c.operationError(link, err)
	}
	return nil
}

// operationError reports a failed publish/subscribe with the reason the connection dropped,
// as Paho Java does, when the failure was caused by the connection going away.
func (c *connection) operationError(link *activeLink, err error) *mqttError {
	select {
	case <-link.client.Done():
		link.awaitLossReason()
		return link.loss(err)
	case <-time.After(connectionLossGrace):
		return newErrorWithCause(err.Error(), err)
	}
}

func (c *connection) ack(msg receivedMessage) {
	_ = msg.client.Ack(msg.publish)
}

func (c *connection) disconnect() *mqttError {
	c.mu.Lock()
	switch c.state {
	case stateClosed:
		c.mu.Unlock()
		return newReasonError(reasonClientClosed, nil)
	case stateDisconnected:
		c.mu.Unlock()
		return newReasonError(reasonClientDisconnected, nil)
	case stateDisconnecting:
		c.mu.Unlock()
		return newReasonError(reasonClientDisconnecting, nil)
	case stateConnecting:
		c.mu.Unlock()
		return newReasonError(reasonConnectInProgress, nil)
	}
	c.state = stateDisconnecting
	c.cancelReconnectCycleLocked()
	link := c.link
	c.mu.Unlock()
	c.closeLink(link)
	c.setState(stateDisconnected)
	return nil
}

// disconnectForcibly is the immediateStop path: it never reports the connection state.
func (c *connection) disconnectForcibly() {
	c.mu.Lock()
	c.cancelReconnectCycleLocked()
	if c.state != stateConnected {
		c.mu.Unlock()
		return
	}
	c.state = stateDisconnecting
	link := c.link
	c.mu.Unlock()
	c.closeLink(link)
	c.setState(stateDisconnected)
}

func (c *connection) closeLink(link *activeLink) {
	_ = link.client.Disconnect(&paho.Disconnect{ReasonCode: 0})
	select {
	case <-link.client.Done():
	case <-time.After(disconnectQuiesce):
	}
}

func (c *connection) close() *mqttError {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case stateClosed:
		return nil
	case stateConnected:
		return newReasonError(reasonClientConnected, nil)
	case stateConnecting:
		return newReasonError(reasonConnectInProgress, nil)
	case stateDisconnecting:
		return newReasonError(reasonClientDisconnecting, nil)
	}
	c.state = stateClosed
	c.cancelReconnectCycleLocked()
	_ = c.session.Close()
	return nil
}

// reconnect starts a reconnect attempt and returns without waiting for it, like Paho Java's
// MqttClient.reconnect().
func (c *connection) reconnect() *mqttError {
	c.mu.Lock()
	switch c.state {
	case stateConnected:
		c.mu.Unlock()
		return newReasonError(reasonClientConnected, nil)
	case stateConnecting:
		c.mu.Unlock()
		return newReasonError(reasonConnectInProgress, nil)
	case stateDisconnecting:
		c.mu.Unlock()
		return newReasonError(reasonClientDisconnecting, nil)
	case stateClosed:
		c.mu.Unlock()
		return newReasonError(reasonClientClosed, nil)
	}
	c.cancelReconnectCycleLocked()
	c.state = stateConnecting
	c.mu.Unlock()
	go func() {
		if err := c.connectToAny(); err != nil {
			c.setState(stateDisconnected)
			if c.opts.automaticReconnect {
				c.startReconnectCycle()
			}
		}
	}()
	return nil
}

// keepAlivePinger sends the first PINGREQ only after a keep-alive interval of inactivity, as
// Paho Java does. paho's DefaultPinger pings as soon as CONNACK arrives, which Mosquitto may
// reject with "PINGREQ before session is active" and drop the connection.
type keepAlivePinger struct {
	*paho.DefaultPinger
}

func (p keepAlivePinger) Run(ctx context.Context, conn net.Conn, keepAlive uint16) error {
	p.PacketSent()
	p.PacketReceived()
	return p.DefaultPinger.Run(ctx, conn, keepAlive)
}

type packetIDSinkKey struct{}

// packetIDSession exposes the packet identifier the session allocates for a publish, which
// paho.golang does not return, so DeliveryToken.messageId matches the identifier on the wire.
type packetIDSession struct {
	session.SessionManager
}

func (s *packetIDSession) AddToSession(ctx context.Context, packet session.Packet, resp chan<- packets.ControlPacket) error {
	if err := s.SessionManager.AddToSession(ctx, packet, resp); err != nil {
		return err
	}
	if sink, ok := ctx.Value(packetIDSinkKey{}).(*uint16); ok {
		if pub, ok := packet.(*packets.Publish); ok {
			*sink = pub.PacketID
		}
	}
	return nil
}
