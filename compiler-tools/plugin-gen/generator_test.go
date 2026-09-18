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
	if !strings.Contains(text, `acme_demo_4 "example.com/plugins/lib/stdlibs/acme/demo/0.0.1/go1.26/compilerplugin"`) {
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
	// These two differ only in where the org/package boundary falls; a naive
	// org + "_" + pkg alias would map both to acme_shared_util.
	writeTestProvider(t, root, "acme_shared", "util", []string{"ValidateService"}, []string{"ValidateService"})
	writeTestProvider(t, root, "acme", "shared_util", []string{"ValidateService"}, []string{"ValidateService"})

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
	for _, expected := range []string{
		`acme_shared_4 "example.com/plugins/lib/stdlibs/acme/shared/0.0.1/go1.26/compilerplugin"`,
		`zeta_shared_4 "example.com/plugins/lib/stdlibs/zeta/shared/0.0.1/go1.26/compilerplugin"`,
		`acme_shared_util_11 "example.com/plugins/lib/stdlibs/acme_shared/util/0.0.1/go1.26/compilerplugin"`,
		`acme_shared_util_4 "example.com/plugins/lib/stdlibs/acme/shared_util/0.0.1/go1.26/compilerplugin"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing deterministic import alias %s:\n%s", expected, text)
		}
	}
}

func TestPluginImportAliasIsInjective(t *testing.T) {
	providers := [][2]string{
		{"acme", "shared"},
		{"acme_shared", "util"},
		{"acme", "shared_util"},
		{"acme_", "shared"},
		{"acme", "_shared"},
		{"a.b", "c"},
		{"a", "b.c"},
	}
	aliases := make(map[string][2]string, len(providers))
	for _, provider := range providers {
		alias := pluginImportAlias(provider[0], provider[1])
		if previous, clash := aliases[alias]; clash {
			t.Fatalf("alias %q is shared by %v and %v", alias, previous, provider)
		}
		aliases[alias] = provider
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
