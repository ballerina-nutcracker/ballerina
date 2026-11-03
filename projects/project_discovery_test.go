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

package projects

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

type discoveryFailureFS struct{ err error }

func (f discoveryFailureFS) Open(string) (fs.File, error) { return nil, f.err }

func TestFindProjectRoot(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"main.bal", "modules/nested/main.bal"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, BallerinaTomlFile), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			got, found, err := FindProjectRoot(os.DirFS, filepath.Join(root, filepath.FromSlash(name)))
			if err != nil || !found || got != root {
				t.Fatalf("got %q, %v, %v; want %q, true, nil", got, found, err, root)
			}
		})
	}
}

func TestFindProjectRootMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	openDir := func(dir string) fs.FS {
		if dir == root {
			return os.DirFS(dir)
		}
		return discoveryFailureFS{fs.ErrNotExist}
	}
	got, found, err := FindProjectRoot(openDir, filepath.Join(root, "main.bal"))
	if err != nil || found || got != "" {
		t.Fatalf("got %q, %v, %v", got, found, err)
	}
}

func TestFindProjectRootFilesystemRoot(t *testing.T) {
	t.Parallel()
	manifest := t.TempDir()
	if err := os.WriteFile(filepath.Join(manifest, BallerinaTomlFile), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.VolumeName(manifest) + string(filepath.Separator)
	openDir := func(dir string) fs.FS {
		if dir == root {
			return os.DirFS(manifest)
		}
		return discoveryFailureFS{fs.ErrNotExist}
	}
	got, found, err := FindProjectRoot(openDir, filepath.Join(root, "nested", "main.bal"))
	if err != nil || !found || got != root {
		t.Fatalf("got %q, %v, %v; want %q", got, found, err, root)
	}
}

func TestFindProjectRootUnreadableAncestor(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{fs.ErrPermission, errors.New("injected I/O failure")} {
		t.Run(failure.Error(), func(t *testing.T) {
			root := t.TempDir()
			blocked := filepath.Join(root, "modules")
			calls := 0
			openDir := func(dir string) fs.FS {
				calls++
				if dir == blocked {
					return discoveryFailureFS{failure}
				}
				if dir == filepath.Join(blocked, "nested") {
					return discoveryFailureFS{fs.ErrNotExist}
				}
				t.Fatalf("searched beyond unreadable ancestor: %s", dir)
				return os.DirFS(root)
			}
			got, found, err := FindProjectRoot(openDir, filepath.Join(blocked, "nested", "main.bal"))
			if !errors.Is(err, failure) || found || got != "" || calls != 2 {
				t.Fatalf("got %q, %v, %v (%d calls)", got, found, err, calls)
			}
		})
	}
}
