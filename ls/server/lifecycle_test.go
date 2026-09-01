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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"errors"
	"testing"
	"time"
)

func TestShutdownThenExitStopsServer(t *testing.T) {
	session := newTestSession(t, false)

	session.sendRequest("2", "shutdown", nil)
	response := session.receive(t)
	if string(response.ID) != `"2"` {
		t.Fatalf("shutdown response ID = %s, want \"2\"", response.ID)
	}
	if string(response.Result) != "null" {
		t.Fatalf("shutdown response result = %s, want null", response.Result)
	}

	session.sendNotification("exit", map[string]any{})
	select {
	case err := <-session.done:
		if err != nil {
			t.Fatalf("Serve after shutdown/exit = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not stop on exit")
	}
}

func TestExitWithoutShutdownFails(t *testing.T) {
	session := newTestSession(t, false)

	session.sendNotification("exit", map[string]any{})
	select {
	case err := <-session.done:
		if !errors.Is(err, errExitedWithoutShutdown) {
			t.Fatalf("Serve after exit without shutdown = %v, want %v", err, errExitedWithoutShutdown)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not stop on exit without shutdown")
	}
}
