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
	"flag"
	"fmt"
	"os"
)

type benchmarkMode string

const (
	timeMode    benchmarkMode = "time"
	memoryMode  benchmarkMode = "memory"
	scalingMode benchmarkMode = "scaling"
)

type config struct {
	baseRef     string
	headRef     string
	target      string
	mode        benchmarkMode
	warmup      int
	runs        int
	exportPath  string
	summaryPath string
	factor      int
	maxRatio    float64
}

func parseConfig() (*config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <base-ref> <head-ref> <target>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s -mode scaling [options] <ref> <target>\n", os.Args[0])
		fs.PrintDefaults()
	}

	mode := fs.String("mode", string(timeMode), "Benchmark mode: time, memory or scaling")
	warmup := fs.Int("warmup", 4, "Number of warmup iterations")
	runs := fs.Int("runs", 10, "Number of benchmark runs")
	exportPath := fs.String("export-html", "", "Path to export HTML report")
	summaryPath := fs.String("export-summary", "", "Path to export the Markdown summary report")
	factor := fs.Int("factor", 4, "Scaling mode: BAL_SCALE value compared against BAL_SCALE=1")
	maxRatio := fs.Float64("max-ratio", 8, "Scaling mode: largest allowed startup-adjusted time ratio")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	cfg := &config{
		mode:        benchmarkMode(*mode),
		warmup:      *warmup,
		runs:        *runs,
		exportPath:  *exportPath,
		summaryPath: *summaryPath,
		factor:      *factor,
		maxRatio:    *maxRatio,
	}
	args := fs.Args()
	if cfg.mode == scalingMode {
		if len(args) < 2 {
			fs.Usage()
			return nil, fmt.Errorf("missing required arguments")
		}
		cfg.headRef, cfg.target = args[0], args[1]
		return cfg, nil
	}
	if len(args) < 3 {
		fs.Usage()
		return nil, fmt.Errorf("missing required arguments")
	}
	cfg.baseRef, cfg.headRef, cfg.target = args[0], args[1], args[2]
	return cfg, nil
}

func (c *config) validate() error {
	if c.mode == "" {
		c.mode = timeMode
	}
	if c.mode != timeMode && c.mode != memoryMode && c.mode != scalingMode {
		return fmt.Errorf("mode must be one of: %s, %s, %s", timeMode, memoryMode, scalingMode)
	}
	if c.mode != scalingMode && c.baseRef == "" {
		return fmt.Errorf("baseRef is required")
	}
	if c.headRef == "" {
		return fmt.Errorf("headRef is required")
	}
	if c.target == "" {
		return fmt.Errorf("target is required")
	}
	if _, err := os.Stat(c.target); os.IsNotExist(err) {
		return fmt.Errorf("target does not exist: %s", c.target)
	}
	if c.warmup < 0 {
		return fmt.Errorf("warmup must be non-negative")
	}
	if c.runs <= 0 {
		return fmt.Errorf("runs must be greater than zero")
	}
	if c.mode == scalingMode {
		return c.validateScaling()
	}
	return nil
}

func (c *config) validateScaling() error {
	if c.factor < 2 {
		return fmt.Errorf("factor must be at least 2")
	}
	if c.maxRatio <= 1 {
		return fmt.Errorf("max-ratio must be greater than 1")
	}
	if c.exportPath != "" {
		return fmt.Errorf("export-html is not supported in scaling mode; use export-summary")
	}
	return nil
}
