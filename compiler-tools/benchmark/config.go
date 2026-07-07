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
	timeMode   benchmarkMode = "time"
	memoryMode benchmarkMode = "memory"
)

type config struct {
	baseRef    string
	headRef    string
	target     string
	mode       benchmarkMode
	warmup     int
	runs       int
	exportPath string
}

func parseConfig() (*config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <base-ref> <head-ref> <target>\n", os.Args[0])
		fs.PrintDefaults()
	}

	mode := fs.String("mode", string(timeMode), "Benchmark mode: time or memory")
	warmup := fs.Int("warmup", 4, "Number of warmup iterations")
	runs := fs.Int("runs", 10, "Number of benchmark runs")
	exportPath := fs.String("export-html", "", "Path to export HTML report")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	args := fs.Args()
	if len(args) < 3 {
		fs.Usage()
		return nil, fmt.Errorf("missing required arguments")
	}

	cfg := &config{
		baseRef:    args[0],
		headRef:    args[1],
		target:     args[2],
		mode:       benchmarkMode(*mode),
		warmup:     *warmup,
		runs:       *runs,
		exportPath: *exportPath,
	}
	return cfg, nil
}

func (c *config) validate() error {
	if c.baseRef == "" {
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
	if c.mode == "" {
		c.mode = timeMode
	}
	if c.mode != timeMode && c.mode != memoryMode {
		return fmt.Errorf("mode must be one of: %s, %s", timeMode, memoryMode)
	}
	if c.warmup < 0 {
		return fmt.Errorf("warmup must be non-negative")
	}
	if c.runs <= 0 {
		return fmt.Errorf("runs must be greater than zero")
	}
	return nil
}
