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
	"bytes"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestLegacyReleaseRequestIgnored(t *testing.T) {
	srv, transport := newBatchServer(t)
	defer srv.cancelAndDrainTrackedRequests()
	var input bytes.Buffer
	for _, message := range []protocol.Message{
		reqMessage(`1`, "initialize", map[string]any{}),
		reqMessage(`2`, "$test/blockRequest", map[string]any{}),
		notifMessage("$pal/releaseRequest", map[string]any{"id": 2}),
	} {
		if err := protocol.WriteMessage(&input, message); err != nil {
			t.Fatal(err)
		}
	}
	transport.reader = bytes.NewReader(input.Bytes())
	if err := srv.Serve(); err != nil {
		t.Fatal(err)
	}
	entry := srv.registry.lookup(`2`)
	if entry == nil || entry.release == nil {
		t.Fatal("test blockRequest did not remain blocked")
	}
	select {
	case <-entry.release:
		t.Fatal("legacy releaseRequest released the blocked request")
	default:
	}
}
