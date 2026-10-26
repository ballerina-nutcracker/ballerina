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
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	scaleEnvVar        = "BAL_SCALE"
	startupProgramName = "startup.bal"
	startupProgram     = "public function main() {\n}\n"
)

var errScalingExceeded = errors.New("super-linear scaling detected")

func (b *benchmark) runScaling(c *cleanups, target *benchmarkTarget) error {
	worktree, err := b.checkoutWorktree(b.headRef)
	if err != nil {
		return err
	}
	c.add(func() { b.removeWorktree(worktree) })

	interpreterBin := builtInterpreterBinaryName()
	fmt.Printf("Building interpreter for %s...\n", b.headRef)
	if err := b.buildInterpreter(worktree, b.headRef, interpreterBin); err != nil {
		return err
	}

	rep, err := b.measureScaling(filepath.Join(worktree, interpreterBin), target, b.workRoot)
	if err != nil {
		return err
	}
	return finishScaling(rep, b.summaryPath)
}

// measureScaling times the startup program and then every target path at
// BAL_SCALE=1 and BAL_SCALE=factor, keeping hyperfine exports in workDir.
func (b *benchmark) measureScaling(interpreter string, target *benchmarkTarget, workDir string) (*scalingReport, error) {
	startupPath := filepath.Join(workDir, startupProgramName)
	if err := os.WriteFile(startupPath, []byte(startupProgram), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write startup program: %w", err)
	}
	// The first run of a freshly built binary is much slower than the rest, and
	// an inflated startup would be subtracted from every case.
	primed := *b
	primed.warmup = max(b.warmup, 1)
	startup, err := primed.timeCommand(interpreter, startupPath, "startup", nil, filepath.Join(workDir, "startup.json"))
	if err != nil {
		return nil, err
	}

	rep := &scalingReport{
		ref:      b.headRef,
		factor:   b.factor,
		maxRatio: b.maxRatio,
		startup:  *startup,
		cases:    make([]scalingCase, 0, len(target.paths)),
	}
	for _, path := range target.paths {
		label := benchmarkResultLabel(target, path)
		unit, err := b.timeAtScale(interpreter, path, label, 1, workDir)
		if err != nil {
			return nil, err
		}
		scaled, err := b.timeAtScale(interpreter, path, label, b.factor, workDir)
		if err != nil {
			return nil, err
		}
		rep.cases = append(rep.cases, scalingCase{label: label, unit: *unit, scaled: *scaled})
	}
	return rep, nil
}

func (b *benchmark) timeAtScale(interpreter, path, label string, scale int, workDir string) (*benchResult, error) {
	name := fmt.Sprintf("%s %s=%d", label, scaleEnvVar, scale)
	exportPath := filepath.Join(workDir, fmt.Sprintf("%s-%d.json", sanitize(path), scale))
	env := []string{fmt.Sprintf("%s=%d", scaleEnvVar, scale)}
	return b.timeCommand(interpreter, path, name, env, exportPath)
}

func (b *benchmark) timeCommand(interpreter, path, name string, env []string, exportPath string) (*benchResult, error) {
	cmds := []string{"--command-name", name, formatBenchmarkCommand(interpreter, path)}
	export, err := b.runHyperfineWithEnv(cmds, exportPath, env)
	if err != nil {
		return nil, err
	}
	if len(export.Results) != 1 {
		return nil, fmt.Errorf("hyperfine export for %q has %d results, want 1", name, len(export.Results))
	}
	return &export.Results[0], nil
}

// finishScaling prints and exports the summary, then reports any failing case
// as errScalingExceeded so the caller can exit with a distinct status.
func finishScaling(rep *scalingReport, summaryPath string) error {
	summary := rep.renderSummary()
	fmt.Print("\n" + summary)
	if summaryPath != "" {
		if err := os.WriteFile(summaryPath, []byte(summary), 0o644); err != nil {
			return fmt.Errorf("failed to write summary report %q: %w", summaryPath, err)
		}
		fmt.Printf("Benchmark summary exported to %s\n", summaryPath)
	}
	if failed := rep.count(verdictFail); failed > 0 {
		return fmt.Errorf("%w: %d case(s) exceeded ratio %.1f at factor %d", errScalingExceeded, failed, rep.maxRatio, rep.factor)
	}
	return nil
}
