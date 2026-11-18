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

package native

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/textproto"
	"strings"
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
	popStateField       = "$pop"
	popConnectPrefix    = "Error while connecting to the POP3 store: "
	popPropertiesPrefix = "Error while initializing POP3 client properties: "
	popFolderError      = "folder is not INBOX"
	popClosedError      = "Not connected"
	popOpenError        = "Open failed"
	popFetchError       = "error fetching POP3 content"
	popStartTLSMissing  = "STLS is required but host does not support STLS"
	popCredentialsError = "Username and password must not contain CR or LF characters"
)

var (
	// popTimeout bounds the connection and each wait on the server; POP3 defines no client
	// timeouts, so it follows SMTP's command timeout.
	popTimeout = 5 * time.Minute
	// popPollInterval is the first wait between checks of an empty mailbox; it doubles after each
	// empty check up to popMaxPollInterval, so a long wait does not log in every second.
	popPollInterval    = time.Second
	popMaxPollInterval = 30 * time.Second
)

// popClient is the connection configuration stored on an email:PopClient. Each receive is its
// own POP3 session, so the client holds no connection.
type popClient struct {
	mailServer
	closed atomic.Bool
}

// popSession is an authenticated POP3 connection.
type popSession struct {
	conn net.Conn
	text *textproto.Conn
}

// popReplyError is a negative (-ERR) POP3 reply.
type popReplyError struct {
	msg string
}

func (e *popReplyError) Error() string {
	return e.msg
}

func initPopClientExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self, _ := args[0].(*values.Object)
		client := &popClient{}
		client.host, _ = args[1].(string)
		client.username, _ = args[2].(string)
		client.password, _ = args[3].(string)
		config, _ := args[4].(*values.Map)
		port, _ := config.Get("port")
		client.port, _ = port.(int64)
		client.security, _ = stringField(config, "security")
		tlsCfg, err := tlsConfigFromSecureSocket(rt, config)
		if err != nil {
			err.Message = popPropertiesPrefix + err.Message
			return err, nil
		}
		client.tls = tlsCfg
		if strings.ContainsAny(client.username+client.password, "\r\n") {
			return emailError(popCredentialsError), nil
		}
		// Like jBallerina, init connects so that bad hosts and credentials fail here.
		s, err := client.connect(rt.Platform())
		if err != nil {
			err.Message = popConnectPrefix + err.Message
			return err, nil
		}
		s.quit()
		s.close()
		self.Put(popStateField, client)
		return nil, nil
	}
}

func popReadExtern(rt *runtime.Runtime, strArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		client := popClientOf(args[0])
		folder, _ := args[1].(string)
		timeout, _ := args[2].(*decimal.Decimal)
		if client.closed.Load() {
			return emailError(popClosedError), nil
		}
		if !strings.EqualFold(folder, defaultFolder) {
			return emailError(popFolderError), nil
		}
		platform := rt.Platform()
		deadline := saturatingAdd(platform.Time.MonotonicNow(), secondsDuration(timeout))
		interval := popPollInterval
		for {
			data, err := client.retrieveFirst(platform)
			if err != nil {
				return err, nil
			}
			if data != nil {
				return convertReceived(ctx, strArrTy, data)
			}
			remaining := deadline - platform.Time.MonotonicNow()
			if remaining <= 0 {
				return nil, nil
			}
			wait(ctx, platform, min(remaining, interval), client.closed.Load)
			interval = min(2*interval, popMaxPollInterval)
			if client.closed.Load() {
				return emailError(popClosedError), nil
			}
		}
	}
}

func popCloseExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	popClientOf(args[0]).closed.Store(true)
	return nil, nil
}

func popClientOf(v values.BalValue) *popClient {
	self, _ := v.(*values.Object)
	state, _ := self.Get(popStateField)
	client, _ := state.(*popClient)
	return client
}

// defaultFolder is email:DEFAULT_FOLDER, the only folder POP3 has.
const defaultFolder = "INBOX"

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

// retrieveFirst returns the first message in the mailbox and deletes it, or nil when the
// mailbox is empty. The deletion is committed by QUIT in the same session. If DELE or QUIT fails
// the message is still returned, and it stays on the server unless QUIT failed after deleting it.
func (c *popClient) retrieveFirst(platform pal.Platform) ([]byte, *values.Error) {
	s, err := c.connect(platform)
	if err != nil {
		return nil, err
	}
	defer s.close()
	stat, cmdErr := s.cmd("STAT")
	if cmdErr != nil {
		return nil, popFailure(popOpenError, cmdErr)
	}
	if count, _, _ := strings.Cut(stat, " "); count == "0" {
		s.quit()
		return nil, nil
	}
	if _, cmdErr = s.cmd("RETR 1"); cmdErr != nil {
		return nil, popFailure(popFetchError, cmdErr)
	}
	data, readErr := s.readMultiline()
	if readErr != nil {
		return nil, popFailure(popFetchError, readErr)
	}
	if _, cmdErr = s.cmd("DELE 1"); cmdErr == nil {
		s.quit()
	}
	return data, nil
}

// popFailure reports a negative reply with the server's text, as JavaMail does, and any other
// failure with message and the error as its cause.
func popFailure(message string, err error) *values.Error {
	if reply, ok := errors.AsType[*popReplyError](err); ok {
		return emailError(reply.msg)
	}
	return emailErrorWithCause(message, err)
}

// connect opens an authenticated session, with implicit TLS or STLS as the security mode selects.
func (c *popClient) connect(platform pal.Platform) (*popSession, *values.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), popTimeout)
	defer cancel()
	var implicitTLS *pal.TLSConfig
	if c.security == securitySSL {
		implicitTLS = c.tls
	}
	conn, err := platform.Net.Dial(ctx, "tcp", c.address(), "", implicitTLS)
	if err != nil {
		return nil, emailErrorWithCause(fmt.Sprintf("Couldn't connect to host, port: %s, %d; timeout %d",
			c.host, c.port, popTimeout.Milliseconds()), err)
	}
	s := newPopSession(platform, conn)
	if _, err := s.reply(); err != nil {
		s.close()
		return nil, popFailure("Connect failed", err)
	}
	if c.security == securityStartTLSAuto || c.security == securityStartTLSAlways {
		var tlsErr *values.Error
		if s, tlsErr = c.startTLS(platform, s); tlsErr != nil {
			return nil, tlsErr
		}
	}
	if _, err := s.cmd("USER %s", c.username); err != nil {
		s.close()
		return nil, popFailure("Connect failed", err)
	}
	if _, err := s.cmd("PASS %s", c.password); err != nil {
		s.close()
		return nil, popFailure("Connect failed", err)
	}
	return s, nil
}

// startTLS upgrades the session with STLS when the server offers it; START_TLS_ALWAYS fails
// when it does not.
func (c *popClient) startTLS(platform pal.Platform, s *popSession) (*popSession, *values.Error) {
	offered, err := s.offersSTLS()
	if err != nil {
		s.close()
		return nil, popFailure("Connect failed", err)
	}
	if !offered {
		if c.security == securityStartTLSAlways {
			s.close()
			return nil, emailError(popStartTLSMissing)
		}
		return s, nil
	}
	if _, err := s.cmd("STLS"); err != nil {
		// Like JavaMail, START_TLS_AUTO continues in plaintext when the server refuses STLS.
		if _, refused := errors.AsType[*popReplyError](err); refused && c.security == securityStartTLSAuto {
			return s, nil
		}
		s.close()
		return nil, popFailure("Connect failed", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), popTimeout)
	defer cancel()
	tlsConn, err := platform.Net.UpgradeTLS(ctx, s.conn, c.address(), c.tls)
	if err != nil {
		s.close()
		return nil, emailErrorWithCause("Could not convert socket to TLS", err)
	}
	return newPopSession(platform, tlsConn), nil
}

func newPopSession(platform pal.Platform, conn net.Conn) *popSession {
	return &popSession{conn: conn, text: textproto.NewConn(idleConn{Conn: conn, now: platform.Time.Now})}
}

// idleConn bounds each read and write by popTimeout, so a long RETR transfer continues while data
// keeps arriving.
type idleConn struct {
	net.Conn
	now func() time.Time
}

func (c idleConn) Read(p []byte) (int, error) {
	if err := c.SetReadDeadline(c.now().Add(popTimeout)); err != nil {
		return 0, err
	}
	return c.Conn.Read(p)
}

func (c idleConn) Write(p []byte) (int, error) {
	if err := c.SetWriteDeadline(c.now().Add(popTimeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}

func (s *popSession) close() {
	_ = s.text.Close()
}

// quit ends the session, committing deletions; a failure leaves the messages on the server.
func (s *popSession) quit() {
	_, _ = s.cmd("QUIT")
}

func (s *popSession) cmd(format string, args ...any) (string, error) {
	if err := s.text.PrintfLine(format, args...); err != nil {
		return "", err
	}
	return s.reply()
}

// reply reads a status line, returning the text after +OK, a popReplyError for -ERR, and a
// protocol error for any other line, as JavaMail does.
func (s *popSession) reply() (string, error) {
	line, err := s.text.ReadLine()
	if err != nil {
		return "", err
	}
	if text, ok := strings.CutPrefix(line, "+OK"); ok {
		return strings.TrimSpace(text), nil
	}
	if text, ok := strings.CutPrefix(line, "-ERR"); ok {
		return "", &popReplyError{strings.TrimSpace(text)}
	}
	return "", textproto.ProtocolError("unexpected response: " + line)
}

// readMultiline reads a dot-terminated response, undoing dot-stuffing and keeping CRLF line ends.
func (s *popSession) readMultiline() ([]byte, error) {
	out := []byte{}
	for {
		line, err := s.text.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == "." {
			return out, nil
		}
		out = append(out, strings.TrimPrefix(line, ".")...)
		out = append(out, '\r', '\n')
	}
}

// offersSTLS reports whether CAPA lists STLS; a server that rejects CAPA offers nothing.
func (s *popSession) offersSTLS() (bool, error) {
	if _, err := s.cmd("CAPA"); err != nil {
		if _, rejected := errors.AsType[*popReplyError](err); rejected {
			return false, nil
		}
		return false, err
	}
	capabilities, err := s.readMultiline()
	if err != nil {
		return false, err
	}
	for line := range strings.SplitSeq(string(capabilities), "\r\n") {
		if strings.EqualFold(strings.TrimSpace(line), "STLS") {
			return true, nil
		}
	}
	return false, nil
}
