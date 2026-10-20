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
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestInitializeTraceFieldAcceptedWithoutError(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{"trace": "verbose"}),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := mustFindResponse(t, responses, `"1"`)
	if resp.Error != nil {
		t.Fatalf("initialize with trace field = %+v, want a normal result", resp)
	}
}

func TestSetTraceIsAcceptAndDiscardNoop(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		notifMessage("$/setTrace", map[string]any{"value": "verbose"}),
		reqMessage(`"2"`, "shutdown", nil),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2 (initialize + shutdown; $/setTrace produces none): %+v", len(responses), responses)
	}
	sh := mustFindResponse(t, responses, `"2"`)
	if string(sh.Result) != "null" {
		t.Fatalf("shutdown response = %+v, want null result", sh)
	}
}
