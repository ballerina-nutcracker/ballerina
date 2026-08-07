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

// Package nativeexec defines the interface for building and executing a custom
// interpreter binary that embeds Go-native function implementations from external
// Ballerina packages.
package nativeexec

import (
	"context"
	"io"
	"io/fs"
	"os"
	"strings"
)

const envNativeMode = "BAL_NATIVE"

// InNativeMode reports whether the current process was launched as a native
// runner (BAL_NATIVE=1 in the environment). When true, native detection is
// skipped to prevent infinite re-execution.
func InNativeMode() bool {
	return os.Getenv(envNativeMode) == "1"
}

// AppendNativeMode returns env with BAL_NATIVE=1 appended or updated in place.
func AppendNativeMode(env []string) []string {
	for i, e := range env {
		if strings.HasPrefix(e, envNativeMode+"=") {
			result := make([]string, len(env))
			copy(result, env)
			result[i] = envNativeMode + "=1"
			return result
		}
	}
	return append(append([]string(nil), env...), envNativeMode+"=1")
}

// NativeExecutor builds a custom interpreter binary that includes Go-native
// function implementations from external Ballerina packages and returns a Runner
// that can re-execute the program with those implementations registered.
type NativeExecutor interface {
	// Available reports whether this executor can build native runners in the
	// current environment.
	Available() bool
	// Prepare compiles a custom interpreter binary that embeds the native
	// sources described by req.Payload. Returns a Runner whose Run method
	// re-executes the current program via the compiled binary.
	Prepare(ctx context.Context, req NativeRunnerRequest) (Runner, error)
	// Build is like Prepare but returns a bare binary path instead of a
	// Runner — for bal build, which hands that path to executable.Pack
	// rather than re-executing into it.
	Build(ctx context.Context, req NativeRunnerRequest) (string, error)
}

// Runner re-executes the program via a previously-built native interpreter binary.
type Runner interface {
	// Run executes the native interpreter and returns its exit code.
	// The caller should propagate the code to os.Exit.
	Run(ctx context.Context) (ExitCode, error)
	// Close frees temporary resources associated with this runner.
	Close() error
}

// ExitCode is the process exit status returned by Runner.Run.
type ExitCode int

// NativeRunnerRequest describes the native sources to embed and the invocation
// parameters to forward to the re-executed binary.
type NativeRunnerRequest struct {
	// Payloads contains the Go source files for each native Ballerina package
	// that must be compiled into the interpreter binary.
	Payloads []NativePayload
	// Stdout and Stderr are forwarded to the native interpreter's output streams.
	Stdout io.Writer
	Stderr io.Writer
	// Args is os.Args[1:] — the original command-line arguments to pass to the
	// re-executed binary.
	Args []string
	// Env is the environment for the re-executed binary (typically os.Environ()
	// with BAL_NATIVE=1 appended).
	Env []string
	// TargetOS/TargetArch cross-compile for a platform other than the host;
	// empty means build for the host (like GOOS/GOARCH). bal run always
	// leaves these empty; bal build sets them from --target-os/--target-arch.
	TargetOS   string
	TargetArch string
}

// NativePayload provides the Go source files that implement native Ballerina
// functions for a single external package.
type NativePayload interface {
	// FS returns an fs.FS whose root contains the Go source files.
	FS() fs.FS
	// GoModuleName returns the Go module path to use for the bundle module
	// (e.g., "ballerinax/redis-native").
	GoModuleName() string
}

// GoSourcePayload is a concrete NativePayload backed by an fs.FS.
type GoSourcePayload struct {
	GoFiles fs.FS
	Module  string
}

func (p *GoSourcePayload) FS() fs.FS            { return p.GoFiles }
func (p *GoSourcePayload) GoModuleName() string { return p.Module }
