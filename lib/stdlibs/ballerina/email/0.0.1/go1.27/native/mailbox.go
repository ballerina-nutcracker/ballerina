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
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	// defaultFolder is email:DEFAULT_FOLDER.
	defaultFolder = "INBOX"
	closedError   = "Not connected"
)

var (
	// receiveTimeout bounds the connection and each wait on a POP3 or IMAP server; neither
	// protocol defines client timeouts, so it follows SMTP's command timeout.
	receiveTimeout = 5 * time.Minute
	// pollInterval is the first wait between checks of an empty folder; it doubles after each
	// empty check up to maxPollInterval, so a long wait does not log in every second.
	pollInterval    = time.Second
	maxPollInterval = 30 * time.Second
)

// mailbox is the configuration of a POP3 or IMAP client. Each receive is its own session, so
// the client holds no connection.
type mailbox struct {
	mailServer
	closed atomic.Bool
	// reading serialises the reads of one client, so they never return the same message.
	reading sync.Mutex
}

// convertMessage converts a received message; tests replace it to make the conversion fail.
var convertMessage = convertReceived

// retrieveFunc returns the first message of a folder with a function that ends its session,
// marking the message as received when received is true, or nil data when the folder is empty.
type retrieveFunc func() (data []byte, finish func(received bool), err *values.Error)

// newMailbox reads the init arguments of a POP3 or IMAP client; secureSocket errors get the
// properties prefix.
func newMailbox(rt *runtime.Runtime, args []values.BalValue, propertiesPrefix string) (*mailbox, *values.Error) {
	m := &mailbox{}
	m.host, _ = args[1].(string)
	m.username, _ = args[2].(string)
	m.password, _ = args[3].(string)
	config, _ := args[4].(*values.Map)
	port, _ := config.Get("port")
	m.port, _ = port.(int64)
	m.security, _ = stringField(config, "security")
	tlsCfg, err := tlsConfigFromSecureSocket(rt, config)
	if err != nil {
		err.Message = propertiesPrefix + err.Message
		return nil, err
	}
	m.tls = tlsCfg
	return m, nil
}

// receive returns the first message retrieve finds, checking again with a growing interval
// until timeout seconds have passed or the client is closed.
func (m *mailbox) receive(ctx *extern.Context, platform pal.Platform, strArrTy semtypes.SemType,
	timeout *decimal.Decimal, retrieve retrieveFunc) (values.BalValue, error) {
	deadline := saturatingAdd(platform.Time.MonotonicNow(), secondsDuration(timeout))
	interval := pollInterval
	for {
		msg, err := m.next(ctx, strArrTy, retrieve)
		if msg != nil || err != nil {
			return msg, err
		}
		remaining := deadline - platform.Time.MonotonicNow()
		if remaining <= 0 {
			return nil, nil
		}
		wait(ctx, platform, min(remaining, interval), m.closed.Load)
		interval = min(2*interval, maxPollInterval)
		if m.closed.Load() {
			return emailError(closedError), nil
		}
	}
}

// next retrieves and converts the first message, holding the client's read lock so a
// concurrent read on the client cannot retrieve the same message.
func (m *mailbox) next(ctx *extern.Context, strArrTy semtypes.SemType, retrieve retrieveFunc) (values.BalValue, error) {
	// TryLock with a yield keeps other strands on this thread, including the lock holder, running.
	for !m.reading.TryLock() {
		<-ctx.Yield()
	}
	defer m.reading.Unlock()
	data, finish, err := retrieve()
	if err != nil {
		return err, nil
	}
	if data == nil {
		return nil, nil
	}
	msg, convErr := convertMessage(ctx, strArrTy, data)
	if finish != nil {
		finish(convErr == nil)
	}
	return msg, convErr
}

// dial connects to the server, with implicit TLS in the SSL security mode.
func (m *mailbox) dial(platform pal.Platform) (net.Conn, *values.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), receiveTimeout)
	defer cancel()
	var implicitTLS *pal.TLSConfig
	if m.security == securitySSL {
		implicitTLS = m.tls
	}
	conn, err := platform.Net.Dial(ctx, "tcp", m.address(), "", implicitTLS)
	if err != nil {
		return nil, emailErrorWithCause(fmt.Sprintf("Couldn't connect to host, port: %s, %d; timeout %d",
			m.host, m.port, receiveTimeout.Milliseconds()), err)
	}
	return conn, nil
}

// upgradeTLS starts TLS on conn after the server accepted STARTTLS (STLS for POP3).
func (m *mailbox) upgradeTLS(platform pal.Platform, conn net.Conn) (net.Conn, *values.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), receiveTimeout)
	defer cancel()
	tlsConn, err := platform.Net.UpgradeTLS(ctx, conn, m.address(), m.tls)
	if err != nil {
		_ = conn.Close()
		return nil, emailErrorWithCause("Could not convert socket to TLS", err)
	}
	return tlsConn, nil
}

func (m *mailbox) startTLS() bool {
	return m.security == securityStartTLSAuto || m.security == securityStartTLSAlways
}

// withTimeout bounds each read and write on conn by receiveTimeout.
func withTimeout(platform pal.Platform, conn net.Conn) net.Conn {
	timeout := receiveTimeout
	return deadlineConn{Conn: conn, now: platform.Time.Now, timeout: &timeout}
}

// secondsDuration converts seconds to a duration, clamping values beyond the int64 range.
func secondsDuration(seconds *decimal.Decimal) time.Duration {
	nanos := seconds.Float64() * float64(time.Second)
	if nanos <= 0 {
		return 0
	}
	if nanos >= math.MaxInt64 {
		return math.MaxInt64
	}
	return time.Duration(nanos)
}

func saturatingAdd(now, d time.Duration) time.Duration {
	if d > 0 && now > math.MaxInt64-d {
		return math.MaxInt64
	}
	return now + d
}

// wait yields the strand until d has passed or stop returns true, as lang.runtime:sleep does,
// so other strands on the thread keep running.
func wait(ctx *extern.Context, platform pal.Platform, d time.Duration, stop func() bool) {
	deadline := platform.Time.MonotonicNow() + d
	for platform.Time.MonotonicNow() < deadline && !stop() {
		<-ctx.Yield()
	}
}

// closeExtern closes the client stored under field; later reads return closedError.
func closeExtern(field string) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		mailboxOf(args[0], field).closed.Store(true)
		return nil, nil
	}
}

// mailboxOf returns the client state stored under field.
func mailboxOf(v values.BalValue, field string) *mailbox {
	self, _ := v.(*values.Object)
	state, _ := self.Get(field)
	m, _ := state.(*mailbox)
	return m
}
