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
	"bufio"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestImapTimeouts checks that a stalled IMAP server fails the connection instead of blocking it.
// The IMAP timeout is minutes long and not configurable, so corpus tests cannot reach it.
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

// TestImapPlaintextGreeting checks that START_TLS_AUTO without STARTTLS hands the session a
// greeting that carries the capabilities already read. Without them the session asks for the
// capabilities again, so the result is the same from Ballerina and corpus tests cannot observe it.
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
