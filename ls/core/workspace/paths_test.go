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
	"path/filepath"
	"runtime"
	"testing"
)

func TestSourcePaths(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX roots; native drive cases have TestWindows coverage")
	}
	for _, root := range []string{"/", "/a/shop", "/a/shop2"} {
		paths := SourcePaths{root: root}
		u, err := FileURIFromPath(filepath.Join(root, "modules/nested/main.bal"))
		if err != nil {
			t.Fatal(err)
		}
		name, ok := paths.NameOf(u)
		if !ok || name != "modules/nested/main.bal" {
			t.Fatalf("%s: name %q, %v", root, name, ok)
		}
		back, ok := paths.URIOf(name)
		if !ok || back != u {
			t.Fatalf("%s: reverse %v, %v", root, back, ok)
		}
	}
	shop := SourcePaths{root: "/a/shop"}
	sibling, _ := FileURIFromPath("/a/shop2/main.bal")
	if _, ok := shop.NameOf(sibling); ok {
		t.Fatal("sibling prefix admitted")
	}
	for _, invalid := range []string{"../main.bal", "/main.bal", "acme/dep/1.0.0::dep.bal", "modules/../main.bal", `modules\main.bal`} {
		if _, ok := shop.URIOf(invalid); ok {
			t.Errorf("admitted %q", invalid)
		}
	}
	a, _ := (SourcePaths{root: "/a/shop"}).URIOf("main.bal")
	b, _ := (SourcePaths{root: "/a/shop2"}).URIOf("main.bal")
	if a == b {
		t.Fatal("same-named documents in different projects collided")
	}
}

func TestSourcePathsCompilerPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	paths := SourcePaths{root: root, prefix: "main.bal/"}
	u, err := FileURIFromPath(filepath.Join(root, "main.bal"))
	if err != nil {
		t.Fatal(err)
	}
	name, ok := paths.NameOf(u)
	if !ok || name != "main.bal/main.bal" {
		t.Fatalf("name=%q ok=%v", name, ok)
	}
	back, ok := paths.URIOf(name)
	if !ok || back != u {
		t.Fatalf("reverse=%v ok=%v", back, ok)
	}
}
