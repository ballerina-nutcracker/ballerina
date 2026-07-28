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

// Command httpbench compares HTTP throughput of the Ballerina Nutcracker
// interpreter between two git refs. For each ref it checks out a worktree,
// builds `bal`, runs the embedded hello service via `bal run`, and drives load
// with wrk — interleaving repeated runs of base and head on the same host so
// environmental noise is shared. It writes a markdown report and a result.json
// verdict (used by the CI gate). Regression is data, not an error: the tool
// exits 0 unless it cannot produce a measurement.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed testdata/hello.bal
var helloSource []byte

type config struct {
	baseRef, headRef string
	repeats          int
	warmup, duration string // wrk -d values, e.g. "3s", "10s"
	conns            int
	threshold        float64 // percent throughput drop that counts as a regression
	exportMD         string
	resultJSON       string
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		os.Exit(2)
	}

	if err := run(cfg); err != nil {
		// Operational failure (build/launch/parse) — emit an error report so the
		// PR comment still has something, and let the gate fail on error:true.
		fmt.Fprintf(os.Stderr, "httpbench: %v\n", err)
		emitError(cfg, err)
		// Exit 0: the artifact + gate handle the failure; a crash here would
		// skip artifact upload and leave no comment.
		return
	}
}

// parseConfig parses flags and the two positional refs out of a local
// FlagSet (rather than the global flag.CommandLine) so it can be unit tested
// with arbitrary argument slices without touching global state or exiting.
func parseConfig(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("httpbench", flag.ContinueOnError)
	fs.IntVar(&cfg.repeats, "repeats", 2, "measured runs per ref (interleaved base/head)")
	fs.StringVar(&cfg.warmup, "warmup", "30s", "wrk warmup duration (discarded)")
	fs.StringVar(&cfg.duration, "duration", "330s", "wrk measured duration")
	fs.IntVar(&cfg.conns, "conns", 50, "wrk concurrent connections")
	fs.Float64Var(&cfg.threshold, "threshold", 10, "throughput drop % that fails the gate")
	fs.StringVar(&cfg.exportMD, "export-md", "", "write the markdown report to this path")
	fs.StringVar(&cfg.resultJSON, "result-json", "", "write the JSON verdict to this path")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: httpbench [flags] <base-ref> <head-ref>\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return config{}, fmt.Errorf("expected exactly 2 positional args (base-ref, head-ref), got %d", fs.NArg())
	}
	cfg.baseRef, cfg.headRef = fs.Arg(0), fs.Arg(1)
	return cfg, nil
}

func run(cfg config) error {
	workRoot, err := os.MkdirTemp("", "httpbench-work-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workRoot)

	helloDir, err := os.MkdirTemp("", "httpbench-hello-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(helloDir)
	helloFile := filepath.Join(helloDir, "hello.bal")
	if err := os.WriteFile(helloFile, helloSource, 0o644); err != nil {
		return err
	}

	baseWT, err := checkoutWorktree(workRoot, "base", cfg.baseRef)
	if err != nil {
		return err
	}
	defer removeWorktree(baseWT)
	headWT, err := checkoutWorktree(workRoot, "head", cfg.headRef)
	if err != nil {
		return err
	}
	defer removeWorktree(headWT)

	fmt.Fprintf(os.Stderr, "Building bal for %s...\n", cfg.baseRef)
	baseBal, err := buildInterpreter(baseWT)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Building bal for %s...\n", cfg.headRef)
	headBal, err := buildInterpreter(headWT)
	if err != nil {
		return err
	}

	var baseS, headS []sample
	for i := 0; i < cfg.repeats; i++ {
		fmt.Fprintf(os.Stderr, "Repeat %d/%d: %s...\n", i+1, cfg.repeats, cfg.baseRef)
		s, err := measureOnce(baseBal, helloFile, cfg)
		if err != nil {
			return fmt.Errorf("measuring %s: %w", cfg.baseRef, err)
		}
		baseS = append(baseS, s)

		fmt.Fprintf(os.Stderr, "Repeat %d/%d: %s...\n", i+1, cfg.repeats, cfg.headRef)
		s, err = measureOnce(headBal, helloFile, cfg)
		if err != nil {
			return fmt.Errorf("measuring %s: %w", cfg.headRef, err)
		}
		headS = append(headS, s)
	}

	md, res := buildReport(cfg.baseRef, cfg.headRef, baseS, headS, cfg)
	fmt.Println(md)
	return emit(cfg, md, res)
}

func emit(cfg config, md string, res result) error {
	if cfg.exportMD != "" {
		if err := writeFile(cfg.exportMD, md); err != nil {
			return err
		}
	}
	if cfg.resultJSON != "" {
		if err := writeResultJSON(cfg.resultJSON, res); err != nil {
			return err
		}
	}
	return nil
}

func emitError(cfg config, cause error) {
	md := fmt.Sprintf("HTTP benchmark could not complete:\n\n```text\n%v\n```\n", cause)
	res := result{Error: true, Message: cause.Error(), ThresholdPct: cfg.threshold}
	if cfg.exportMD != "" {
		_ = writeFile(cfg.exportMD, md)
	}
	if cfg.resultJSON != "" {
		_ = writeResultJSON(cfg.resultJSON, res)
	}
}
