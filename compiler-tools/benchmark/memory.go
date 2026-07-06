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
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"time"
)

const timePath = "/usr/bin/time"

var memorySampleTimeout = 5 * time.Minute

type memoryTool struct {
	path       string
	args       []string
	parserName string
	parser     func(string) (float64, error)
}

var (
	execCommandContext = exec.CommandContext
	gnuMaxRSSPattern   = regexp.MustCompile(`Maximum resident set size \(kbytes\):\s*(\d+)`)
	macOSMaxRSSPattern = regexp.MustCompile(`(?m)^\s*(\d+)\s+maximum resident set size\s*$`)
)

func currentMemoryTool() (memoryTool, error) {
	tool := memoryTool{path: timePath}
	switch runtime.GOOS {
	case "darwin":
		tool.args = []string{"-l"}
		tool.parserName = "macOS time -l"
		tool.parser = parseMacOSMaxRSSMiB
	case "linux":
		tool.args = []string{"-v"}
		tool.parserName = "GNU time -v"
		tool.parser = parseGNUMaxRSSMiB
	default:
		return memoryTool{}, fmt.Errorf("memory mode is not supported on %s", runtime.GOOS)
	}
	return tool, nil
}

func requireMemoryTool() error {
	tool, err := currentMemoryTool()
	if err != nil {
		return err
	}
	if _, err := os.Stat(tool.path); err != nil {
		return fmt.Errorf("%s is required at %s for memory mode: %w", tool.parserName, tool.path, err)
	}
	return nil
}

func (b *benchmark) runMemoryBenchmark(baseWorktree, headWorktree, target string, interpreterBin string) (*benchExport, error) {
	base, err := b.runMemoryCommand(filepath.Join(baseWorktree, interpreterBin), target)
	if err != nil {
		return nil, fmt.Errorf("failed to run memory benchmark for %s: %w", b.baseRef, err)
	}
	head, err := b.runMemoryCommand(filepath.Join(headWorktree, interpreterBin), target)
	if err != nil {
		return nil, fmt.Errorf("failed to run memory benchmark for %s: %w", b.headRef, err)
	}
	return &benchExport{Results: []benchResult{base, head}}, nil
}

func (b *benchmark) runMemoryCommand(interpreter, target string) (benchResult, error) {
	for i := 0; i < b.warmup; i++ {
		if _, err := runMemorySample(interpreter, target); err != nil {
			return benchResult{}, fmt.Errorf("warmup run %d failed: %w", i+1, err)
		}
	}

	samples := make([]float64, 0, b.runs)
	for i := 0; i < b.runs; i++ {
		rssMiB, err := runMemorySample(interpreter, target)
		if err != nil {
			return benchResult{}, fmt.Errorf("benchmark run %d failed: %w", i+1, err)
		}
		samples = append(samples, rssMiB)
	}

	mean, stddev, median := memoryStats(samples)
	return benchResult{
		Command: formatBenchmarkCommand(interpreter, target),
		Mean:    mean,
		Stddev:  stddev,
		Median:  median,
	}, nil
}

func runMemorySample(interpreter, target string) (float64, error) {
	tool, err := currentMemoryTool()
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), memorySampleTimeout)
	defer cancel()

	args := append(append([]string{}, tool.args...), interpreter, "run", target)
	cmd := execCommandContext(ctx, tool.path, args...)
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return 0, fmt.Errorf("command timed out after %s: %w\n%s", memorySampleTimeout, ctx.Err(), stderr.String())
		}
		return 0, fmt.Errorf("command failed: %w\n%s", err, stderr.String())
	}
	return tool.parser(stderr.String())
}

func memoryStats(samples []float64) (float64, float64, float64) {
	if len(samples) == 0 {
		return 0, 0, 0
	}
	var sum float64
	for _, sample := range samples {
		sum += sample
	}
	mean := sum / float64(len(samples))

	var variance float64
	for _, sample := range samples {
		diff := sample - mean
		variance += diff * diff
	}
	stddev := 0.0
	if len(samples) > 1 {
		stddev = math.Sqrt(variance / float64(len(samples)-1))
	}

	sorted := append([]float64{}, samples...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	median := sorted[mid]
	if len(sorted)%2 == 0 {
		median = (sorted[mid-1] + sorted[mid]) / 2
	}
	return mean, stddev, median
}

func parseMaxRSSMiB(output string) (float64, error) {
	if runtime.GOOS == "darwin" {
		return parseMacOSMaxRSSMiB(output)
	}
	return parseGNUMaxRSSMiB(output)
}

func parseGNUMaxRSSMiB(output string) (float64, error) {
	match := gnuMaxRSSPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, fmt.Errorf("failed to parse maximum resident set size from GNU time output")
	}
	rssKiB, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse maximum resident set size %q: %w", match[1], err)
	}
	return rssKiB / 1024.0, nil
}

func parseMacOSMaxRSSMiB(output string) (float64, error) {
	match := macOSMaxRSSPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, fmt.Errorf("failed to parse maximum resident set size from macOS time output")
	}
	rssBytes, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse maximum resident set size %q: %w", match[1], err)
	}
	return rssBytes / 1024.0 / 1024.0, nil
}
