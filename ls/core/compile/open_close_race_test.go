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

package compile

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

func TestSerialOpenCloseDuringCompilation(t *testing.T) {
	t.Parallel()
	projectsSvc, compiler := newTestServices(t)
	root := t.TempDir()
	text := "public function main() { missing(); }\n"
	for name, content := range map[string]string{
		"Ballerina.toml": "[package]\norg = \"test\"\nname = \"open_close\"\nversion = \"0.1.0\"\n",
		"main.bal":       text,
		"other.bal":      "function other() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	main := fileURI(t, filepath.Join(root, "main.bal"))
	other := fileURI(t, filepath.Join(root, "other.bal"))
	applyOpen(t, projectsSvc, main, text)
	compiler.Flush()
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	first := true
	compiler.compileFn = func(pkg *projects.Package, cyc compileCycle) cycleResult {
		if !first {
			return realCompilePackage(pkg, cyc)
		}
		first = false
		close(entered)
		for {
			select {
			case <-release:
				return cycleResult{}
			default:
				projectsSvc.CurrentProject(root)
				cyc.stale()
				time.Sleep(time.Millisecond)
			}
		}
	}
	if _, err := projectsSvc.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeUpdate, URI: main, Text: text, Version: 2,
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("compilation did not reach barrier")
	}
	done := make(chan error, 1)
	go func() {
		for range 40 {
			for _, kind := range []workspace.ChangeKind{workspace.ChangeOpen, workspace.ChangeClose} {
				if _, err := projectsSvc.Apply(context.Background(), workspace.DocumentChange{
					Kind: kind, URI: other, Text: "function other() {}\n", Version: 1,
				}); err != nil {
					done <- err
					return
				}
			}
		}
		done <- nil
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serial open/close stopped responding during compilation")
	}
	unblock()
	compiler.Flush()
	diags, gen, ok := compiler.DiagnosticsFor(root)
	current, known := projectsSvc.Generation(root)
	if !ok || !known || gen != current || len(diags[main]) != 1 {
		t.Fatalf("diagnostic generation=%d current=%d ok=%v known=%v diagnostics=%v", gen, current, ok, known, diags)
	}
	if _, open := projectsSvc.Snapshot(other); open {
		t.Fatal("last close left the other document open")
	}
}
