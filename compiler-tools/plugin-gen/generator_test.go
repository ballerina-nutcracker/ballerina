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

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedRegistryIsCurrent(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(filepath.Join(root, registryDirectory, "plugins.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, committed) {
		t.Fatal("generated compiler plugin registry is stale; run go run ./compiler-tools/plugin-gen -root .")
	}
}

func TestGenerateDerivesImportAndPreservesOrder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/plugins\n\ngo 1.26\n")
	writeTestProvider(t, root, "acme", "demo", []string{"First", "Second"}, []string{"First", "Second"})

	generated, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	if !strings.Contains(text, `acme_demo "example.com/plugins/lib/stdlibs/acme/demo/0.0.1/go1.26/compilerplugin"`) {
		t.Fatalf("derived import missing:\n%s", text)
	}
	if strings.Count(text, `example.com/plugins/lib/stdlibs/acme/demo/0.0.1/go1.26/compilerplugin`) != 1 {
		t.Fatalf("provider implementation was imported more than once:\n%s", text)
	}
	if strings.Index(text, `"First"`) > strings.Index(text, `"Second"`) {
		t.Fatalf("registry order was not preserved:\n%s", text)
	}
}

func TestGenerateRejectsMissingFunction(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/plugins\n\ngo 1.26\n")
	writeTestProvider(t, root, "acme", "demo", []string{"Missing"}, nil)
	if _, err := generate(root); err == nil {
		t.Fatal("expected missing function error")
	}
}

// Module discovery is generator-only behavior and cannot be exercised from Ballerina source.
func TestGenerateAllowsModuleDirectiveTrailingComment(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/plugins // deprecated\n\ngo 1.26\n")
	writeTestProvider(t, root, "acme", "demo", []string{"ValidateService"}, []string{"ValidateService"})

	generated, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), `"example.com/plugins/lib/stdlibs/acme/demo/0.0.1/go1.26/compilerplugin"`) {
		t.Fatalf("derived import missing:\n%s", generated)
	}
}

func TestGenerateOrdersProvidersAndAvoidsAliasCollisions(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/plugins\n\ngo 1.26\n")
	writeTestProvider(t, root, "zeta", "shared", []string{"ValidateService"}, []string{"ValidateService"})
	writeTestProvider(t, root, "acme", "shared", []string{"ValidateService"}, []string{"ValidateService"})

	generated, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	again, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, again) {
		t.Fatal("generator output is not deterministic")
	}
	text := string(generated)
	if strings.Index(text, `register("acme", "shared"`) > strings.Index(text, `register("zeta", "shared"`) {
		t.Fatalf("providers are not ordered lexically:\n%s", text)
	}
	if !strings.Contains(text, `acme_shared "example.com/plugins/lib/stdlibs/acme/shared/0.0.1/go1.26/compilerplugin"`) ||
		!strings.Contains(text, `zeta_shared "example.com/plugins/lib/stdlibs/zeta/shared/0.0.1/go1.26/compilerplugin"`) {
		t.Fatalf("distinct deterministic import aliases missing:\n%s", text)
	}
}

func writeTestProvider(t *testing.T, root, org, pkg string, declared, implemented []string) {
	t.Helper()
	directory := filepath.Join(root, stdlibDirectory, org, pkg, "0.0.1", "go1.26")
	writeTestFile(t, filepath.Join(directory, ballerinaManifestFile),
		"[package]\norg = \""+org+"\"\nname = \""+pkg+"\"\nversion = \"0.0.1\"\n")
	manifest := ""
	for _, function := range declared {
		manifest += "[[plugin]]\nstage = \"after-semantics\"\nfunction = \"" + function + "\"\n"
	}
	writeTestFile(t, filepath.Join(directory, compilerPluginManifest), manifest)
	content := "package compilerplugin\n"
	for _, function := range implemented {
		content += "func " + function + "() {}\n"
	}
	writeTestFile(t, filepath.Join(directory, compilerPluginDirectory, "plugin.go"), content)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
