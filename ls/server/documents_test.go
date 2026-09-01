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

func TestApplyChangesUTF16(t *testing.T) {
	result, ok := applyChanges("😀x\n", []protocol.TextDocumentContentChangeEvent{
		protocol.NewTextDocumentContentChangeEventTextDocumentContentChangePartial(protocol.TextDocumentContentChangePartial{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 2},
				End:   protocol.Position{Line: 0, Character: 3},
			},
			Text: "y",
		}),
	})
	if !ok || result != "😀y\n" {
		t.Fatalf("applyChanges = %q ok=%v, want %q", result, ok, "😀y\n")
	}
}

func TestApplyChangesRejectsInvalidBatchAtomically(t *testing.T) {
	_, ok := applyChanges("abc", []protocol.TextDocumentContentChangeEvent{
		protocol.NewTextDocumentContentChangeEventTextDocumentContentChangePartial(protocol.TextDocumentContentChangePartial{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 1},
			},
			Text: "x",
		}),
		protocol.NewTextDocumentContentChangeEventTextDocumentContentChangePartial(protocol.TextDocumentContentChangePartial{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 1},
				End:   protocol.Position{Line: 0, Character: 2},
			},
			RangeLength: protocol.NewOptional(uint32(2)),
			Text:        "y",
		}),
	})
	if ok {
		t.Fatal("invalid batch expected rejection, got acceptance")
	}
}

func TestApplyChangesEmptyChangesRejected(t *testing.T) {
	if _, ok := applyChanges("abc", nil); ok {
		t.Fatal("empty changes expected rejection")
	}
}
