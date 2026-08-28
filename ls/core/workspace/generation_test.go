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

// TestGenerationMonotonicAndProjectUpdated verifies that Apply (open then
// update) increments the per-source-root generation exactly once each and
// publishes ProjectUpdated inline (WM-E4) carrying the new generation.
func TestGenerationMonotonicAndProjectUpdated(t *testing.T) {
	bus := event.New()
	defer bus.Close()
	platform, _ := palnative.NewPlatform()
	svc := New(platform, bus)

	var updates []event.ProjectUpdatedEvent
	bus.Subscribe([]event.Kind{event.ProjectUpdated}, func(e event.Event) {
		updates = append(updates, e.(event.ProjectUpdatedEvent))
	})

	u := fileURI(t, "file:///workspace/gen.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeOpen, URI: u, Text: "import ballerina/io;\n", Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}
	g1, ok := svc.Generation("/workspace")
	if !ok || g1 != 1 {
		t.Fatalf("after open: generation = %d ok=%v, want 1", g1, ok)
	}

	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "import ballerina/io;\nfunction foo(){}", Version: 2,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	g2, ok := svc.Generation("/workspace")
	if !ok || g2 != 2 {
		t.Fatalf("after update: generation = %d ok=%v, want 2", g2, ok)
	}

	if len(updates) != 2 {
		t.Fatalf("ProjectUpdated published %d times, want 2", len(updates))
	}
	if updates[0].Generation() != 1 || updates[1].Generation() != 2 {
		t.Fatalf("ProjectUpdated generations = [%d,%d], want [1,2]",
			updates[0].Generation(), updates[1].Generation())
	}
}

// TestPersistentProjectAcrossUpdates verifies the modifier-chain publication
// model (branch 1): a content update reuses the SAME project (and thus the
// persistent per-source-root CompilerEnvironment) rather than reloading a
// fresh project per change.
func TestPersistentProjectAcrossUpdates(t *testing.T) {
	bus := event.New()
	defer bus.Close()
	platform, _ := palnative.NewPlatform()
	svc := New(platform, bus)

	u := fileURI(t, "file:///workspace/persist.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeOpen, URI: u, Text: "import ballerina/io;\n", Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}
	p1, _, ok := svc.CurrentProject("/workspace")
	if !ok || p1 == nil {
		t.Fatal("no project after open")
	}

	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "import ballerina/io;\nfunction foo(){}", Version: 2,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	p2, _, ok := svc.CurrentProject("/workspace")
	if !ok || p2 == nil {
		t.Fatal("no project after update")
	}
	if p1 != p2 {
		t.Fatalf("modifier chain should reuse the same project; got different pointers %p vs %p", p1, p2)
	}
}

// TestOpenDocumentsUnder verifies the open-document enumeration under a source
// root used by the server CE subscriber (branch 10).
func TestOpenDocumentsUnder(t *testing.T) {
	bus := event.New()
	defer bus.Close()
	platform, _ := palnative.NewPlatform()
	svc := New(platform, bus)

	u1 := fileURI(t, "file:///workspace/od1.bal")
	u2 := fileURI(t, "file:///workspace/od2.bal")
	uOther := fileURI(t, "file:///other/od3.bal")
	for _, u := range []uri.DocumentURI{u1, u2, uOther} {
		if _, err := svc.Apply(context.Background(), DocumentChange{
			Kind: ChangeOpen, URI: u, Text: "import ballerina/io;\n", Version: 1, LanguageID: "ballerina",
		}); err != nil {
			t.Fatalf("open %s: %v", u, err)
		}
	}
	got := svc.OpenDocumentsUnder("/workspace")
	wantCount := 2
	if len(got) != wantCount {
		t.Fatalf("OpenDocumentsUnder(/workspace) = %d, want %d", len(got), wantCount)
	}
}
