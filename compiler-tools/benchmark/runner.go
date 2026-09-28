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
	"strconv"
	"strings"
	"time"
)

const builtInterpreter = "bal"

type (
	benchmark struct {
		config
		workRoot string
	}
	runResult struct {
		export benchExport
		label  string
	}
)

func (b *benchmark) run() error {
	if b.mode == timeMode {
		if _, err := exec.LookPath("hyperfine"); err != nil {
			return fmt.Errorf("hyperfine is required but was not found in PATH; please install it and retry: %w", err)
		}
	} else if err := requireMemoryTool(); err != nil {
		return err
	}

	pruneWorktrees()

	target, err := resolveTarget(b.target)
	if err != nil {
		return fmt.Errorf("failed to resolve benchmark target: %w", err)
	}

	// Each resource registers its teardown as it is created, so an interrupt
	// tears down exactly what exists at that moment.
	var c cleanups
	defer c.run()
	defer onInterrupt(c.run)()

	workRoot, err := os.MkdirTemp("", "bal-bench-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	c.add(func() { _ = os.RemoveAll(workRoot) })
	b.workRoot = workRoot

	baseWorktree, err := b.checkoutWorktree(b.baseRef)
	if err != nil {
		return err
	}
	c.add(func() { b.removeWorktree(baseWorktree) })

	headWorktree, err := b.checkoutWorktree(b.headRef)
	if err != nil {
		return err
	}
	c.add(func() { b.removeWorktree(headWorktree) })

	interpreterBin := builtInterpreterBinaryName()
	fmt.Printf("Building interpreter for %s...\n", b.baseRef)
	if err := b.buildInterpreter(baseWorktree, b.baseRef, interpreterBin); err != nil {
		return err
	}
	fmt.Printf("Building interpreter for %s...\n", b.headRef)
	if err := b.buildInterpreter(headWorktree, b.headRef, interpreterBin); err != nil {
		return err
	}

	exportDir, err := os.MkdirTemp("", "bal-bench-exports-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for exports: %w", err)
	}
	c.add(func() { _ = os.RemoveAll(exportDir) })

	results, err := b.runBenchmarks(baseWorktree, headWorktree, target, interpreterBin, exportDir)
	if err != nil {
		return err
	}

	rep := report{
		BaseRef:   b.baseRef,
		HeadRef:   b.headRef,
		Mode:      b.mode,
		Generated: time.Now(),
		results:   results,
	}
	return writeReports(&rep, b.exportPath, b.summaryPath)
}

func writeReports(rep *report, exportPath, summaryPath string) error {
	if exportPath != "" {
		if err := rep.export(exportPath); err != nil {
			return err
		}
		fmt.Printf("Benchmark report exported to %s\n", exportPath)
	}
	if summaryPath != "" {
		if err := rep.exportSummary(summaryPath); err != nil {
			return err
		}
		fmt.Printf("Benchmark summary exported to %s\n", summaryPath)
	}
	return nil
}

func (b *benchmark) runBenchmarks(baseWorktree, headWorktree string, target *benchmarkTarget, interpreterBin, exportDir string) ([]runResult, error) {
	results := make([]runResult, 0, len(target.paths))
	for _, path := range target.paths {
		var export *benchExport
		var err error
		if b.mode == memoryMode {
			export, err = b.runMemoryBenchmark(baseWorktree, headWorktree, path, interpreterBin)
		} else {
			cmds := b.benchmarkCmdPair(baseWorktree, headWorktree, target.root, path, target.mode, interpreterBin)
			exportPath := filepath.Join(exportDir, fmt.Sprintf("%s.json", sanitize(path)))
			export, err = b.runHyperfine(cmds, exportPath)
		}
		if err != nil {
			return nil, err
		}
		results = append(results, runResult{
			label:  benchmarkResultLabel(target, path),
			export: *export,
		})
	}
	return results, nil
}

func (b *benchmark) checkoutWorktree(ref string) (string, error) {
	path := filepath.Join(b.workRoot, "worktree-"+sanitize(ref)+"-"+sanitize(filepath.Base(b.workRoot)))
	b.removeWorktree(path)

	if err := runCmd(".", "git", "worktree", "add", "--detach", path, ref); err != nil {
		return "", fmt.Errorf("failed to checkout worktree for ref %q: %w", ref, err)
	}
	return path, nil
}

// removeWorktree unregisters the worktree at path. --force is passed twice:
// git writes a "locked" marker for the duration of `worktree add`, and an
// interrupted add leaves one behind that a single --force refuses to remove.
func (b *benchmark) removeWorktree(path string) {
	_ = runCmdSilent(".", "git", "worktree", "remove", "--force", "--force", path)
}

// pruneWorktrees drops registrations whose directories were removed behind
// git's back, e.g. by the OS reaping the temporary directory.
func pruneWorktrees() {
	_ = runCmdSilent(".", "git", "worktree", "prune")
}

func (b *benchmark) buildInterpreter(worktreePath, ref, output string) error {
	// -trimpath keeps the build cache shareable across checkouts: without it the
	// compiler keys every package by its absolute source path, so each temporary
	// worktree seeds its own unshareable slice of GOCACHE.
	if err := runCmd(worktreePath, "go", "build", "-trimpath", "-o", output, "./cli/cmd"); err != nil {
		return fmt.Errorf("failed to build interpreter for ref %q: %w", ref, err)
	}
	return nil
}

func (b *benchmark) hyperfineFlags() []string {
	args := []string{"--show-output"}
	if b.warmup > 0 {
		args = append(args, "--warmup", strconv.Itoa(b.warmup))
	}
	args = append(args, "--runs", strconv.Itoa(b.runs))
	return args
}

func (b *benchmark) runHyperfine(cmds []string, jsonExportPath string) (*benchExport, error) {
	args := b.hyperfineFlags()
	args = append(args, "--export-json", jsonExportPath)
	args = append(args, cmds...)
	if err := runCmd(".", "hyperfine", args...); err != nil {
		return nil, fmt.Errorf("failed to run hyperfine: %w", err)
	}
	return parseHyperfineExport(jsonExportPath)
}

func (b *benchmark) benchmarkCmdArgs(ref, interpreter, root, target string, mode targetMode) []string {
	command := formatBenchmarkCommand(interpreter, target)
	if mode == multipleFilesMode {
		ref = fmt.Sprintf("%s (%s)", ref, getRelativeLabel(root, target))
	}
	return []string{"--command-name", ref, command}
}

func (b *benchmark) benchmarkCmdPair(baseWorktree, headWorktree, root, target string, mode targetMode, interpreterBin string) []string {
	baseCmd := b.benchmarkCmdArgs(b.baseRef, filepath.Join(baseWorktree, interpreterBin), root, target, mode)
	headCmd := b.benchmarkCmdArgs(b.headRef, filepath.Join(headWorktree, interpreterBin), root, target, mode)
	return append(baseCmd, headCmd...)
}

func getRelativeLabel(root, path string) string {
	if root != "" {
		if rel, err := filepath.Rel(root, path); err == nil {
			return rel
		}
	}
	return filepath.Base(path)
}

func sanitize(ref string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':':
			return '-'
		default:
			return r
		}
	}, ref)
}

func shellQuoteForOS(goos, s string) string {
	if goos == "windows" {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func formatBenchmarkCommand(interpreter, target string) string {
	return formatBenchmarkCommandForOS(runtime.GOOS, interpreter, target)
}

func formatBenchmarkCommandForOS(goos, interpreter, target string) string {
	cmd := fmt.Sprintf("%s run %s", shellQuoteForOS(goos, interpreter), shellQuoteForOS(goos, target))
	if goos == "windows" {
		return `"` + cmd + `"`
	}
	return cmd
}

func builtInterpreterBinaryName() string {
	if runtime.GOOS == "windows" {
		return builtInterpreter + ".exe"
	}
	return builtInterpreter
}

func benchmarkResultLabel(target *benchmarkTarget, path string) string {
	if target.mode != multipleFilesMode {
		return target.label
	}
	label := path
	for strings.HasPrefix(label, "../") {
		label = strings.TrimPrefix(label, "../")
	}
	return label
}

func runCmd(dir, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runCmdSilent(dir, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stdout = nil
	c.Stderr = nil
	return c.Run()
}
