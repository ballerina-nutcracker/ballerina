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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Ballerina corpus programs cannot exercise host JSONL persistence or write failures.
func TestSampleStreamPersistsPartialRole(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "samples.jsonl")
	w, err := newSampleWriter(path, sampleMetadata{Type: "metadata", Version: 1, BaseSHA: "base", HeadSHA: "head", Suite: "time", Phase: "final", Cases: []string{"case.bal"}, Counts: map[string]int{"case.bal": 2}, Unit: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.write(sampleEvent{CaseID: "case.bal", Pair: 0, Role: "base", Position: 0, Value: 1, Unit: "s", Metrics: make(map[string]float64)}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unflushed role: %s", data)
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &event); err != nil {
		t.Fatal(err)
	}
	if event["role"] != "base" || event["type"] != "sample" {
		t.Fatal(event)
	}
	if err := w.close(false); err != nil {
		t.Fatal(err)
	}
	if err := w.write(sampleEvent{Value: 1}); err == nil {
		t.Fatal("write to closed file succeeded")
	}
}

// Pair order and warmup exclusion are host orchestration, outside corpus execution.
func TestAcquirePairsOrderAndFailure(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "samples.jsonl")
		w, err := newSampleWriter(path, sampleMetadata{Type: "metadata", Version: 1, Cases: []string{"x"}, Counts: map[string]int{"x": 4}})
		if err != nil {
			t.Fatal(err)
		}
		b := &benchmark{config: config{warmup: 2}}
		var order []int
		result, err := b.acquirePairs([]string{"B", "H"}, 4, "x", "s", w, func(role int) (float64, error) {
			order = append(order, role)
			if fail && len(order) == 6 {
				return 0, fmt.Errorf("second role failed")
			}
			if len(order) <= 4 {
				return 100, nil
			}
			return float64(role + 1), nil
		})
		if fail {
			if err == nil {
				t.Fatal("missing failure")
			}
			data, _ := os.ReadFile(path)
			if strings.Count(string(data), `"type":"sample"`) != 1 {
				t.Fatal(string(data))
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			want := []int{0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0}
			if !slices.Equal(order, want) {
				t.Fatal(order)
			}
			if result.Results[0].Mean != 1 || result.Results[1].Mean != 2 {
				t.Fatal(result)
			}
		}
		_ = w.close(false)
	}
}

// Count validation and stable IDs are tool CLI inputs, unavailable in corpus programs.
func TestRunCountsAndStableIDs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		data  string
		valid bool
	}{
		{`{"version":1,"counts":{"a":12,"b":14},"plans":{}}`, true},
		{`{"version":1,"counts":{"a":12}}`, false},
		{`{"version":1,"counts":{"a":12,"x":14}}`, false},
		{`{"version":1,"counts":{"a":0,"b":14}}`, false},
		{`{"version":1,"counts":{"a":1.5,"b":14}}`, false},
	} {
		path := filepath.Join(t.TempDir(), "counts.json")
		if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := loadRunCounts(path, []string{"a", "b"}, 10)
		if (err == nil) != test.valid {
			t.Fatalf("%s: %v", test.data, err)
		}
	}
	for _, test := range []struct {
		target     benchmarkTarget
		path, want string
	}{
		{benchmarkTarget{mode: packageMode, root: "/tmp/work"}, "/tmp/work", "."},
		{benchmarkTarget{mode: singleFileMode}, "/tmp/a.bal", "a.bal"},
		{benchmarkTarget{mode: multipleFilesMode, root: "/tmp/work"}, "/tmp/work/a/b.bal", "a/b.bal"},
	} {
		if got := benchmarkCaseID(&test.target, test.path); got != test.want {
			t.Fatalf("%s != %s", got, test.want)
		}
	}
}

// Hyperfine scheduling and shell arguments are host-tool behaviour, outside corpus tests.
// Not parallel: replaces PATH for the hyperfine adapter.
func TestTimeAdapterBalancedPairs(t *testing.T) {
	skipHyperfineStubUnsupported(t)
	dir := t.TempDir()
	script := `#!/bin/sh
out=""
name=""
while [ "$#" -gt 0 ]; do
 case "$1" in
 --export-json) shift; out="$1" ;;
 --command-name) shift; name="$1" ;;
 esac
 shift
done
printf '%s\n' "$name" >> "$PAIR_LOG"
printf '{"results":[{"command":"%s","times":[0.5],"stddev":null}]}' "$name" > "$out"
`
	if err := os.WriteFile(filepath.Join(dir, "hyperfine"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "order")
	t.Setenv("PAIR_LOG", log)
	prependPath(t, dir)
	b := &benchmark{config: config{warmup: 2}}
	export, err := b.runTimeBenchmark([]string{"--command-name", "B", "base command", "--command-name", "H", "head command"}, 4, dir, "case", nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "B\nH\nH\nB\nB\nH\nH\nB\nB\nH\nH\nB\n" {
		t.Fatal(string(data))
	}
	if export.Results[0].Command != "base command" || len(export.Results[1].Times) != 4 {
		t.Fatal(export)
	}
}

// Memory process order and warmup samples are controlled by the tool, not user Ballerina.
// Not parallel: replaces the process-launch adapter.
func TestMemoryAdapterBalancedPairs(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)
	original := execCommandContext
	restore := stubExecCommand(t, []memoryCommandStub{{stderr: memoryCommandOutputForMiB(2)}})
	stub := execCommandContext
	var roles []string
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		roles = append(roles, filepath.Base(filepath.Dir(args[len(args)-3])))
		return stub(ctx, name, args...)
	}
	defer func() { restore(); execCommandContext = original }()
	b := &benchmark{config: config{warmup: 0}}
	_, err := b.runMemoryBenchmark("/B", "/H", "case.bal", "bal", 4, "case", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(roles, []string{"B", "H", "H", "B", "B", "H", "H", "B"}) {
		t.Fatal(roles)
	}
}

// This probes the real hyperfine JSON contract; a Ballerina corpus cannot invoke hyperfine.
func TestRealHyperfineSingleSample(t *testing.T) {
	t.Parallel()
	requireHyperfine(t)
	b := &benchmark{config: config{warmup: 0}}
	export, err := b.runTimeBenchmark([]string{"--command-name", "base", "sleep 1", "--command-name", "head", "sleep 1"}, 2, t.TempDir(), "probe", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range export.Results {
		if len(role.Times) != 2 || role.Mean <= 0 || role.Stddev < 0 {
			t.Fatal(role)
		}
	}
}

// Independent tool modules must produce the same versioned infrastructure wire format.
func TestSampleWriterCommonWireFixture(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "raw.jsonl")
	w, err := newSampleWriter(path, sampleMetadata{Type: "metadata", Version: 1, BaseSHA: strings.Repeat("a", 40), HeadSHA: strings.Repeat("b", 40), Suite: "time", Phase: "fixed", Cases: []string{"case.bal"}, Counts: map[string]int{"case.bal": 1}, Unit: "s"})
	if err != nil {
		t.Fatal(err)
	}
	for role := range 2 {
		if err := w.write(sampleEvent{CaseID: "case.bal", Pair: 0, Role: []string{"base", "head"}[role], Position: role, Value: float64(role + 1), Unit: "s", Metrics: make(map[string]float64)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.completeCase("case.bal"); err != nil {
		t.Fatal(err)
	}
	if err := w.close(true); err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/sample-v1.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantLines := strings.Split(strings.TrimSpace(string(expected)), "\n")
	gotLines := strings.Split(strings.TrimSpace(string(actual)), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatal(string(actual))
	}
	for i, line := range gotLines {
		var got, want map[string]any
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(wantLines[i]), &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("line %d: %v != %v", i, got, want)
		}
	}
}

// These flag combinations are rejected by host CLI validation before worktree creation.
func TestRawConfigValidation(t *testing.T) {
	t.Parallel()
	target := writeTempBalFile(t)
	for _, cfg := range []config{
		{baseRef: "B", headRef: "H", target: target, runs: 1, phase: "unknown"},
		{baseRef: "B", headRef: "H", target: target, runs: 10, runsExplicit: true, runsPath: "counts.json"},
	} {
		if err := cfg.validate(); err == nil {
			t.Fatal("invalid flags accepted")
		}
	}
}

// Commit identity belongs to detached host worktrees, outside Ballerina corpus execution.
func TestSampleSHAUsesDetachedWorktree(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "control")
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "initial")
	initial := git("rev-parse", "HEAD")
	detached := filepath.Join(t.TempDir(), "detached")
	git("worktree", "add", "--detach", detached, initial)
	defer git("worktree", "remove", "--force", detached)
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "advanced")
	got, err := resolveSampleSHA(detached)
	if err != nil {
		t.Fatal(err)
	}
	if got != initial || got == git("rev-parse", "HEAD") {
		t.Fatalf("detached metadata SHA = %s, want %s", got, initial)
	}
}
