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
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestSmtpTimeouts shortens the RFC 5321 timeouts, which are minutes long and so cannot be
// exercised from Ballerina corpus tests, and checks that a stalled server fails the send (#9743).
func TestSmtpTimeouts(t *testing.T) {
	// Not parallel: overrides the package-level SMTP timeouts.
	saved := map[*time.Duration]time.Duration{}
	for _, d := range []*time.Duration{&smtpGreetingTimeout, &smtpCommandTimeout} {
		saved[d] = *d
		*d = 200 * time.Millisecond
	}
	t.Cleanup(func() {
		for d, v := range saved {
			*d = v
		}
	})
	platform := pal.Platform{
		Net:  pal.Net{Dial: palnative.Dial, UpgradeTLS: palnative.UpgradeTLS},
		Time: pal.Time{Now: time.Now},
	}

	cases := []struct {
		name     string
		greeting string
		want     string
	}{
		{"no greeting", "", "Could not connect to SMTP host: 127.0.0.1"},
		{"no reply to EHLO", "220 ready\r\n", smtpReadError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			port, received := stalledServer(t, tc.greeting)
			client := &smtpClient{host: "127.0.0.1", port: port, security: "START_TLS_NEVER"}
			start := time.Now()
			err := client.deliver(platform, &outgoingMessage{recipients: []string{"bob@localhost"}})
			if err == nil || !strings.HasPrefix(err.Message, tc.want) {
				t.Fatalf("deliver error = %v, want prefix %q", errorMessage(err), tc.want)
			}
			if _, ok := err.Cause.(*values.Error); !ok {
				t.Errorf("deliver error has no cause")
			}
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Errorf("deliver took %v, want it bounded by the timeout", elapsed)
			}
			// A timed-out EHLO is an I/O failure, so HELO must not be tried and wait again.
			if commands := received(); strings.Contains(commands, "HELO") {
				t.Errorf("client fell back to HELO after an I/O failure: %q", commands)
			}
		})
	}
	// A server that keeps sending reply lines cannot extend the wait for the whole reply.
	t.Run("paced greeting", func(t *testing.T) {
		client := &smtpClient{host: "127.0.0.1", port: pacedServer(t), security: "START_TLS_NEVER"}
		start := time.Now()
		err := client.deliver(platform, &outgoingMessage{recipients: []string{"bob@localhost"}})
		if err == nil || !strings.HasPrefix(err.Message, "Could not connect to SMTP host: 127.0.0.1") {
			t.Fatalf("deliver error = %v, want a connect failure", errorMessage(err))
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("deliver took %v, want it bounded by the greeting timeout", elapsed)
		}
	})
}

// pacedServer sends a greeting continuation line every 50ms for 3 seconds without finishing it.
func pacedServer(t *testing.T) int64 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for range 60 {
			if _, err := conn.Write([]byte("220-wait\r\n")); err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	})
	t.Cleanup(func() {
		_ = ln.Close()
		wg.Wait()
	})
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	n, _ := strconv.ParseInt(port, 10, 64)
	return n
}

// stalledServer accepts connections, writes greeting and then never responds. It returns its port
// and a function that waits for the client to close its connection and reports everything it wrote.
func stalledServer(t *testing.T, greeting string) (int64, func() string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	var (
		mu       sync.Mutex
		received strings.Builder
		conns    []net.Conn
		readers  sync.WaitGroup
	)
	done := make(chan struct{})
	closed := make(chan struct{}, 16)
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conns = append(conns, conn)
			// Outlasts every client timeout these tests use; when it expires the reader closes the
			// connection, so a client without a read deadline fails its elapsed-time check instead of hanging.
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
			_, _ = conn.Write([]byte(greeting))
			readers.Go(func() {
				buf := make([]byte, 512)
				for {
					n, err := conn.Read(buf)
					mu.Lock()
					received.Write(buf[:n])
					mu.Unlock()
					if err != nil {
						_ = conn.Close()
						closed <- struct{}{}
						return
					}
				}
			})
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
		for _, c := range conns {
			_ = c.Close()
		}
		readers.Wait()
	})
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	n, _ := strconv.ParseInt(port, 10, 64)
	return n, func() string {
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Fatalf("client did not close its connection")
		}
		mu.Lock()
		defer mu.Unlock()
		return received.String()
	}
}

func causeMessage(err *values.Error) string {
	if cause, ok := err.Cause.(*values.Error); ok {
		return cause.Message
	}
	return "<no cause>"
}

func errorMessage(err *values.Error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Message
}

// TestSmtpStartTLSDeadline checks that the STARTTLS reply's read deadline does not cut short the
// TLS handshake, which has its own command timeout. Like TestSmtpTimeouts, it shortens timeouts
// that are minutes long, so corpus tests cannot reach it.
func TestSmtpStartTLSDeadline(t *testing.T) {
	// Not parallel: overrides the package-level SMTP timeouts.
	saved := smtpCommandTimeout
	smtpCommandTimeout = 600 * time.Millisecond
	t.Cleanup(func() { smtpCommandTimeout = saved })
	platform := pal.Platform{
		Net:  pal.Net{Dial: palnative.Dial, UpgradeTLS: palnative.UpgradeTLS},
		Time: pal.Time{Now: time.Now},
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 512)
		_, _ = conn.Write([]byte("220 ready\r\n"))
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("250-fake\r\n250 STARTTLS\r\n"))
		_, _ = conn.Read(buf)
		// The reply comes late in the STARTTLS window, and the handshake then stalls past its end.
		time.Sleep(400 * time.Millisecond)
		_, _ = conn.Write([]byte("220 go ahead\r\n"))
		time.Sleep(400 * time.Millisecond)
	})
	t.Cleanup(func() {
		_ = ln.Close()
		wg.Wait()
	})
	_, portText, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.ParseInt(portText, 10, 64)
	client := &smtpClient{host: "127.0.0.1", port: port, security: "START_TLS_ALWAYS", tls: &pal.TLSConfig{}}
	deliverErr := client.deliver(platform, &outgoingMessage{recipients: []string{"bob@localhost"}})
	if deliverErr == nil || deliverErr.Message != "Could not convert socket to TLS" {
		t.Fatalf("deliver error = %v, want a TLS conversion failure", errorMessage(deliverErr))
	}
	// The server closes the connection mid-handshake, within the handshake's own budget.
	if cause, ok := deliverErr.Cause.(*values.Error); !ok || strings.Contains(cause.Message, "timeout") {
		t.Errorf("handshake failed with %q, want the server's close, not the STARTTLS read deadline", causeMessage(deliverErr))
	}
}

// TestPopTimeouts checks that a stalled POP3 server fails the connection instead of blocking it.
// The POP3 timeout is minutes long and not configurable, so corpus tests cannot reach it.
func TestPopTimeouts(t *testing.T) {
	// Not parallel: overrides the package-level popTimeout.
	saved := popTimeout
	popTimeout = 200 * time.Millisecond
	t.Cleanup(func() { popTimeout = saved })
	platform := pal.Platform{
		Net:  pal.Net{Dial: palnative.Dial, UpgradeTLS: palnative.UpgradeTLS},
		Time: pal.Time{Now: time.Now},
	}
	for _, greeting := range []string{"", "+OK ready\r\n"} {
		port, _ := stalledServer(t, greeting)
		client := &popClient{host: "127.0.0.1", port: port, security: "START_TLS_NEVER"}
		start := time.Now()
		_, err := client.connect(platform)
		if err == nil || err.Message != "Connect failed" {
			t.Fatalf("greeting %q: connect error = %v, want Connect failed", greeting, errorMessage(err))
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("greeting %q: connect took %v, want it bounded by the timeout", greeting, elapsed)
		}
	}
}
