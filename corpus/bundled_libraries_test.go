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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package corpus

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/lib/langlibs"
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
)

// TestBundledLibrariesCompileClean compiles every bundled langlib and stdlib module
// from source and fails on any error or warning it produces. It is a Go test because it
// must enumerate and compile every bundled module, which a .bal corpus test can't do.
func TestBundledLibrariesCompileClean(t *testing.T) {
	t.Parallel()
	modules, err := bundledModuleNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) == 0 {
		t.Fatal("no bundled modules found")
	}
	for _, module := range modules {
		t.Run(module, func(t *testing.T) {
			t.Parallel()
			compileBundledModule(t, module)
		})
	}
}

// bundledModuleNames lists every bundled module as <org>/<module>, including sub-modules.
// The embedded trees are laid out as <org>/<package>/<version>/<platform>/[modules/<sub>].
func bundledModuleNames() ([]string, error) {
	var names []string
	for _, fsys := range []fs.FS{langlibs.FS, stdlibs.FS} {
		platformDirs, err := fs.Glob(fsys, "*/*/*/*")
		if err != nil {
			return nil, err
		}
		for _, platformDir := range platformDirs {
			info, err := fs.Stat(fsys, platformDir)
			if err != nil {
				return nil, err
			}
			if !info.IsDir() {
				continue
			}
			parts := strings.Split(platformDir, "/")
			org, pkg := parts[0], parts[1]
			if slices.Contains(names, org+"/"+pkg) {
				return nil, fmt.Errorf("%s/%s is bundled more than once (found again at %s)", org, pkg, platformDir)
			}
			names = append(names, org+"/"+pkg)
			subModules, err := fs.ReadDir(fsys, path.Join(platformDir, "modules"))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
			for _, sub := range subModules {
				if sub.IsDir() {
					names = append(names, org+"/"+pkg+"."+sub.Name())
				}
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

func compileBundledModule(t *testing.T, module string) {
	projectDir := t.TempDir()
	source := fmt.Sprintf("import %s as _;\n\npublic function main() {\n}\n", quotedImportPath(module))
	if err := os.WriteFile(filepath.Join(projectDir, "main.bal"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := projects.Load(os.DirFS(projectDir), "main.bal", projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	compilation := result.Project().CurrentPackage().Compilation()
	if !compiledFromSource(compilation, module) {
		t.Errorf("%s was not compiled from source", module)
	}
	diagResult := compilation.DiagnosticResult()
	if diagResult.HasErrors() || diagResult.HasWarnings() {
		var rendered strings.Builder
		testharness.PrintDiagnostics(os.DirFS(projectDir), &rendered, diagResult, compilation.DiagnosticEnv())
		t.Errorf("%s produced diagnostics:\n%s", module, rendered.String())
	}
}

// quotedImportPath quotes every module name component after the first, since
// components such as `int` and `any` are keywords.
func quotedImportPath(module string) string {
	org, name, _ := strings.Cut(module, "/")
	components := strings.Split(name, ".")
	for i := 1; i < len(components); i++ {
		components[i] = "'" + components[i]
	}
	return org + "/" + strings.Join(components, ".")
}

func compiledFromSource(compilation *projects.PackageCompilation, module string) bool {
	for _, desc := range compilation.SourceCompiledModules() {
		if desc.Org().Value()+"/"+desc.Name().String() == module {
			return true
		}
	}
	return false
}
