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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const integrationCoverDirEnv = "CODECOV_INTEGRATION_COVERDIR"

// treeGenCLI runs tree-gen through a coverage-instrumented binary when
// integration coverage is requested, and through `go run` otherwise.
type treeGenCLI struct {
	modDir   string
	binPath  string
	coverDir string
}

func TestGenerateFile(t *testing.T) {
	t.Parallel()
	tests := []generateCase{
		{
			name:            "writes output",
			templateContent: "Hello {{.NodeType}}",
			templatePath:    "template.tmpl",
			outputPath:      "out.go",
			data:            map[string]interface{}{"NodeType": "node", "AbstractTypes": map[string]bool{}},
			wantOutput:      "Hello node",
		},
		{
			name:            "title and isAbstract funcs",
			templateContent: `{{title ""}}|{{title "abc"}}|{{isAbstract "A"}}|{{isAbstract "B"}}`,
			templatePath:    "template.tmpl",
			outputPath:      "out.txt",
			data:            map[string]interface{}{"NodeType": "node", "AbstractTypes": map[string]bool{"A": true}},
			wantOutput:      "|Abc|true|false",
		},
		{
			name:            "creates nested output directory",
			templateContent: "ok",
			templatePath:    "template.tmpl",
			outputPath:      filepath.Join("nested", "deep", "out.go"),
			data:            map[string]interface{}{"NodeType": "node", "AbstractTypes": map[string]bool{}},
			wantOutput:      "ok",
		},
		{
			name:            "missing template",
			templatePath:    "missing.tmpl",
			outputPath:      "out.go",
			data:            map[string]interface{}{"NodeType": "node", "AbstractTypes": map[string]bool{}},
			wantErrContains: "loading template",
		},
		{
			name:            "template execute error",
			templateContent: `{{template "nonexistent" .}}`,
			templatePath:    "template.tmpl",
			outputPath:      "out.go",
			data:            map[string]interface{}{"NodeType": "node", "AbstractTypes": map[string]bool{}},
			wantErrContains: "executing template",
		},
		{
			name:            "isAbstract wrong map type",
			templateContent: `{{isAbstract "X"}}`,
			templatePath:    "template.tmpl",
			outputPath:      "out.txt",
			data:            map[string]interface{}{"NodeType": "node", "AbstractTypes": "not-a-map"},
			wantOutput:      "false",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runGenerateCase(t, tc)
		})
	}
}

func TestCLI(t *testing.T) {
	t.Parallel()
	cli := newTreeGenCLI(t, moduleDir(t))
	tests := []cliCase{
		{
			name: "generates main output",
			files: map[string]string{
				"nodes.json":    `[{"name":"N","kind":"K","attributes":[]}]`,
				"template.tmpl": `{{range .Nodes}}{{.Name}}{{end}}`,
				"want/gen.go":   "N",
				"stdout-marker": "Successfully generated",
			},
			args: []string{
				"-config", "@nodes.json",
				"-type", "node",
				"-template", "@template.tmpl",
				"-output", "@gen.go",
			},
			wantStdoutContain: "@stdout-marker",
			wantFiles:         map[string]string{"gen.go": "@want/gen.go"},
		},
		{
			name: "generates with util",
			files: map[string]string{
				"nodes.json":   `[]`,
				"main.tmpl":    `main`,
				"util.tmpl":    `util`,
				"want/main.go": "main",
				"want/util.go": "util",
			},
			args: []string{
				"-config", "@nodes.json",
				"-type", "st-node",
				"-template", "@main.tmpl",
				"-output", "@main.go",
				"-util-template", "@util.tmpl",
				"-util-output", "@util.go",
			},
			wantFiles: map[string]string{"main.go": "@want/main.go", "util.go": "@want/util.go"},
		},
		{
			name: "missing required flags",
			files: map[string]string{
				"nodes.json":    `[{"name":"N","kind":"K","attributes":[]}]`,
				"template.tmpl": `{{range .Nodes}}{{.Name}}{{end}}`,
			},
			args: []string{
				"-config", "@nodes.json",
				"-type", "node",
				"-template", "@template.tmpl",
			},
			wantErrContains: "all flags required (config, type, template, output)",
		},
		{
			name: "invalid node type",
			args: []string{
				"-config", "@x",
				"-type", "other",
				"-template", "@y",
				"-output", "@z",
			},
			wantErrContains: "type must be",
		},
		{
			name: "missing config",
			args: []string{
				"-config", "@nope.json",
				"-type", "node",
				"-template", "@y",
				"-output", "@z",
			},
			wantErrContains: "Error reading config file",
		},
		{
			name: "bad json",
			files: map[string]string{
				"bad.json": "not json",
				"t.tmpl":   "unused",
			},
			args: []string{
				"-config", "@bad.json",
				"-type", "node",
				"-template", "@t.tmpl",
				"-output", "@out.go",
			},
			wantErrContains: "Error parsing config JSON",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runCLICase(t, cli, tc)
		})
	}
}

type generateCase struct {
	name            string
	templateContent string
	templatePath    string
	outputPath      string
	data            map[string]interface{}
	wantOutput      string
	wantErrContains string
}

type cliCase struct {
	name              string
	files             map[string]string
	args              []string
	wantErrContains   string
	wantStdoutContain string
	wantFiles         map[string]string
}

func moduleDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func newTreeGenCLI(t *testing.T, modDir string) treeGenCLI {
	t.Helper()
	coverDir, err := resolveIntegrationCoverDir()
	if err != nil {
		t.Fatal(err)
	}
	if coverDir == "" {
		return treeGenCLI{modDir: modDir}
	}
	name := "tree-gen"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), name)
	if err := buildTreeGenBinary(coverDir, modDir, binPath); err != nil {
		t.Fatal(err)
	}
	return treeGenCLI{modDir: modDir, binPath: binPath, coverDir: coverDir}
}

func goRunTreeGen(t *testing.T, cli treeGenCLI, args ...string) ([]byte, error) {
	t.Helper()
	var cmd *exec.Cmd
	if cli.binPath != "" {
		cmd = exec.Command(cli.binPath, args...)
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+cli.coverDir)
	} else {
		cmd = exec.Command("go", append([]string{"run", "."}, args...)...)
	}
	cmd.Dir = cli.modDir
	return cmd.CombinedOutput()
}

func buildTreeGenBinary(coverDir, modDir, outputPath string) error {
	args := []string{"build", "-o", outputPath}
	if coverDir != "" {
		args = append(args, "-cover", "-coverpkg=./...")
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = modDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build tree-gen: %w\n%s", err, string(out))
	}
	return nil
}

func resolveIntegrationCoverDir() (string, error) {
	d := os.Getenv(integrationCoverDirEnv)
	if d == "" {
		return "", nil
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s %q: %w", integrationCoverDirEnv, d, err)
	}
	return d, nil
}

func runGenerateCase(t *testing.T, tc generateCase) {
	t.Helper()
	workDir := t.TempDir()
	templatePath := filepath.Join(workDir, tc.templatePath)
	if tc.templateContent != "" {
		mustWriteFile(t, templatePath, tc.templateContent)
	}
	outputPath := filepath.Join(workDir, tc.outputPath)

	err := generateFile(templatePath, outputPath, tc.data)
	assertErrorContains(t, err, tc.wantErrContains)
	if tc.wantErrContains != "" {
		return
	}
	assertFileContent(t, outputPath, tc.wantOutput)
}

func runCLICase(t *testing.T, cli treeGenCLI, tc cliCase) {
	t.Helper()
	workDir := t.TempDir()
	for rel, content := range tc.files {
		if strings.HasPrefix(rel, "want/") || rel == "stdout-marker" {
			continue
		}
		mustWriteFile(t, filepath.Join(workDir, rel), content)
	}
	args := resolveCasePaths(workDir, tc.args)
	out, err := goRunTreeGen(t, cli, args...)
	if tc.wantErrContains != "" {
		if err == nil {
			t.Fatalf("expected command to fail, output: %q", string(out))
		}
		if !strings.Contains(string(out), tc.wantErrContains) {
			t.Fatalf("output %q does not contain %q", string(out), tc.wantErrContains)
		}
		return
	}
	if err != nil {
		t.Fatalf("tree-gen run failed: %v\n%s", err, out)
	}
	if marker := resolveExpectedValue(workDir, tc.files, tc.wantStdoutContain); marker != "" && !strings.Contains(string(out), marker) {
		t.Fatalf("stdout %q does not contain %q", string(out), marker)
	}
	for rel, want := range tc.wantFiles {
		assertFileContent(t, filepath.Join(workDir, rel), resolveExpectedValue(workDir, tc.files, want))
	}
}

func resolveCasePaths(workDir string, args []string) []string {
	resolved := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			resolved = append(resolved, filepath.Join(workDir, strings.TrimPrefix(arg, "@")))
			continue
		}
		resolved = append(resolved, arg)
	}
	return resolved
}

func resolveExpectedValue(workDir string, files map[string]string, value string) string {
	if !strings.HasPrefix(value, "@") {
		return value
	}
	key := strings.TrimPrefix(value, "@")
	if content, ok := files[key]; ok {
		return content
	}
	return filepath.Join(workDir, key)
}

func assertErrorContains(t *testing.T, err error, wantContains string) {
	t.Helper()
	if wantContains == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), wantContains) {
		t.Fatalf("error %v does not contain %q", err, wantContains)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if normalize(string(got)) != normalize(want) {
		t.Fatalf("%s got %q want %q", path, string(got), want)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func normalize(s string) string {
	return strings.TrimSuffix(s, "\n")
}
