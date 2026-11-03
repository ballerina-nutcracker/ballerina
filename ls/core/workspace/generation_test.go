// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
)

func TestGenerationMonotonicAndProjectUpdated(t *testing.T) {
	t.Parallel()
	bus := event.New()
	defer bus.Close()
	svc := New(os.DirFS, bus)
	var updates []event.ProjectUpdatedEvent
	bus.Subscribe([]event.Kind{event.ProjectUpdated}, func(e event.Event) { updates = append(updates, e.(event.ProjectUpdatedEvent)) })
	u, err := FileURIFromPath(filepath.Join(t.TempDir(), "gen.bal"))
	if err != nil {
		t.Fatal(err)
	}
	for version := int32(1); version <= 2; version++ {
		kind := ChangeOpen
		if version == 2 {
			kind = ChangeUpdate
		}
		if _, err := svc.Apply(context.Background(), DocumentChange{Kind: kind, URI: u, Text: "public function main() {}", Version: version}); err != nil {
			t.Fatal(err)
		}
		gen, ok := svc.Generation(u.Path())
		if !ok || gen != uint64(version) {
			t.Fatalf("generation=%d ok=%v", gen, ok)
		}
	}
	if len(updates) != 2 || updates[0].Generation() != 1 || updates[1].Generation() != 2 {
		t.Fatalf("updates=%v", updates)
	}
}

func TestPersistentProjectAcrossUpdates(t *testing.T) {
	t.Parallel()
	bus := event.New()
	defer bus.Close()
	svc := New(os.DirFS, bus)
	u, err := FileURIFromPath(filepath.Join(t.TempDir(), "persist.bal"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeOpen, URI: u, Text: "public function main() {}", Version: 1}); err != nil {
		t.Fatal(err)
	}
	p1, _, ok := svc.CurrentProject(u.Path())
	if !ok || p1 == nil {
		t.Fatal("no project after open")
	}
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeUpdate, URI: u, Text: "public function main() { }", Version: 2}); err != nil {
		t.Fatal(err)
	}
	p2, _, ok := svc.CurrentProject(u.Path())
	if !ok || p1 != p2 {
		t.Fatal("content update replaced the project")
	}
}

func TestOpenDocumentsUnder(t *testing.T) {
	t.Parallel()
	bus := event.New()
	defer bus.Close()
	svc := New(os.DirFS, bus)
	root := t.TempDir()
	for _, name := range []string{"od1.bal", "od2.bal"} {
		u, err := FileURIFromPath(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeOpen, URI: u, Text: "public function main() {}", Version: 1}); err != nil {
			t.Fatal(err)
		}
		if got := svc.OpenDocumentsUnder(u.Path()); len(got) != 1 || got[0] != u {
			t.Fatalf("open documents=%v", got)
		}
	}
}
