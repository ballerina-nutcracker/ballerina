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
)

// scalingExceededExitCode separates a failing scaling verdict from a tool or
// workload failure, which exits 1.
const scalingExceededExitCode = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if errors.Is(err, errScalingExceeded) {
			os.Exit(scalingExceededExitCode)
		}
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}

	b := &benchmark{config: *cfg}
	if err := b.run(); err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	return nil
}
