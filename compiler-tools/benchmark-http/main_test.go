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
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigValid(t *testing.T) {
	t.Parallel()
	cfg, err := parseConfig([]string{
		"--repeats=3", "--warmup=5s", "--duration=7s", "--conns=20", "--threshold=15",
		"--export-md=/tmp/x.md", "--result-json=/tmp/x.json",
		"base-ref", "head-ref",
	})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	want := config{
		repeats: 3, warmup: "5s", duration: "7s", conns: 20, threshold: 15,
		exportMD: "/tmp/x.md", resultJSON: "/tmp/x.json",
		baseRef: "base-ref", headRef: "head-ref",
	}
	if cfg != want {
		t.Errorf("parseConfig() = %+v, want %+v", cfg, want)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := parseConfig([]string{"base", "head"})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.repeats != 2 || cfg.warmup != "30s" || cfg.duration != "330s" || cfg.conns != 50 || cfg.threshold != 10 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if cfg.baseRef != "base" || cfg.headRef != "head" {
		t.Errorf("unexpected refs: base=%q head=%q", cfg.baseRef, cfg.headRef)
	}
}

func TestParseConfigRequiresTwoPositionalArgs(t *testing.T) {
	t.Parallel()
	if _, err := parseConfig([]string{"only-one-ref"}); err == nil {
		t.Error("expected an error when fewer than 2 refs are given")
	}
	if _, err := parseConfig([]string{"a", "b", "c"}); err == nil {
		t.Error("expected an error when more than 2 refs are given")
	}
}

func TestParseConfigRejectsUnknownFlag(t *testing.T) {
	t.Parallel()
	if _, err := parseConfig([]string{"--nope", "a", "b"}); err == nil {
		t.Error("expected an error for an unknown flag")
	}
}

func TestEmitWritesBothOutputs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config{
		exportMD:   filepath.Join(dir, "report.md"),
		resultJSON: filepath.Join(dir, "result.json"),
	}
	res := result{Regressed: true, ThroughputDelta: -12.3}
	if err := emit(cfg, "# report", res); err != nil {
		t.Fatalf("emit: %v", err)
	}

	md, err := os.ReadFile(cfg.exportMD)
	if err != nil || string(md) != "# report" {
		t.Errorf("report.md = %q, err = %v", md, err)
	}

	data, err := os.ReadFile(cfg.resultJSON)
	if err != nil {
		t.Fatalf("reading result.json: %v", err)
	}
	var got result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal result.json: %v", err)
	}
	if got != res {
		t.Errorf("result.json = %+v, want %+v", got, res)
	}
}

func TestEmitSkipsUnsetPaths(t *testing.T) {
	t.Parallel()
	if err := emit(config{}, "ignored", result{}); err != nil {
		t.Fatalf("emit with no export paths should be a no-op, got: %v", err)
	}
}

func TestEmitPropagatesWriteErrors(t *testing.T) {
	t.Parallel()
	cfg := config{exportMD: filepath.Join(t.TempDir(), "missing-dir", "report.md")}
	if err := emit(cfg, "x", result{}); err == nil {
		t.Fatal("expected an error writing to a nonexistent directory")
	}
}

func TestEmitErrorWritesFailureReport(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config{
		exportMD:   filepath.Join(dir, "report.md"),
		resultJSON: filepath.Join(dir, "result.json"),
		threshold:  10,
	}
	emitError(cfg, fmt.Errorf("boom"))

	md, err := os.ReadFile(cfg.exportMD)
	if err != nil || !strings.Contains(string(md), "boom") {
		t.Errorf("report.md = %q, err = %v, want it to mention the cause", md, err)
	}

	data, err := os.ReadFile(cfg.resultJSON)
	if err != nil {
		t.Fatalf("reading result.json: %v", err)
	}
	var got result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal result.json: %v", err)
	}
	if !got.Error || got.Message != "boom" || got.ThresholdPct != 10 {
		t.Errorf("unexpected result.json contents: %+v", got)
	}
}

func TestRunFailsFastForInvalidBaseRef(t *testing.T) {
	t.Parallel()
	cfg := config{
		baseRef: "definitely-not-a-real-ref-xyz", headRef: "HEAD",
		repeats: 1, warmup: "1s", duration: "1s", conns: 2,
	}
	if err := run(cfg); err == nil {
		t.Fatal("expected an error for an invalid base ref")
	}
}

// TestRunEndToEndSelfComparison drives the full base->head pipeline (worktree
// checkout, bal build, service launch, wrk load, report+verdict) comparing
// HEAD against itself. It requires wrk on PATH — see requireWrk.
// Not parallel: run() binds the same hardcoded servicePort as
// TestMeasureOnceProducesSample.
func TestRunEndToEndSelfComparison(t *testing.T) {
	requireWrk(t)
	dir := t.TempDir()
	cfg := config{
		baseRef: "HEAD", headRef: "HEAD",
		repeats: 1, warmup: "1s", duration: "1s", conns: 4, threshold: 10,
		exportMD:   filepath.Join(dir, "report.md"),
		resultJSON: filepath.Join(dir, "result.json"),
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	md, err := os.ReadFile(cfg.exportMD)
	if err != nil || len(md) == 0 {
		t.Errorf("expected a non-empty report.md, err = %v", err)
	}

	data, err := os.ReadFile(cfg.resultJSON)
	if err != nil {
		t.Fatalf("reading result.json: %v", err)
	}
	var res result
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("unmarshal result.json: %v", err)
	}
	if res.Error {
		t.Errorf("unexpected error result: %+v", res)
	}
}
