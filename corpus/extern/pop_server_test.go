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
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakePopServer is a minimal in-process POP3 server over a shared mailbox. Deletions are applied
// only when a session ends with QUIT, as RFC 1939 requires.
type fakePopServer struct {
	ln        net.Listener
	mode      smtpServerMode
	tlsConfig *tls.Config
	username  string
	password  string
	// hiddenSessions is the number of sessions that see an empty mailbox before the messages
	// become visible.
	hiddenSessions int
	failRetr       bool
	refuseSTLS     bool
	failDele       bool
	dropCAPA       bool

	mu       sync.Mutex
	mailbox  []string
	sessions int
	commands []string
	wg       sync.WaitGroup
}

func newFakePopServer(t *testing.T, mode smtpServerMode, tlsConfig *tls.Config, mailbox []string, configure func(*fakePopServer)) *fakePopServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	s := &fakePopServer{ln: ln, mode: mode, tlsConfig: tlsConfig, username: "bob", password: "secret", mailbox: mailbox}
	if configure != nil {
		configure(s)
	}
	s.wg.Add(1)
	go s.acceptLoop()
	t.Cleanup(s.close)
	return s
}

func (s *fakePopServer) port() string {
	_, port, _ := net.SplitHostPort(s.ln.Addr().String())
	return port
}

func (s *fakePopServer) close() {
	_ = s.ln.Close()
	s.wg.Wait()
}

func (s *fakePopServer) remaining() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.mailbox)
}

func (s *fakePopServer) log() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

func (s *fakePopServer) acceptLoop() {
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

func (s *fakePopServer) record(command string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, command)
}

func (s *fakePopServer) serve(conn net.Conn) {
	isTLS := false
	if s.mode == smtpImplicitTLS {
		conn = tls.Server(conn, s.tlsConfig)
		isTLS = true
	}
	defer func() { _ = conn.Close() }()
	text := textproto.NewConn(conn)
	reply := func(format string, args ...any) { _ = text.PrintfLine(format, args...) }
	reply("+OK fake POP3 ready")
	var visible []string
	deleted := map[int]bool{}
	user := ""
	for {
		line, err := text.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		verb = strings.ToUpper(verb)
		if verb == "PASS" {
			s.record("PASS")
		} else {
			s.record(line)
		}
		switch verb {
		case "CAPA":
			if s.dropCAPA {
				return
			}
			reply("+OK")
			reply("USER")
			if s.mode == smtpStartTLS && !isTLS {
				reply("STLS")
			}
			reply(".")
		case "STLS":
			if s.refuseSTLS {
				reply("-ERR TLS not available")
				continue
			}
			reply("+OK begin TLS")
			conn = tls.Server(conn, s.tlsConfig)
			text = textproto.NewConn(conn)
			isTLS = true
		case "USER":
			user = arg
			reply("+OK")
		case "PASS":
			if user != s.username || arg != s.password {
				reply("-ERR [AUTH] Authentication failed")
				continue
			}
			s.mu.Lock()
			s.sessions++
			if s.sessions > s.hiddenSessions {
				visible = append([]string(nil), s.mailbox...)
			}
			s.mu.Unlock()
			reply("+OK logged in")
		case "STAT":
			reply("+OK %d %d", len(visible), len(strings.Join(visible, "")))
		case "RETR":
			n, _ := strconv.Atoi(arg)
			if s.failRetr {
				reply("-ERR message locked")
				continue
			}
			if n < 1 || n > len(visible) || deleted[n] {
				reply("-ERR no such message")
				continue
			}
			reply("+OK")
			w := text.DotWriter()
			_, _ = fmt.Fprint(w, visible[n-1])
			_ = w.Close()
		case "DELE":
			if s.failDele {
				reply("-ERR maildrop is read-only")
				continue
			}
			n, _ := strconv.Atoi(arg)
			deleted[n] = true
			reply("+OK")
		case "QUIT":
			s.mu.Lock()
			var kept []string
			for i, m := range s.mailbox {
				if i >= len(visible) || !deleted[i+1] {
					kept = append(kept, m)
				}
			}
			s.mailbox = kept
			s.mu.Unlock()
			reply("+OK bye")
			return
		default:
			reply("-ERR unknown command")
		}
	}
}
