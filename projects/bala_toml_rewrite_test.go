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

package projects

import "testing"

func TestRewriteBallerinaTomlForBala_PackageHeaderWithTrailingComment(t *testing.T) {
	t.Parallel()
	content := "[package] # package metadata\n" +
		"org = \"testorg\"\n" +
		"name = \"testpkg\"\n" +
		"version = \"0.1.0\"\n" +
		"readme = \"README.md\"\n"

	manifest := NewPackageManifestFromParams(PackageManifestParams{Readme: "README.md"})
	got := rewriteBallerinaTomlForBala(content, manifest)

	want := "[package] # package metadata\n" +
		"org = \"testorg\"\n" +
		"name = \"testpkg\"\n" +
		"version = \"0.1.0\"\n" +
		"readme = \"docs/README.md\"\n"
	if got != want {
		t.Errorf("rewriteBallerinaTomlForBala() =\n%s\nwant:\n%s", got, want)
	}
}

func TestRewriteBallerinaTomlForBala_SingleQuotedReadme(t *testing.T) {
	t.Parallel()
	content := "[package]\n" +
		"org = \"testorg\"\n" +
		"name = \"testpkg\"\n" +
		"version = \"0.1.0\"\n" +
		"readme = 'README.md'\n"

	manifest := NewPackageManifestFromParams(PackageManifestParams{Readme: "README.md"})
	got := rewriteBallerinaTomlForBala(content, manifest)

	// Rewritten in place (no duplicate readme key), normalized to
	// double-quoted — TOML doesn't distinguish string-literal styles
	// semantically, so this is not itself a behavior change.
	want := "[package]\n" +
		"org = \"testorg\"\n" +
		"name = \"testpkg\"\n" +
		"version = \"0.1.0\"\n" +
		"readme = \"docs/README.md\"\n"
	if got != want {
		t.Errorf("rewriteBallerinaTomlForBala() =\n%s\nwant:\n%s", got, want)
	}
}

func TestRewriteBallerinaTomlForBala_SingleQuotedModuleName(t *testing.T) {
	t.Parallel()
	content := "[package]\n" +
		"org = \"testorg\"\n" +
		"name = \"testpkg\"\n" +
		"version = \"0.1.0\"\n" +
		"\n" +
		"[[package.modules]]\n" +
		"name = 'testpkg.extra'\n"

	manifest := NewPackageManifestFromParams(PackageManifestParams{
		Modules: []ManifestModule{
			NewManifestModule("testpkg.extra", false, "", "modules/extra/README.md"),
		},
	})
	got := rewriteBallerinaTomlForBala(content, manifest)
	// An inserted readme line lands right after the [[package.modules]]
	// header (mod.start), before the block's own existing content.
	want := "[package]\n" +
		"org = \"testorg\"\n" +
		"name = \"testpkg\"\n" +
		"version = \"0.1.0\"\n" +
		"\n" +
		"[[package.modules]]\n" +
		"readme = \"docs/modules/testpkg.extra/README.md\"\n" +
		"name = 'testpkg.extra'\n"
	if got != want {
		t.Errorf("rewriteBallerinaTomlForBala() =\n%s\nwant:\n%s", got, want)
	}
}
