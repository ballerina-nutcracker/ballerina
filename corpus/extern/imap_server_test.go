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
	"bytes"
	"crypto/tls"
	"log"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// fakeImapServer serves an in-memory mailbox for bob/secret and records the client's commands.
type fakeImapServer struct {
	ln     net.Listener
	user   *imapmemserver.User
	server *imapserver.Server

	mu  sync.Mutex
	log bytes.Buffer
}

// imapMessage is a message to load into a mailbox, with its flags.
type imapMessage struct {
	data  string
	flags []imap.Flag
}

func (s *fakeImapServer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.log.Write(p)
}

// readOnlyMailbox is a mailbox the fake server opens read-only.
const readOnlyMailbox = "ReadOnly"

// readOnlySession opens readOnlyMailbox read-only.
type readOnlySession struct {
	imapserver.Session
}

func (s readOnlySession) Select(name string, options *imap.SelectOptions) (*imap.SelectData, error) {
	data, err := s.Session.Select(name, options)
	if err == nil && name == readOnlyMailbox {
		data.ReadOnly = true
	}
	return data, err
}

// newFakeImapServer starts a server. With tlsConfig, STARTTLS is offered, or the whole connection
// uses TLS when implicitTLS is set; LOGIN is allowed without TLS only when neither is. caps adds
// capabilities to IMAP4rev1.
func newFakeImapServer(t *testing.T, tlsConfig *tls.Config, implicitTLS bool, mailboxes map[string][]imapMessage, caps ...imap.Cap) *fakeImapServer {
	t.Helper()
	s := &fakeImapServer{user: imapmemserver.NewUser("bob", "secret")}
	for name, messages := range mailboxes {
		if err := s.user.Create(name, nil); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		for _, m := range messages {
			s.add(t, name, m)
		}
	}
	mem := imapmemserver.New()
	mem.AddUser(s.user)
	options := &imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return readOnlySession{mem.NewSession()}, nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
		InsecureAuth: tlsConfig == nil || implicitTLS,
		DebugWriter:  s,
		Logger:       log.New(&bytes.Buffer{}, "", 0),
	}
	for _, c := range caps {
		options.Caps[c] = struct{}{}
	}
	if !implicitTLS {
		options.TLSConfig = tlsConfig
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	s.ln = ln
	if implicitTLS {
		ln = tls.NewListener(ln, tlsConfig)
	}
	s.server = imapserver.New(options)
	go func() { _ = s.server.Serve(ln) }()
	t.Cleanup(s.stop)
	return s
}

func (s *fakeImapServer) add(t *testing.T, mailbox string, m imapMessage) {
	t.Helper()
	data := strings.ReplaceAll(m.data, "\n", "\r\n")
	if _, err := s.user.Append(mailbox, literal{strings.NewReader(data)}, &imap.AppendOptions{Flags: m.flags}); err != nil {
		t.Fatalf("appending to %s: %v", mailbox, err)
	}
}

// stop closes the listener and every open connection.
func (s *fakeImapServer) stop() {
	_ = s.server.Close()
}

func (s *fakeImapServer) port() string {
	_, port, _ := net.SplitHostPort(s.ln.Addr().String())
	return port
}

// status returns the number of messages and unseen messages in a mailbox.
func (s *fakeImapServer) status(t *testing.T, mailbox string) (messages, unseen uint32) {
	t.Helper()
	data, err := s.user.Status(mailbox, &imap.StatusOptions{NumMessages: true, NumUnseen: true})
	if err != nil {
		t.Fatalf("status of %s: %v", mailbox, err)
	}
	return *data.NumMessages, *data.NumUnseen
}

// commands returns the commands the clients sent, without their tags.
func (s *fakeImapServer) commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for line := range strings.SplitSeq(s.log.String(), "\r\n") {
		if tag, command, ok := strings.Cut(line, " "); ok && tag != "*" && tag != "+" && !strings.HasPrefix(command, "OK") &&
			!strings.HasPrefix(command, "NO") && !strings.HasPrefix(command, "BAD") {
			out = append(out, command)
		}
	}
	return out
}

type literal struct {
	*strings.Reader
}

func (l literal) Size() int64 { return l.Reader.Size() }
