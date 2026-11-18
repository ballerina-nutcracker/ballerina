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
	"errors"
	"net"
	"net/textproto"
	"strings"

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
	popOpenError        = "Open failed"
	popFetchError       = "error fetching POP3 content"
	popStartTLSMissing  = "STLS is required but host does not support STLS"
	popCredentialsError = "Username and password must not contain CR or LF characters"
)

// popClient runs the POP3 sessions of an email:PopClient.
type popClient struct {
	*mailbox
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
		m, err := newMailbox(rt, args, popPropertiesPrefix)
		if err != nil {
			return err, nil
		}
		if strings.ContainsAny(m.username+m.password, "\r\n") {
			return emailError(popCredentialsError), nil
		}
		// Like jBallerina, init connects so that bad hosts and credentials fail here.
		s, err := popClient{m}.connect(rt.Platform())
		if err != nil {
			err.Message = popConnectPrefix + err.Message
			return err, nil
		}
		s.quit()
		s.close()
		args[0].(*values.Object).Put(popStateField, m)
		return nil, nil
	}
}

func popReadExtern(rt *runtime.Runtime, strArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		client := popClient{mailboxOf(args[0], popStateField)}
		folder, _ := args[1].(string)
		timeout, _ := args[2].(*decimal.Decimal)
		if client.closed.Load() {
			return emailError(closedError), nil
		}
		if !strings.EqualFold(folder, defaultFolder) {
			return emailError(popFolderError), nil
		}
		platform := rt.Platform()
		return client.receive(ctx, platform, strArrTy, timeout, func() ([]byte, func(bool), *values.Error) {
			data, err := client.retrieveFirst(platform)
			return data, nil, err
		})
	}
}

// retrieveFirst returns the first message in the mailbox and deletes it, or nil when the
// mailbox is empty. The deletion is committed by QUIT in the same session. If DELE or QUIT fails
// the message is still returned, and it stays on the server unless QUIT failed after deleting it.
func (c popClient) retrieveFirst(platform pal.Platform) ([]byte, *values.Error) {
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
func (c popClient) connect(platform pal.Platform) (*popSession, *values.Error) {
	conn, err := c.dial(platform)
	if err != nil {
		return nil, err
	}
	s := newPopSession(platform, conn)
	if _, err := s.reply(); err != nil {
		s.close()
		return nil, popFailure("Connect failed", err)
	}
	if c.startTLS() {
		var tlsErr *values.Error
		if s, tlsErr = c.upgradeSession(platform, s); tlsErr != nil {
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

// upgradeSession upgrades the session with STLS when the server offers it; START_TLS_ALWAYS
// fails when it does not.
func (c popClient) upgradeSession(platform pal.Platform, s *popSession) (*popSession, *values.Error) {
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
	tlsConn, tlsErr := c.upgradeTLS(platform, s.conn)
	if tlsErr != nil {
		return nil, tlsErr
	}
	return newPopSession(platform, tlsConn), nil
}

func newPopSession(platform pal.Platform, conn net.Conn) *popSession {
	return &popSession{conn: conn, text: textproto.NewConn(withTimeout(platform, conn))}
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

// reply reads a status line, returning the text after +OK or a popReplyError for -ERR.
func (s *popSession) reply() (string, error) {
	line, err := s.text.ReadLine()
	if err != nil {
		return "", err
	}
	if text, ok := strings.CutPrefix(line, "+OK"); ok {
		return strings.TrimSpace(text), nil
	}
	return "", &popReplyError{strings.TrimSpace(strings.TrimPrefix(line, "-ERR"))}
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
