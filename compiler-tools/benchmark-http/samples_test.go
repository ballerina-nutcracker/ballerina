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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
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

// CLI validation must happen before creating worktrees or services.
func TestRejectInvalidMeasurementOptions(t *testing.T) {
	t.Parallel()
	for _, flag := range []string{"--repeats=0", "--conns=-1", "--warmup=0s", "--duration=bad", "--threshold=NaN", "--threshold=100", "--phase=other"} {
		if _, err := parseConfig([]string{flag, "base", "head"}); err == nil {
			t.Fatalf("accepted %s", flag)
		}
	}
}

// Host service ordering and interrupted export cannot be exercised by corpus tests.
func TestHTTPBalancedPairsAndPartialFailure(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "raw.jsonl")
		w, err := newSampleWriter(path, sampleMetadata{Type: "metadata", Version: 1, Cases: []string{"hello"}, Counts: map[string]int{"hello": 4}})
		if err != nil {
			t.Fatal(err)
		}
		var order []int
		samples, err := collectHTTPPairs(config{repeats: 4}, w, func(role int) (sample, error) {
			order = append(order, role)
			if fail && len(order) == 2 {
				return sample{}, fmt.Errorf("failed head")
			}
			return sample{wrkMetrics: wrkMetrics{throughput: float64(role + 1)}}, nil
		})
		if fail {
			if err == nil {
				t.Fatal("missing failure")
			}
			data, _ := os.ReadFile(path)
			if strings.Count(string(data), `"type":"sample"`) != 1 || strings.Contains(string(data), "case_complete") {
				t.Fatal(string(data))
			}
			_ = w.close(false)
		} else {
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(order, []int{0, 1, 1, 0, 0, 1, 1, 0}) {
				t.Fatal(order)
			}
			if len(samples[0]) != 4 || samples[1][0].throughput != 2 {
				t.Fatal(samples)
			}
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
	expected, err := os.ReadFile("../benchmark/testdata/sample-v1.jsonl")
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

// A zero-request host load run cannot be generated by Ballerina corpus assertions.
func TestHTTPRejectsZeroThroughputWithoutRawExport(t *testing.T) {
	t.Parallel()
	metrics, err := parseWrk("0 requests in 1.00s, 0B read\nRequests/sec: 0.00\n")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	got, err := collectHTTPPairs(config{repeats: 2}, nil, func(int) (sample, error) {
		calls++
		return sample{wrkMetrics: metrics}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "invalid throughput") || got != nil || calls != 1 {
		t.Fatalf("zero-throughput load: samples=%v calls=%d err=%v", got, calls, err)
	}
}

// Host sample production cannot be exercised through Ballerina corpus programs.
func TestHTTPPairMetricWireContract(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "samples.jsonl")
	w, err := newSampleWriter(path, sampleMetadata{Type: "metadata"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = collectHTTPPairs(config{repeats: 1}, w, func(role int) (sample, error) {
		return sample{wrkMetrics: wrkMetrics{throughput: 100, avgMs: 1, stdevMs: 2, maxMs: 3, errPct: 4, p99Ms: 5}, startupSec: 6, rssMB: 7}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Fatalf("unexpected stream: %s", data)
	}
	for _, line := range lines[1:3] {
		var event sampleEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		want := map[string]float64{"avg_ms": 1, "stdev_ms": 2, "max_ms": 3, "error_pct": 4, "p99_ms": 5, "startup_s": 6, "rss_mib": 7}
		if event.Unit != "req/s" || event.Value != 100 || !reflect.DeepEqual(event.Metrics, want) {
			t.Fatalf("HTTP sample wire contract: %+v, want unit req/s, value 100, metrics %v", event, want)
		}
	}
}

// Host suite metadata is produced before service launch, outside the corpus.
// Not parallel: this test replaces external tools through PATH.
func TestHTTPRunMetadataWireContract(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell tool stubs require Unix")
	}
	dir := t.TempDir()
	scripts := map[string]string{
		"git": `#!/bin/sh
if [ "$1" = "worktree" ] && [ "$2" = "add" ]; then mkdir -p "$4"; fi
if [ "$1" = "-C" ]; then printf '%040d\n' 1; fi
`,
		"go": "#!/bin/sh\nexit 0\n",
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	path := filepath.Join(dir, "samples.jsonl")
	// The build stub deliberately omits bal so acquisition stops after metadata.
	err := run(config{baseRef: "base", headRef: "head", repeats: 1, samplesJSONL: path, phase: "final"})
	if err == nil || !strings.Contains(err.Error(), "failed to start service") {
		t.Fatalf("run error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var metadata sampleMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Suite != "http" || metadata.Unit != "req/s" {
		t.Fatalf("HTTP metadata: %+v", metadata)
	}
}
