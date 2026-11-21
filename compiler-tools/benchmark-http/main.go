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
	"math"
	"os"
	"path/filepath"
	"time"
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
	samplesJSONL     string
	phase            string
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
		if cfg.samplesJSONL != "" {
			os.Exit(1)
		}
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
	fs.StringVar(&cfg.samplesJSONL, "samples-jsonl", "", "Raw paired sample stream")
	fs.StringVar(&cfg.phase, "phase", "fixed", "Sample phase: fixed, pilot, final")
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
	if err := cfg.validate(); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func (cfg config) validate() error {
	if cfg.repeats <= 0 || cfg.conns <= 0 {
		return fmt.Errorf("repeats and connections must be positive")
	}
	if math.IsNaN(cfg.threshold) || math.IsInf(cfg.threshold, 0) || cfg.threshold <= 0 || cfg.threshold >= 100 {
		return fmt.Errorf("threshold must be finite and in (0,100)")
	}
	if cfg.phase != "fixed" && cfg.phase != "pilot" && cfg.phase != "final" {
		return fmt.Errorf("invalid phase %q", cfg.phase)
	}
	for _, value := range []string{cfg.warmup, cfg.duration} {
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			return fmt.Errorf("invalid duration %q", value)
		}
	}
	return nil
}

func run(cfg config) error {
	pruneWorktrees()

	// Each resource registers its teardown as it is created, so an interrupt
	// tears down exactly what exists at that moment.
	var c cleanups
	defer c.run()
	defer onInterrupt(c.run)()

	workRoot, err := os.MkdirTemp("", "httpbench-work-*")
	if err != nil {
		return err
	}
	c.add(func() { _ = os.RemoveAll(workRoot) })

	helloDir, err := os.MkdirTemp("", "httpbench-hello-*")
	if err != nil {
		return err
	}
	c.add(func() { _ = os.RemoveAll(helloDir) })
	helloFile := filepath.Join(helloDir, "hello.bal")
	if err := os.WriteFile(helloFile, helloSource, 0o644); err != nil {
		return err
	}

	baseWT, err := checkoutWorktree(workRoot, "base", cfg.baseRef)
	if err != nil {
		return err
	}
	c.add(func() { removeWorktree(baseWT) })
	headWT, err := checkoutWorktree(workRoot, "head", cfg.headRef)
	if err != nil {
		return err
	}
	c.add(func() { removeWorktree(headWT) })

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

	var writer *sampleWriter
	if cfg.samplesJSONL != "" {
		base, err := resolveSampleSHA(baseWT)
		if err != nil {
			return err
		}
		head, err := resolveSampleSHA(headWT)
		if err != nil {
			return err
		}
		writer, err = newSampleWriter(cfg.samplesJSONL, sampleMetadata{Type: "metadata", Version: 1, BaseSHA: base, HeadSHA: head, Suite: "http", Phase: cfg.phase, Cases: []string{"hello"}, Counts: map[string]int{"hello": cfg.repeats}, Unit: "req/s"})
		if err != nil {
			return err
		}
		defer func() { _ = writer.close(false) }()
	}
	samples, err := collectHTTPPairs(cfg, writer, func(role int) (sample, error) {
		s, err := measureOnce([]string{baseBal, headBal}[role], helloFile, cfg, &c)
		if err != nil {
			return s, fmt.Errorf("measuring %s: %w", []string{cfg.baseRef, cfg.headRef}[role], err)
		}
		return s, nil
	})
	if err != nil {
		return err
	}
	baseS, headS := samples[0], samples[1]

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

func collectHTTPPairs(cfg config, writer *sampleWriter, measure func(int) (sample, error)) ([][]sample, error) {
	samples := make([][]sample, 2)
	for pair := 0; pair < cfg.repeats; pair++ {
		for position := 0; position < 2; position++ {
			role := (pair + position) % 2
			s, err := measure(role)
			if err != nil {
				return nil, err
			}
			if s.throughput <= 0 || math.IsNaN(s.throughput) || math.IsInf(s.throughput, 0) {
				return nil, fmt.Errorf("invalid throughput")
			}
			samples[role] = append(samples[role], s)
			if writer != nil {
				if err := writer.write(sampleEvent{CaseID: "hello", Pair: pair, Role: []string{"base", "head"}[role], Position: position, Value: s.throughput, Unit: "req/s", Metrics: map[string]float64{"avg_ms": s.avgMs, "stdev_ms": s.stdevMs, "max_ms": s.maxMs, "error_pct": s.errPct, "p99_ms": s.p99Ms, "startup_s": s.startupSec, "rss_mib": s.rssMB}}); err != nil {
					return nil, err
				}
			}
		}
	}
	if writer != nil {
		if err := writer.completeCase("hello"); err != nil {
			return nil, err
		}
		if err := writer.close(true); err != nil {
			return nil, err
		}
	}
	return samples, nil
}
