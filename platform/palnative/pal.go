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

// Package palnative provides the native-CLI implementation of pal.Platform.
// The HTTP factory and its TLS plumbing live in http.go; IO is small enough
// to inline here. Other environments (e.g. WASM/web-editor) supply their own
// pal.Platform without importing this package.
package palnative

import (
	"bytes"
	"os"
	"os/exec"
	"os/user"
	"time"

	"ballerina-lang-go/platform/pal"
)

var processStart = time.Now()

// NewPlatform returns the native-CLI pal.Platform, wiring os.Stdout/Stderr for
// IO and NewHTTPClient for HTTP. The returned cleanup function releases signal
// resources owned by the platform.
func NewPlatform() (pal.Platform, func()) {
	signals, cleanupSignals := newSignalSource()
	return pal.Platform{
		IO: pal.IO{
			Stdout: func(p []byte) (n int, err error) { return os.Stdout.Write(p) },
			Stderr: func(p []byte) (n int, err error) { return os.Stderr.Write(p) },
		},
		FS: pal.FS{
			ReadFile: func(path string) ([]byte, error) {
				return os.ReadFile(path)
			},
			WriteFile: func(path string, data []byte) error {
				return os.WriteFile(path, data, 0o644)
			},
			AppendFile: func(path string, data []byte) (err error) {
				f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
				if err != nil {
					return err
				}
				defer func() {
					if cerr := f.Close(); cerr != nil && err == nil {
						err = cerr
					}
				}()
				_, err = f.Write(data)
				return err
			},
		},
		OS: pal.OS{
			GetEnv: func(name string) string {
				return os.Getenv(name)
			},
			GetUsername: func() string {
				u, err := user.Current()
				if err != nil {
					return ""
				}
				return u.Username
			},
			GetUserHome: func() string {
				home, err := os.UserHomeDir()
				if err != nil {
					return ""
				}
				return home
			},
			SetEnv: func(key, val string) error {
				return os.Setenv(key, val)
			},
			UnsetEnv: func(key string) error {
				return os.Unsetenv(key)
			},
			ListEnv: func() map[string]string {
				result := make(map[string]string)
				for _, e := range os.Environ() {
					for i := 0; i < len(e); i++ {
						if e[i] == '=' {
							result[e[:i]] = e[i+1:]
							break
						}
					}
				}
				return result
			},
			Exec: Exec,
		},
		Time: pal.Time{
			Now:          time.Now,
			MonotonicNow: func() time.Duration { return time.Since(processStart) },
		},
		HTTP: pal.HTTP{
			NewClient: NewHTTPClient,
		},
		Signals: signals,
	}, cleanupSignals
}

// Exec starts a subprocess and returns a handle to it. It is exported so test
// harnesses can wire real subprocess execution into an otherwise in-memory PAL.
func Exec(command string, args []string, envOverride map[string]string) (pal.ProcessHandle, error) {
	cmd := exec.Command(command, args...) //nolint:gosec
	if len(envOverride) > 0 {
		env := os.Environ()
		for k, v := range envOverride {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &nativeProcess{cmd: cmd, stdout: &stdout, stderr: &stderr}, nil
}

type nativeProcess struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (p *nativeProcess) WaitForExit() (int, error) {
	err := p.cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

func (p *nativeProcess) ReadStdout() ([]byte, error) {
	_ = p.cmd.Wait()
	return p.stdout.Bytes(), nil
}

func (p *nativeProcess) ReadStderr() ([]byte, error) {
	_ = p.cmd.Wait()
	return p.stderr.Bytes(), nil
}

func (p *nativeProcess) Kill() {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}
