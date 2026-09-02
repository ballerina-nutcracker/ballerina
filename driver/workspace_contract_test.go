// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func TestFindWorkspaceSourcesPreservesManifestOrderAndExactMembership(t *testing.T) {
	fsys := workspaceFS()
	cx := newDriverContext()
	workspace, err := driver.FindWorkspaceSources(cx, fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	if cx.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", cx.Diagnostics())
	}
	if len(workspace.Members) != 2 {
		t.Fatalf("member count = %d, want 2", len(workspace.Members))
	}
	if workspace.Members[0].Root != "lib" || workspace.Members[1].Root != "apps/main" {
		t.Fatalf("member order = %q, %q", workspace.Members[0].Root, workspace.Members[1].Root)
	}
	if !workspace.Resolution.Offline {
		t.Fatal("workspace did not inherit resolution options from its first member")
	}
	if member, ok := workspace.Member("apps/main"); !ok || member.Sources.Descriptor.Name != "app" {
		t.Fatalf("exact member lookup failed: %#v, %v", member, ok)
	}
	if _, ok := workspace.Member("apps/main/subdir"); ok {
		t.Fatal("nested directory was accepted as a workspace member")
	}
}

func TestWorkspaceResolverPrecedesFallbackAndPreservesTerminalOwnership(t *testing.T) {
	fsys := workspaceFS()
	cx := newDriverContext()
	workspace, err := driver.FindWorkspaceSources(cx, fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	fallback := &recordingWorkspaceFallback{}
	resolver := driver.NewWorkspaceDependencyResolver(fsys, workspace, fallback, workspace.Resolution)

	resolvedFS, root, sources, _, err := resolver.Resolve(context.Background(), driver.DependencyRequest{
		Descriptor: driver.PackageDescriptor{Org: "testorg", Name: "lib"}, ModuleName: "lib",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolvedFS == nil || root != "lib" || sources == nil || fallback.calls != 0 {
		t.Fatalf("workspace resolution = (%v, %q, %#v), fallback calls = %d", resolvedFS, root, sources, fallback.calls)
	}

	resolvedFS, root, sources, _, err = resolver.Resolve(context.Background(), driver.DependencyRequest{
		Descriptor: driver.PackageDescriptor{Org: "testorg", Name: "lib"}, ModuleName: "lib.missing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolvedFS != nil || root != "" || sources != nil || fallback.calls != 0 {
		t.Fatalf("owned missing module fell through: (%v, %q, %#v), calls = %d", resolvedFS, root, sources, fallback.calls)
	}

	_, _, _, _, err = resolver.Resolve(context.Background(), driver.DependencyRequest{
		Descriptor: driver.PackageDescriptor{Org: "testorg", Name: "lib", Version: "2.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fallback.calls != 1 || !fallback.last.Resolution.Offline {
		t.Fatalf("version mismatch did not use fallback with workspace options: calls = %d, request = %#v", fallback.calls, fallback.last)
	}

	_, _, _, _, err = resolver.Resolve(context.Background(), driver.DependencyRequest{
		Descriptor: driver.PackageDescriptor{Org: "testorg", Name: "lib", Version: "1.0.0"}, Repository: "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fallback.calls != 2 || fallback.last.Repository != "local" {
		t.Fatalf("explicit repository did not bypass workspace: calls = %d, request = %#v", fallback.calls, fallback.last)
	}
}

type recordingWorkspaceFallback struct {
	calls int
	last  driver.DependencyRequest
}

func (r *recordingWorkspaceFallback) Resolve(_ context.Context, request driver.DependencyRequest) (
	fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error,
) {
	r.calls++
	r.last = request
	return nil, "", nil, nil, nil
}

func workspaceFS() fstest.MapFS {
	return fstest.MapFS{
		"Ballerina.toml":           &fstest.MapFile{Data: []byte("[workspace]\npackages = [\"lib\", \"apps/main\"]\n")},
		"lib/Ballerina.toml":       &fstest.MapFile{Data: []byte("[package]\norg = \"testorg\"\nname = \"lib\"\nversion = \"1.0.0\"\n[build-options]\noffline = true\n")},
		"lib/lib.bal":              &fstest.MapFile{Data: []byte("public function value() returns int { return 1; }")},
		"apps/main/Ballerina.toml": &fstest.MapFile{Data: []byte("[package]\norg = \"testorg\"\nname = \"app\"\nversion = \"1.0.0\"\n")},
		"apps/main/main.bal":       &fstest.MapFile{Data: []byte("public function main() {}")},
	}
}
