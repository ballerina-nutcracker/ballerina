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

package palnative

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

// TestDial_HandshakeTimeout verifies that Dial bounds the TLS handshake by
// TLSConfig.HandshakeTimeout and closes the connection when it expires.
func TestDial_HandshakeTimeout(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	peerClosed := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			peerClosed <- err
			return
		}
		defer func() { _ = conn.Close() }()
		// Never answer the ClientHello; drain until the client closes its side.
		_, err = io.Copy(io.Discard, conn)
		peerClosed <- err
	}()

	start := time.Now()
	_, err = Dial(context.Background(), "tcp", ln.Addr().String(), "", &pal.TLSConfig{
		InsecureSkipVerify: true,
		HandshakeTimeout:   200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected handshake timeout error, got nil")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("handshake timeout not applied: Dial returned after %v", elapsed)
	}

	select {
	case err := <-peerClosed:
		if err != nil {
			t.Fatalf("expected clean EOF on the peer, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client connection was not closed after the handshake failed")
	}
}
