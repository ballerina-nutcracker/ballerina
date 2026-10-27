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

package corpus

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
)

func TestExternalSourceEvents(t *testing.T) {
	for _, kind := range []string{"create", "change", "delete"} {
		t.Run(kind, func(t *testing.T) {
			runTranscriptWithCheck(t, "sync/testdata/external-source-"+kind+".json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
				u, _ := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
				diags, gen, ok := compiler.DiagnosticsFor(root)
				want := 0
				if kind == "delete" {
					want = 1
				}
				if !ok || gen <= 1 || len(diags[u]) != want {
					t.Fatalf("external %s must refresh sibling diagnostics: %v, generation=%d, available=%v", kind, diags[u], gen, ok)
				}
				assertFreshEventDiagnostics(t, ws, compiler, root, root)
			})
		})
	}
}

func TestDeletedOpenMember(t *testing.T) {
	runTranscriptWithCheck(t, "sync/testdata/deleted-open-member.json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
		project, _, ok := ws.CurrentProject(root)
		if !ok {
			t.Fatal("project lost")
		}
		if _, member := project.DocumentID("modules/tools/helper.bal"); !member {
			t.Fatal("deleted open member lost across sibling and manifest reloads")
		}
		u, _ := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
		diags, _, ok := compiler.DiagnosticsFor(root)
		if !ok || len(diags[u]) != 0 {
			t.Fatalf("buffer must keep module available: %v", diags[u])
		}
		assertFreshEventDiagnostics(t, ws, compiler, root, root)
	})
}

func TestExternalOpenBuffer(t *testing.T) {
	runTranscriptWithCheck(t, "sync/testdata/external-open-buffer.json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
		u, _ := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
		diags, gen, ok := compiler.DiagnosticsFor(root)
		if !ok || gen <= 1 || len(diags[u]) != 0 {
			t.Fatalf("event must compile authoritative buffer: %v, generation=%d", diags[u], gen)
		}
		assertFreshEventDiagnostics(t, ws, compiler, root, root)
	})
}

func TestBackgroundSourceChange(t *testing.T) {
	runTranscriptWithCheck(t, "sync/testdata/background-source-change.json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
		u, _ := workspace.FileURIFromPath(filepath.Join(root, "helper.bal"))
		text, ok := ws.DocumentText(u)
		if !ok || text != "function renamed() {}\n" {
			t.Fatalf("closed cached root must refresh loaded text: %q, available=%v", text, ok)
		}
		_, snapGen, available := compiler.DiagnosticsFor(root)
		gen, _ := ws.Generation(root)
		if !available || snapGen != gen {
			t.Fatal("background compilation was not refreshed")
		}
	})
}

func TestBackgroundManifestChange(t *testing.T) {
	runTranscriptWithCheck(t, "sync/testdata/background-manifest-change.json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
		if _, _, ok := ws.CurrentProject(root); ok {
			t.Fatal("manifest must invalidate cached roots without open buffers")
		}
		if _, _, ok := compiler.DiagnosticsFor(root); ok {
			t.Fatal("evicted compilation still current")
		}
	})
}

func TestWorkspaceEventEdgeCases(t *testing.T) {
	for _, name := range []string{"deleted-member-close-with-sibling", "deleted-member-final-close", "deleted-default-member-close", "new-buffer-not-member", "absent-unmapped-source-event", "single-file-root-delete", "source-reload-failure", "empty-project-reload-failure", "unrelated-file-event", "unrelated-single-file-not-member", "manifest-events-diagnostics"} {
		t.Run(name, func(t *testing.T) {
			runTranscriptWithCheck(t, "sync/testdata/"+name+".json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
				switch name {
				case "single-file-root-delete", "source-reload-failure", "empty-project-reload-failure":
					key := root
					if name == "single-file-root-delete" {
						key = filepath.Join(root, "main.bal")
					}
					if _, _, ok := ws.CurrentProject(key); ok {
						t.Fatal("failed reload retained project")
					}
					if _, _, ok := compiler.DiagnosticsFor(key); ok {
						t.Fatal("failed reload retained compilation")
					}
				case "absent-unmapped-source-event":
					if gen, ok := ws.Generation(root); !ok || gen <= 1 {
						t.Fatal("absent unmapped event did not refresh cached owner")
					}
					assertFreshEventDiagnostics(t, ws, compiler, root, root)
				case "unrelated-file-event":
					if gen, ok := ws.Generation(root); !ok || gen != 1 {
						t.Fatal("unrelated event refreshed project")
					}
					if _, _, ok := ws.CurrentProject(filepath.Join(root, "other")); ok {
						t.Fatal("unrelated event loaded another project")
					}
				case "unrelated-single-file-not-member":
					key := filepath.Join(root, "nested")
					project, _, ok := ws.CurrentProject(key)
					if !ok {
						t.Fatal("nested package lost")
					}
					if _, member := project.DocumentID("main.bal"); member {
						t.Fatal("unrelated single-file project promoted a buffer-only nested source")
					}
					assertFreshEventDiagnostics(t, ws, compiler, root, key)
				case "manifest-events-diagnostics":
					key := filepath.Join(root, "main.bal")
					if _, _, ok := ws.CurrentProject(key); !ok {
						t.Fatal("manifest deletion failed to rediscover single file")
					}
					if _, _, ok := ws.CurrentProject(root); ok {
						t.Fatal("manifest deletion retained build project")
					}
					assertFreshEventDiagnostics(t, ws, compiler, root, key)
				default:
					project, _, ok := ws.CurrentProject(root)
					if !ok {
						t.Fatal("project lost")
					}
					path := "modules/tools/helper.bal"
					switch name {
					case "new-buffer-not-member":
						path = "new.bal"
					case "deleted-default-member-close":
						path = "helper.bal"
					}
					if _, member := project.DocumentID(path); member {
						t.Fatalf("unexpected synthetic membership for %s", path)
					}
					assertFreshEventDiagnostics(t, ws, compiler, root, root)
				}
			})
		})
	}
}

func TestDeletedMemberStaysAbsentAfterReload(t *testing.T) {
	for _, name := range []string{"deleted-default-member-close", "deleted-member-final-close-reopen"} {
		t.Run(name, func(t *testing.T) {
			runTranscriptWithCheck(t, "sync/testdata/"+name+".json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
				project, gen, ok := ws.CurrentProject(root)
				if !ok {
					t.Fatal("project lost after close and reload")
				}
				if _, loaded := project.DocumentID("after_close_reload.bal"); !loaded {
					t.Fatal("fixture did not reload the root after closing the deleted member")
				}
				if _, member := project.DocumentID("helper.bal"); member {
					t.Fatal("deleted member was rediscovered after its buffer closed")
				}
				helper, _ := workspace.FileURIFromPath(filepath.Join(root, "helper.bal"))
				if _, open := ws.Snapshot(helper); open {
					t.Fatal("deleted member buffer remains open")
				}
				if _, available := ws.DocumentText(helper); available {
					t.Fatal("deleted member text survived close and reload")
				}
				main, _ := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
				diags, snapGen, available := compiler.DiagnosticsFor(root)
				if !available || snapGen != gen || len(diags[main]) != 1 || diags[main][0].Message != "Unknown symbol: helper" {
					t.Fatalf("post-close reload must preserve the current sibling error: %v, generation=%d, want=%d", diags[main], snapGen, gen)
				}
				assertFreshEventDiagnostics(t, ws, compiler, root, root)
			})
		})
	}
}

func assertFreshEventDiagnostics(t *testing.T, ws *workspace.ProjectService, compiler *compile.CompilationService, diskRoot, key string) {
	t.Helper()
	freshRoot := t.TempDir()
	err := fs.WalkDir(os.DirFS(diskRoot), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(freshRoot, name), 0o755)
		}
		content, err := os.ReadFile(filepath.Join(diskRoot, name))
		if err != nil {
			return err
		}
		applyDiskWrite(t, freshRoot, &diskWriteAction{Path: name, Content: string(content)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	project, _, ok := ws.CurrentProject(key)
	if !ok {
		t.Fatal("no current project for fresh comparison")
	}
	open := ws.OpenDocumentsUnder(key)
	if len(open) == 0 {
		return
	}
	for _, uri := range open {
		name, contained := ws.Paths(project).NameOf(uri)
		if !contained {
			continue
		}
		if _, member := project.DocumentID(name); !member {
			continue
		}
		snap, _ := ws.Snapshot(uri)
		rel, err := filepath.Rel(diskRoot, uri.Path())
		if err != nil {
			t.Fatal(err)
		}
		applyDiskWrite(t, freshRoot, &diskWriteAction{Path: filepath.ToSlash(rel), Content: snap.Text})
	}
	bus := event.New()
	defer bus.Close()
	fresh := workspace.New(os.DirFS, bus)
	freshCompiler := compile.New(fresh, bus, compile.WithDebounce(0))
	defer freshCompiler.Shutdown()
	for _, uri := range open {
		rel, _ := filepath.Rel(diskRoot, uri.Path())
		freshURI, _ := workspace.FileURIFromPath(filepath.Join(freshRoot, rel))
		snap, _ := ws.Snapshot(uri)
		if _, err := fresh.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeOpen, URI: freshURI, Text: snap.Text, Version: snap.Version, LanguageID: snap.LanguageID}); err != nil {
			t.Fatal(err)
		}
	}
	freshCompiler.Flush()
	relKey, _ := filepath.Rel(diskRoot, key)
	actual, _, ok := compiler.DiagnosticsFor(key)
	expected, _, freshOK := freshCompiler.DiagnosticsFor(filepath.Join(freshRoot, relKey))
	if !ok || !freshOK {
		t.Fatal("missing current or fresh diagnostics")
	}
	for _, uri := range open {
		rel, _ := filepath.Rel(diskRoot, uri.Path())
		freshURI, _ := workspace.FileURIFromPath(filepath.Join(freshRoot, rel))
		if !reflect.DeepEqual(actual[uri], expected[freshURI]) {
			t.Fatalf("event diagnostics differ from fresh LS compilation for %s: %v != %v", rel, actual[uri], expected[freshURI])
		}
	}
}

func TestCloseDiscardsOverlay(t *testing.T) {
	runTranscriptWithCheck(t, "sync/testdata/close-discards-overlay.json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
		u, _ := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
		diags, _, ok := compiler.DiagnosticsFor(root)
		if !ok || len(diags[u]) != 0 {
			t.Fatalf("close must refresh sibling diagnostics from disk: %v, available=%v", diags[u], ok)
		}
		assertFreshEventDiagnostics(t, ws, compiler, root, root)
	})
}
