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

// fakePopServer is a minimal in-process POP3 server over a shared mailbox. As RFC 1939 requires,
// a session holds the maildrop exclusively from PASS to its end, and deletions are applied only
// when it ends with QUIT.
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
	// garbleCAPA answers CAPA with a line that is neither +OK nor -ERR.
	garbleCAPA bool

	mu       sync.Mutex
	mailbox  []string
	sessions int
	commands []string
	locked   bool
	conns    map[net.Conn]struct{}
	closed   bool
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

// close stops accepting and closes open sessions, so handlers blocked on a client return.
func (s *fakePopServer) close() {
	_ = s.ln.Close()
	s.mu.Lock()
	s.closed = true
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.mu.Unlock()
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
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		if s.conns == nil {
			s.conns = map[net.Conn]struct{}{}
		}
		s.conns[conn] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				s.mu.Lock()
				delete(s.conns, conn)
				s.mu.Unlock()
			}()
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
	holdsLock := false
	defer func() {
		if holdsLock {
			s.mu.Lock()
			s.locked = false
			s.mu.Unlock()
		}
	}()
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
			if s.garbleCAPA {
				reply("garbage")
				continue
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
			if s.locked {
				s.mu.Unlock()
				reply("-ERR [IN-USE] maildrop locked")
				continue
			}
			s.locked, holdsLock = true, true
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
			// RFC 1939 section 6 releases the maildrop before the reply, so the client's next session
			// cannot find it still locked.
			if holdsLock {
				s.locked, holdsLock = false, false
			}
			s.mu.Unlock()
			reply("+OK bye")
			return
		default:
			reply("-ERR unknown command")
		}
	}
}
