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

package workspace

import (
	"context"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func newTestProjectService() *ProjectService {
	platform, _ := palnative.NewPlatform()
	return New(platform, event.New())
}

func fileURI(t *testing.T, raw string) uri.DocumentURI {
	t.Helper()
	u, err := uri.NewFileURI(raw)
	if err != nil {
		t.Fatalf("NewFileURI(%q): %v", raw, err)
	}
	return u
}

func TestApplyOpenStoresDocument(t *testing.T) {
	svc := newTestProjectService()
	u := fileURI(t, "file:///workspace/main.bal")
	snap, err := svc.Apply(context.Background(), DocumentChange{
		Kind:       ChangeOpen,
		URI:        u,
		Text:       "abc",
		Version:    1,
		LanguageID: "ballerina",
	})
	if err != nil {
		t.Fatalf("Apply open: %v", err)
	}
	if snap.Text != "abc" || snap.Version != 1 || snap.LanguageID != "ballerina" {
		t.Fatalf("snapshot = %+v, want text=abc version=1 lang=ballerina", snap)
	}
	got, ok := svc.Snapshot(u)
	if !ok || got.Text != "abc" || got.Version != 1 {
		t.Fatalf("Snapshot = %+v ok=%v, want stored document", got, ok)
	}
}

func TestApplyUpdateChecksVersionMonotonicity(t *testing.T) {
	svc := newTestProjectService()
	u := fileURI(t, "file:///workspace/main.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeOpen, URI: u, Text: "current", Version: 2, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "stale", Version: 2,
	}); err == nil {
		t.Fatal("stale update expected error, got nil")
	}
	got, _ := svc.Snapshot(u)
	if got.Text != "current" || got.Version != 2 {
		t.Fatalf("after stale update: snapshot = %+v, want original", got)
	}
	snap, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "fresh", Version: 3,
	})
	if err != nil {
		t.Fatalf("fresh update: %v", err)
	}
	if snap.Text != "fresh" || snap.Version != 3 {
		t.Fatalf("snapshot = %+v, want text=fresh version=3", snap)
	}
}

func TestApplyUpdateRejectsUnknownDocument(t *testing.T) {
	svc := newTestProjectService()
	u := fileURI(t, "file:///workspace/main.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "x", Version: 1,
	}); err == nil {
		t.Fatal("update of unknown document expected error")
	}
}

func TestApplyCloseRemovesDocument(t *testing.T) {
	svc := newTestProjectService()
	u := fileURI(t, "file:///workspace/main.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeOpen, URI: u, Text: "abc", Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeClose, URI: u,
	}); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, ok := svc.Snapshot(u); ok {
		t.Fatal("closed document was retained")
	}
}

func TestApplyCloseRejectsUnknownDocument(t *testing.T) {
	svc := newTestProjectService()
	u := fileURI(t, "file:///workspace/main.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeClose, URI: u,
	}); err == nil {
		t.Fatal("close of unknown document expected error")
	}
}

func TestApplyClosePreservesLanguageID(t *testing.T) {
	svc := newTestProjectService()
	u := fileURI(t, "file:///workspace/main.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeOpen, URI: u, Text: "abc", Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}
	snap, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "xyz", Version: 2,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if snap.LanguageID != "ballerina" {
		t.Fatalf("languageID = %q, want ballerina", snap.LanguageID)
	}
}

func TestShutdownIsNoOp(t *testing.T) {
	svc := newTestProjectService()
	if err := svc.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
