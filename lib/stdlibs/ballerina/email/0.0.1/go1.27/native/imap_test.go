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
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestImapTimeouts checks that a stalled IMAP server fails the connection instead of blocking it.
func TestImapTimeouts(t *testing.T) {
	// Not parallel: overrides the package-level receiveTimeout.
	saved := receiveTimeout
	receiveTimeout = 200 * time.Millisecond
	t.Cleanup(func() { receiveTimeout = saved })
	platform := pal.Platform{
		Net:  pal.Net{Dial: palnative.Dial, UpgradeTLS: palnative.UpgradeTLS},
		Time: pal.Time{Now: time.Now},
	}
	for _, tc := range []struct{ security, greeting string }{
		{"START_TLS_NEVER", ""},
		{"START_TLS_NEVER", "* OK ready\r\n"},
		{"START_TLS_AUTO", ""},
		{"START_TLS_AUTO", "* OK ready\r\n"},
	} {
		port, _ := stalledServer(t, tc.greeting)
		client := imapClient{&mailbox{mailServer: mailServer{host: "127.0.0.1", port: port, security: tc.security}}}
		start := time.Now()
		if _, err := client.connect(platform); err == nil || err.Message != imapConnectFailed {
			t.Fatalf("%s, greeting %q: connect error = %v, want %s", tc.security, tc.greeting, errorMessage(err), imapConnectFailed)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("%s, greeting %q: connect took %v, want it bounded by the timeout", tc.security, tc.greeting, elapsed)
		}
	}
}

// TestImapRefusedStartTLS checks that a refused STARTTLS fails the connection in both STARTTLS
// modes, as JavaMail does, before the credentials are sent.
func TestImapRefusedStartTLS(t *testing.T) {
	t.Parallel()
	for _, security := range []string{"START_TLS_AUTO", "START_TLS_ALWAYS"} {
		_, err := negotiate(t, security, "* OK ready\r\n", "* CAPABILITY IMAP4rev1 STARTTLS\r\nS1 OK done\r\n", "S2 NO not now\r\n")
		if err == nil || err.Message != "not now" {
			t.Errorf("%s: error = %v, want not now", security, errorMessage(err))
		}
	}
}

// TestImapCapabilityCase checks that IMAP keywords are matched case-insensitively.
func TestImapCapabilityCase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ greeting, capability string }{
		{"* ok [capability imap4rev1 starttls] ready\r\n", ""},
		{"* ok ready\r\n", "* capability imap4rev1 starttls\r\nS1 ok done\r\n"},
		{"* ok ready\r\n", "S1 OK [CAPABILITY IMAP4rev1 STARTTLS] done\r\n"},
	} {
		replies := []string{"S2 no not now\r\n"}
		if tc.capability != "" {
			replies = append([]string{tc.capability}, replies...)
		}
		_, err := negotiate(t, "START_TLS_ALWAYS", tc.greeting, replies...)
		if err == nil || err.Message != "not now" {
			t.Errorf("greeting %q: error = %v, want STARTTLS sent and refused", tc.greeting, errorMessage(err))
		}
	}
}

// TestImapPlaintextGreeting checks that START_TLS_AUTO without STARTTLS hands the session a
// greeting that carries the capabilities already read.
func TestImapPlaintextGreeting(t *testing.T) {
	t.Parallel()
	greeting, err := negotiate(t, "START_TLS_AUTO", "* OK ready\r\n", "* CAPABILITY IMAP4rev1 AUTH=PLAIN\r\nS1 OK done\r\n")
	if err != nil {
		t.Fatalf("negotiateTLS: %v", errorMessage(err))
	}
	if line, _ := bufio.NewReader(greeting).ReadString('\n'); line != "* OK [CAPABILITY IMAP4rev1 AUTH=PLAIN] ready\r\n" {
		t.Errorf("session starts with %q, want the greeting with the capabilities", line)
	}
}

// TestImapLongLine checks that a line longer than the limit fails the connection.
func TestImapLongLine(t *testing.T) {
	t.Parallel()
	_, err := negotiate(t, "START_TLS_AUTO", "* OK "+strings.Repeat("x", imapMaxLineLength))
	if err == nil || err.Message != imapConnectFailed {
		t.Errorf("error = %v, want %s", errorMessage(err), imapConnectFailed)
	}
}

// negotiate runs negotiateTLS against a server that sends greeting and then answers each
// command with the next reply.
func negotiate(t *testing.T, security, greeting string, replies ...string) (io.Reader, *values.Error) {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	go func() {
		defer func() { _ = server.Close() }()
		r := bufio.NewReader(server)
		if _, err := server.Write([]byte(greeting)); err != nil {
			return
		}
		for _, reply := range replies {
			if _, err := r.ReadString('\n'); err != nil {
				return
			}
			_, _ = server.Write([]byte(reply))
		}
		_, _ = io.Copy(io.Discard, r)
	}()
	c := imapClient{&mailbox{mailServer: mailServer{host: "127.0.0.1", security: security}}}
	_, reader, err := c.negotiateTLS(pal.Platform{Time: pal.Time{Now: time.Now}}, client)
	return reader, err
}
