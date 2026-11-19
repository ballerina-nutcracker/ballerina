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
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestSmtpTimeouts shortens the RFC 5321 timeouts, which are minutes long and so cannot be
// exercised from Ballerina corpus tests, and checks that a stalled server fails the send.
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
}

// stalledServer accepts connections, writes greeting and then never responds. It returns its port
// and a function reporting everything the client wrote.
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
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conns = append(conns, conn)
			_, _ = conn.Write([]byte(greeting))
			readers.Go(func() {
				buf := make([]byte, 512)
				for {
					n, err := conn.Read(buf)
					mu.Lock()
					received.Write(buf[:n])
					mu.Unlock()
					if err != nil {
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
		mu.Lock()
		defer mu.Unlock()
		return received.String()
	}
}

func errorMessage(err *values.Error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Message
}

// TestPopTimeouts checks that a stalled POP3 server fails the connection instead of blocking it.
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

func TestSecondsDuration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		seconds string
		want    time.Duration
	}{
		{"1.5", 1500 * time.Millisecond},
		{"-1", 0},
		{"10000000000", math.MaxInt64},
	} {
		seconds, _ := decimal.FromString(tc.seconds)
		if got := secondsDuration(seconds); got != tc.want {
			t.Errorf("secondsDuration(%s) = %v, want %v", tc.seconds, got, tc.want)
		}
	}
	if got := saturatingAdd(time.Hour, math.MaxInt64); got != math.MaxInt64 {
		t.Errorf("saturatingAdd overflowed to %v", got)
	}
}
