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

//go:build debug

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/ballerina-nutcracker/ballerina/context"

	"github.com/spf13/cobra"
)

const (
	traceFlagName        = "trace"
	nestedFlagName       = "nested"
	defaultTraceFileName = "traces.json"
)

// traceFlagValue holds the --trace value: the bare flag selects
// defaultTraceFileName, "--trace=<path>" supplies its own.
var traceFlagValue string

// nestedFlagValue holds the --nested value, which records the children each
// phase starts instead of the phase spans alone.
var nestedFlagValue bool

func registerTraceFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&traceFlagValue, traceFlagName, "",
		"Write frontend compilation traces as Chrome Trace Event JSON (default "+defaultTraceFileName+")")
	cmd.Flags().Lookup(traceFlagName).NoOptDefVal = defaultTraceFileName
	cmd.Flags().BoolVar(&nestedFlagValue, nestedFlagName, false,
		"Record the child spans within each traced phase (requires --trace)")
}

// runTraceOptions reports how the recorder is configured and where its output
// goes. Relative paths are resolved against the process working directory.
func runTraceOptions(cmd *cobra.Command) (options context.TraceOptions, path string, err error) {
	nested := cmd.Flags().Lookup(nestedFlagName).Changed
	flag := cmd.Flags().Lookup(traceFlagName)
	if !flag.Changed {
		if nested {
			return context.TraceOptions{}, "", errors.New("--nested requires --trace")
		}
		return context.TraceOptions{}, "", nil
	}
	value := flag.Value.String()
	if value == "" {
		return context.TraceOptions{}, "", errors.New("--trace requires a path; use --trace or --trace=<path>")
	}
	absPath, err := filepath.Abs(value)
	if err != nil {
		return context.TraceOptions{}, "", fmt.Errorf("resolve trace path %s: %w", value, err)
	}
	return context.TraceOptions{Enabled: true, Nested: nestedFlagValue}, absPath, nil
}

// runTraceOutput owns trace output for one run: the destination path and
// whether the single write attempt has been made.
type runTraceOutput struct {
	path      string
	attempted bool
}

func newRunTraceOutput(path string) *runTraceOutput {
	return &runTraceOutput{path: path}
}

// finalize serializes and writes the recording at most once, and reports
// whether that attempt failed. A later call neither rewrites nor reports: the
// caller that made the attempt has already handled its result.
func (o *runTraceOutput) finalize(env *context.CompilerEnvironment) error {
	if o.path == "" || o.attempted {
		return nil
	}
	o.attempted = true
	return writeTrace(o.path, env)
}

// reportFailureDuringPanic writes the trace while a panic unwinds through the
// run, reporting a write failure without suppressing the original panic.
func (o *runTraceOutput) reportFailureDuringPanic(env *context.CompilerEnvironment) {
	if o.path == "" {
		return
	}
	recovered := recover()
	if recovered == nil {
		return
	}
	if err := o.finalize(env); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	// Re-panicking below unwinds from this deferred call, which would replace
	// the panicking frames in the reported stack, so print them while they are
	// still live.
	fmt.Fprintf(os.Stderr, "panic: %v\n\n%s\n", recovered, debug.Stack())
	panic(recovered)
}

func writeTrace(path string, env *context.CompilerEnvironment) error {
	data, err := env.TraceJSON()
	if err != nil {
		return fmt.Errorf("serialize trace: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create trace directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write trace %s: %w", path, err)
	}
	return nil
}
