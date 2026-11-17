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

package extern_test

import (
	"crypto/tls"
	"encoding/base64"
	"io"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"testing"
)

type smtpServerMode int

const (
	smtpPlain smtpServerMode = iota
	smtpStartTLS
	smtpImplicitTLS
)

// fakeSmtpServer is a minimal in-process SMTP server that records every message it accepts, so
// email:SmtpClient tests can assert on the exact wire content.
type fakeSmtpServer struct {
	ln        net.Listener
	mode      smtpServerMode
	tlsConfig *tls.Config
	// authMechanisms is the AUTH extension parameter; empty means AUTH is not advertised.
	authMechanisms string
	username       string
	password       string
	rejectRcpt     string
	rejectFrom     string
	rejectData     bool
	rejectEhlo     bool
	// legacyAuth advertises the pre-RFC 2554 "AUTH=LOGIN" form instead of the AUTH extension.
	legacyAuth bool
	// dropAfterMail closes the connection once MAIL FROM is accepted.
	dropAfterMail bool
	smtpUTF8      bool
	eightBitMIME  bool

	mu       sync.Mutex
	messages []*smtpReceived
	wg       sync.WaitGroup
}

type smtpReceived struct {
	tls        bool
	authUser   string
	from       string
	mailParams string
	recipients []string
	data       string
}

func newFakeSmtpServer(t *testing.T, mode smtpServerMode, tlsConfig *tls.Config, configure func(*fakeSmtpServer)) *fakeSmtpServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	s := &fakeSmtpServer{ln: ln, mode: mode, tlsConfig: tlsConfig, authMechanisms: "LOGIN PLAIN",
		username: "alice", password: "secret"}
	if configure != nil {
		configure(s)
	}
	s.wg.Add(1)
	go s.acceptLoop()
	t.Cleanup(s.close)
	return s
}

func (s *fakeSmtpServer) port() string {
	_, port, _ := net.SplitHostPort(s.ln.Addr().String())
	return port
}

func (s *fakeSmtpServer) close() {
	_ = s.ln.Close()
	s.wg.Wait()
}

func (s *fakeSmtpServer) received() []*smtpReceived {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*smtpReceived(nil), s.messages...)
}

func (s *fakeSmtpServer) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.serve(conn)
		}()
	}
}

func (s *fakeSmtpServer) serve(conn net.Conn) {
	isTLS := false
	if s.mode == smtpImplicitTLS {
		conn = tls.Server(conn, s.tlsConfig)
		isTLS = true
	}
	defer func() { _ = conn.Close() }()
	text := textproto.NewConn(conn)
	reply := func(format string, args ...any) { _ = text.PrintfLine(format, args...) }
	reply("220 fake ESMTP ready")
	session := &smtpReceived{}
	for {
		line, err := text.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO":
			if s.rejectEhlo {
				reply("502 command not implemented")
				continue
			}
			lines := []string{"fake"}
			switch {
			case s.legacyAuth:
				lines = append(lines, "AUTH=LOGIN")
			case s.authMechanisms != "":
				lines = append(lines, "AUTH "+s.authMechanisms)
			}
			if s.mode == smtpStartTLS && !isTLS {
				lines = append(lines, "STARTTLS")
			}
			if s.smtpUTF8 {
				lines = append(lines, "SMTPUTF8")
			}
			if s.eightBitMIME {
				lines = append(lines, "8BITMIME")
			}
			for i, l := range lines {
				sep := "-"
				if i == len(lines)-1 {
					sep = " "
				}
				reply("250%s%s", sep, l)
			}
		case "HELO":
			reply("250 fake")
		case "STARTTLS":
			reply("220 ready to start TLS")
			conn = tls.Server(conn, s.tlsConfig)
			text = textproto.NewConn(conn)
			isTLS = true
		case "AUTH":
			s.auth(text, arg, session, reply)
		case "MAIL":
			_, path, _ := strings.Cut(arg, ":")
			path, params, _ := strings.Cut(path, " ")
			from := strings.Trim(path, "<>")
			if s.rejectFrom != "" && from == s.rejectFrom {
				reply("550 sender rejected")
				continue
			}
			session.from = from
			session.mailParams = params
			reply("250 ok")
			if s.dropAfterMail {
				return
			}
		case "RCPT":
			_, rcpt, _ := strings.Cut(arg, ":")
			rcpt = strings.TrimSuffix(strings.TrimPrefix(rcpt, "<"), ">")
			if rcpt == s.rejectRcpt {
				reply("550 no such user")
				continue
			}
			session.recipients = append(session.recipients, rcpt)
			reply("250 ok")
		case "DATA":
			reply("354 end data with <CR><LF>.<CR><LF>")
			data, err := io.ReadAll(text.DotReader())
			if err != nil {
				return
			}
			if s.rejectData {
				reply("554 message rejected")
				continue
			}
			session.tls = isTLS
			session.data = string(data)
			s.mu.Lock()
			s.messages = append(s.messages, session)
			s.mu.Unlock()
			session = &smtpReceived{authUser: session.authUser}
			reply("250 queued")
		case "RSET":
			session = &smtpReceived{authUser: session.authUser}
			reply("250 ok")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("500 unknown command")
		}
	}
}

func (s *fakeSmtpServer) auth(text *textproto.Conn, arg string, session *smtpReceived, reply func(string, ...any)) {
	mechanism, initial, _ := strings.Cut(arg, " ")
	decode := func(v string) string {
		b, _ := base64.StdEncoding.DecodeString(v)
		return string(b)
	}
	var user, pass string
	switch strings.ToUpper(mechanism) {
	case "LOGIN":
		reply("334 VXNlcm5hbWU6")
		line, _ := text.ReadLine()
		user = decode(line)
		reply("334 UGFzc3dvcmQ6")
		line, _ = text.ReadLine()
		pass = decode(line)
	case "PLAIN":
		fields := strings.Split(decode(initial), "\x00")
		if len(fields) == 3 {
			user, pass = fields[1], fields[2]
		}
	default:
		reply("504 unrecognized authentication type")
		return
	}
	if user != s.username || pass != s.password {
		reply("535 5.7.8 Authentication failed")
		return
	}
	session.authUser = user
	reply("235 2.7.0 Authentication successful")
}
