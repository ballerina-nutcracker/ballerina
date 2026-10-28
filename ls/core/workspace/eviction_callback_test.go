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
	"reflect"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
)

func TestEvictionCallbackReadsCompletedIndex(t *testing.T) {
	t.Parallel()
	bus := event.New()
	defer bus.Close()
	svc := New(os.DirFS, bus, WithMaxProjects(1))
	root := t.TempDir()
	first, _ := FileURIFromPath(filepath.Join(root, "first.bal"))
	second, _ := FileURIFromPath(filepath.Join(root, "second.bal"))
	open := func(uri DocumentURI) error {
		_, err := svc.Apply(context.Background(), DocumentChange{
			Kind: ChangeOpen, URI: uri, Text: "public function main() {}\n", Version: 1,
		})
		return err
	}
	if err := open(first); err != nil {
		t.Fatal(err)
	}
	var order []event.Kind
	var evictedRoot string
	var oldPresent, newPresent bool
	var newGen uint64
	bus.Subscribe([]event.Kind{event.ProjectEvicted, event.ProjectRegistered, event.ProjectUpdated}, func(e event.Event) {
		order = append(order, e.Kind())
		if e.Kind() == event.ProjectEvicted {
			evictedRoot = e.SourceRoot()
			_, oldPresent = svc.Generation(first.Path())
			newGen, newPresent = svc.Generation(second.Path())
		}
	})
	done := make(chan error, 1)
	go func() { done <- open(second) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("opening second project blocked in inline eviction callback")
	}
	if evictedRoot != first.Path() || oldPresent || !newPresent || newGen == 0 {
		t.Fatalf("callback state: evicted=%q old=%v new=%v generation=%d", evictedRoot, oldPresent, newPresent, newGen)
	}
	if !reflect.DeepEqual(order, []event.Kind{event.ProjectEvicted, event.ProjectRegistered, event.ProjectUpdated}) {
		t.Fatalf("event order = %v", order)
	}
}
