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
	"bufio"
	"net"
	"sync"
	"testing"
)

// newScriptedServer accepts connections, writes greeting to each and answers each line the client
// sends with the next reply. It closes the connection after the last reply, so a client that reads
// or sends more fails at once instead of waiting for a timeout. It returns the port.
func newScriptedServer(t *testing.T, greeting string, replies ...string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Go(func() {
				defer func() { _ = conn.Close() }()
				r := bufio.NewReader(conn)
				if _, err := conn.Write([]byte(greeting)); err != nil {
					return
				}
				for _, reply := range replies {
					if _, err := r.ReadString('\n'); err != nil {
						return
					}
					if _, err := conn.Write([]byte(reply)); err != nil {
						return
					}
				}
			})
		}
	})
	t.Cleanup(func() {
		_ = ln.Close()
		wg.Wait()
	})
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}
