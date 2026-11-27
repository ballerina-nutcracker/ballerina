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
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	securitySSL            = "SSL"
	securityStartTLSAuto   = "START_TLS_AUTO"
	securityStartTLSAlways = "START_TLS_ALWAYS"

	smtpStateField = "$smtp"
	// smtpMaxLineSize and smtpMaxReplySize bound a reply well above RFC 5321's 512-octet line limit.
	smtpMaxLineSize  = 4096
	smtpMaxReplySize = 64 * 1024
	smtpSendError    = "Error while sending the message to SMTP server : "
	smtpReadError    = "Exception reading response"
)

// SMTP client timeouts from RFC 5321 section 4.5.3.2; each bounds one wait on the server.
var (
	smtpGreetingTimeout = 5 * time.Minute
	smtpCommandTimeout  = 5 * time.Minute
	smtpDataInitTimeout = 2 * time.Minute
	smtpDataTimeout     = 3 * time.Minute
	smtpDataEndTimeout  = 10 * time.Minute
)

// smtpClient is the connection configuration stored on an email:SmtpClient. Like jBallerina, the
// client connects per send, so it holds no connection and needs no locking.
type smtpClient struct {
	host     string
	port     int64
	security string
	tls      *pal.TLSConfig
	username string
	password string
	auth     bool
}

func initSmtpClientExtern(rt *runtime.Runtime, withAuth bool) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self, _ := args[0].(*values.Object)
		client := &smtpClient{auth: withAuth}
		client.host, _ = args[1].(string)
		config, _ := args[len(args)-1].(*values.Map)
		if withAuth {
			client.username, _ = args[2].(string)
			client.password, _ = args[3].(string)
		}
		port, _ := config.Get("port")
		client.port, _ = port.(int64)
		client.security, _ = stringField(config, "security")
		tlsCfg, err := tlsConfigFromSecureSocket(rt, config)
		if err != nil {
			return err, nil
		}
		client.tls = tlsCfg
		self.Put(smtpStateField, client)
		return nil, nil
	}
}

func sendExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self, _ := args[0].(*values.Object)
		email, _ := args[1].(*values.Map)
		state, _ := self.Get(smtpStateField)
		client, _ := state.(*smtpClient)
		msg, err := buildMessage(ctx, rt, client.username, email)
		if err != nil {
			return err, nil
		}
		if err := client.deliver(rt.Platform(), msg); err != nil {
			return err, nil
		}
		return nil, nil
	}
}

// smtpFailure reports a negative SMTP reply with its code, as JavaMail does, and any other failure
// as JavaMail's read error with the underlying error as the cause.
func smtpFailure(err error) *values.Error {
	if protoErr, ok := errors.AsType[*textproto.Error](err); ok {
		return emailError(replyText(protoErr))
	}
	return emailErrorWithCause(smtpReadError, err)
}

// sendFailure reports a rejected MAIL, DATA or message as JavaMail's SendFailedException with no
// invalid addresses, hence the trailing space.
func sendFailure(err error) *values.Error {
	if protoErr, ok := errors.AsType[*textproto.Error](err); ok {
		return emailError(smtpSendError + replyText(protoErr) + " ")
	}
	return smtpFailure(err)
}

func replyText(err *textproto.Error) string {
	return strconv.Itoa(err.Code) + " " + err.Msg
}

// deadlineConn bounds every write by the session's current timeout; reads are bounded per reply.
type deadlineConn struct {
	net.Conn
	now     func() time.Time
	timeout *time.Duration
}

func (c deadlineConn) Write(p []byte) (int, error) {
	if err := c.SetWriteDeadline(c.now().Add(*c.timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}

type smtpSession struct {
	conn    net.Conn
	text    *textproto.Conn
	ext     map[string]string
	now     func() time.Time
	timeout time.Duration
}

func newSmtpSession(platform pal.Platform, conn net.Conn) *smtpSession {
	s := &smtpSession{conn: conn, now: platform.Time.Now, timeout: smtpGreetingTimeout}
	s.text = textproto.NewConn(deadlineConn{Conn: conn, now: platform.Time.Now, timeout: &s.timeout})
	return s
}

// readReply reads one reply, which must complete within timeout however the server paces it.
func (s *smtpSession) readReply(timeout time.Duration, expectCode int) (string, error) {
	if err := s.conn.SetReadDeadline(s.now().Add(timeout)); err != nil {
		return "", err
	}
	return s.readResponse(expectCode)
}

// readResponse reads a reply as textproto's ReadResponse does, also accepting a line that is only
// the reply code, which RFC 5321 section 4.2 allows and textproto rejects.
func (s *smtpSession) readResponse(expectCode int) (string, error) {
	var lines []string
	code, size := 0, 0
	for {
		line, err := s.readLine()
		if err != nil {
			return "", err
		}
		if size += len(line); size > smtpMaxReplySize {
			return "", textproto.ProtocolError("response too long")
		}
		lineCode, convErr := strconv.Atoi(line[:min(len(line), 3)])
		if len(line) < 3 || (len(line) > 3 && line[3] != ' ' && line[3] != '-') || convErr != nil || lineCode < 100 {
			return "", textproto.ProtocolError("invalid response: " + line)
		}
		if code != 0 && lineCode != code {
			return "", textproto.ProtocolError("inconsistent codes in multi-line response: " + line)
		}
		code = lineCode
		lines = append(lines, line[min(len(line), 4):])
		if len(line) == 3 || line[3] == ' ' {
			break
		}
	}
	msg := strings.Join(lines, "\n")
	if 1 <= expectCode && expectCode < 10 && code/100 != expectCode || 10 <= expectCode && expectCode < 100 && code/10 != expectCode ||
		100 <= expectCode && code != expectCode {
		return msg, &textproto.Error{Code: code, Msg: msg}
	}
	return msg, nil
}

// readLine reads one reply line of at most smtpMaxLineSize bytes, so a faulty server cannot make
// the client buffer an unbounded line.
func (s *smtpSession) readLine() (string, error) {
	var line []byte
	for {
		chunk, more, err := s.text.R.ReadLine()
		if err != nil {
			return "", err
		}
		if len(line)+len(chunk) > smtpMaxLineSize {
			return "", textproto.ProtocolError("response line too long")
		}
		line = append(line, chunk...)
		if !more {
			return string(line), nil
		}
	}
}

func (s *smtpSession) close() {
	_ = s.text.Close()
}

func (s *smtpSession) cmd(timeout time.Duration, expectCode int, format string, args ...any) (string, error) {
	s.timeout = timeout
	id, err := s.text.Cmd(format, args...)
	if err != nil {
		return "", err
	}
	s.text.StartResponse(id)
	defer s.text.EndResponse(id)
	return s.readReply(timeout, expectCode)
}

// hello sends EHLO, falling back to HELO when the server rejects it, and records the advertised
// extensions.
func (s *smtpSession) hello() error {
	name := "localhost"
	if addr, ok := s.conn.LocalAddr().(*net.TCPAddr); ok {
		if addr.IP.To4() != nil {
			name = "[" + addr.IP.String() + "]"
		} else {
			name = "[IPv6:" + addr.IP.String() + "]"
		}
	}
	s.ext = map[string]string{}
	msg, err := s.cmd(smtpCommandTimeout, 250, "EHLO %s", name)
	if err != nil {
		// 421 closes the session (RFC 5321 section 3.8), so HELO is not tried.
		if protoErr, rejected := errors.AsType[*textproto.Error](err); !rejected || protoErr.Code == 421 {
			return err
		}
		_, err = s.cmd(smtpCommandTimeout, 250, "HELO %s", name)
		return err
	}
	for _, line := range strings.Split(msg, "\n")[1:] {
		keyword, params, _ := strings.Cut(line, " ")
		s.ext[strings.ToUpper(keyword)] = params
	}
	return nil
}

func (c *smtpClient) address() string {
	return net.JoinHostPort(c.host, strconv.FormatInt(c.port, 10))
}

func (c *smtpClient) connectError(cause error) *values.Error {
	return emailErrorWithCause(fmt.Sprintf("Could not connect to SMTP host: %s, port: %d", c.host, c.port), cause)
}

func (c *smtpClient) connect(platform pal.Platform) (*smtpSession, *values.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), smtpGreetingTimeout)
	defer cancel()
	var implicitTLS *pal.TLSConfig
	if c.security == securitySSL {
		implicitTLS = c.tls
	}
	conn, err := platform.Net.Dial(ctx, "tcp", c.address(), "", implicitTLS)
	if err != nil {
		return nil, c.connectError(err)
	}
	s := newSmtpSession(platform, conn)
	if _, err := s.readReply(smtpGreetingTimeout, 220); err != nil {
		s.close()
		return nil, c.connectError(err)
	}
	if err := s.hello(); err != nil {
		s.close()
		return nil, smtpFailure(err)
	}
	if c.security != securityStartTLSAuto && c.security != securityStartTLSAlways {
		return s, nil
	}
	if _, ok := s.ext["STARTTLS"]; !ok {
		if c.security == securityStartTLSAlways {
			s.close()
			return nil, emailError("STARTTLS is required but host does not support STARTTLS")
		}
		return s, nil
	}
	if _, err := s.cmd(smtpCommandTimeout, 220, "STARTTLS"); err != nil {
		s.close()
		return nil, smtpFailure(err)
	}
	handshakeCtx, cancelHandshake := context.WithTimeout(context.Background(), smtpCommandTimeout)
	defer cancelHandshake()
	// The STARTTLS reply's read deadline must not cut short the handshake's own budget.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		s.close()
		return nil, emailErrorWithCause("Could not convert socket to TLS", err)
	}
	tlsConn, err := platform.Net.UpgradeTLS(handshakeCtx, conn, c.address(), c.tls)
	if err != nil {
		return nil, emailErrorWithCause("Could not convert socket to TLS", err)
	}
	s = newSmtpSession(platform, tlsConn)
	if err := s.hello(); err != nil {
		s.close()
		return nil, smtpFailure(err)
	}
	return s, nil
}

// authMechanisms returns the advertised SASL mechanisms, including the legacy "AUTH=LOGIN" form
// that JavaMail also recognizes.
func (s *smtpSession) authMechanisms() ([]string, bool) {
	params, ok := s.ext["AUTH"]
	if legacy, found := s.ext["AUTH=LOGIN"]; found {
		params, ok = "LOGIN "+legacy+" "+params, true
	}
	return strings.Fields(strings.ToUpper(params)), ok
}

// authenticate prefers LOGIN over PLAIN, as JavaMail's default mechanism order does. A server
// that does not advertise AUTH is used without authentication, also like JavaMail.
func (c *smtpClient) authenticate(s *smtpSession) *values.Error {
	mechanisms, ok := s.authMechanisms()
	if !c.auth || !ok {
		return nil
	}
	encode := base64.StdEncoding.EncodeToString
	var err error
	switch {
	case slices.Contains(mechanisms, "LOGIN"):
		if _, err = s.cmd(smtpCommandTimeout, 334, "AUTH LOGIN"); err == nil {
			if _, err = s.cmd(smtpCommandTimeout, 334, "%s", encode([]byte(c.username))); err == nil {
				_, err = s.cmd(smtpCommandTimeout, 235, "%s", encode([]byte(c.password)))
			}
		}
	case slices.Contains(mechanisms, "PLAIN"):
		_, err = s.cmd(smtpCommandTimeout, 235, "AUTH PLAIN %s", encode([]byte("\x00"+c.username+"\x00"+c.password)))
	default:
		return emailError("No authentication mechanisms supported by both server and client")
	}
	if err != nil {
		return smtpFailure(err)
	}
	return nil
}

func (c *smtpClient) deliver(platform pal.Platform, msg *outgoingMessage) *values.Error {
	if len(msg.recipients) == 0 {
		return emailError(smtpSendError + "No recipient addresses ")
	}
	s, err := c.connect(platform)
	if err != nil {
		return err
	}
	defer s.close()
	if err := c.authenticate(s); err != nil {
		return err
	}
	mailParams := ""
	if msg.utf8 || msg.utf8MessageID {
		if _, ok := s.ext["SMTPUTF8"]; !ok {
			if msg.utf8 {
				return emailError("SMTPUTF8 is required for non-ASCII addresses but host does not support SMTPUTF8")
			}
			return emailError("SMTPUTF8 is required for non-ASCII message IDs but host does not support SMTPUTF8")
		}
		mailParams = " SMTPUTF8"
	}
	data := msg.data
	if msg.data7 != nil {
		if _, ok := s.ext["8BITMIME"]; ok {
			mailParams += " BODY=8BITMIME"
		} else {
			data = msg.data7
		}
	}
	if _, err := s.cmd(smtpCommandTimeout, 250, "MAIL FROM:<%s>%s", msg.envelopeFrom, mailParams); err != nil {
		return sendFailure(err)
	}
	var invalid []string
	for _, rcpt := range msg.recipients {
		if _, err := s.cmd(smtpCommandTimeout, 25, "RCPT TO:<%s>", rcpt); err != nil {
			protoErr, rejected := errors.AsType[*textproto.Error](err)
			if !rejected {
				return smtpFailure(err)
			}
			// 421 closes the session (RFC 5321 section 3.8), so it fails the send, not the recipient.
			if protoErr.Code == 421 {
				return sendFailure(err)
			}
			invalid = append(invalid, rcpt)
		}
	}
	if len(invalid) > 0 {
		_, _ = s.cmd(smtpCommandTimeout, 250, "RSET")
		return emailError(smtpSendError + "Invalid Addresses " + strings.Join(invalid, ","))
	}
	if _, err := s.cmd(smtpDataInitTimeout, 354, "DATA"); err != nil {
		return sendFailure(err)
	}
	s.timeout = smtpDataTimeout
	w := s.text.DotWriter()
	if _, err := w.Write(data); err != nil {
		return emailErrorWithCause("IOException while sending message", err)
	}
	if err := w.Close(); err != nil {
		return emailErrorWithCause("IOException while sending message", err)
	}
	if _, err := s.readReply(smtpDataEndTimeout, 250); err != nil {
		return sendFailure(err)
	}
	_, _ = s.cmd(smtpCommandTimeout, 221, "QUIT")
	return nil
}
