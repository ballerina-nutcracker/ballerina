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
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	servicePort = 9090
	serviceURL  = "http://127.0.0.1:9090/hello"
)

// sample is one measured run of one ref: the wrk metrics plus the startup
// time and peak RSS captured around it.
type sample struct {
	wrkMetrics
	startupSec float64
	rssMB      float64 // 0 when unavailable (e.g. non-Linux dev host)
}

// --- git worktree + interpreter build (adapted from compiler-tools/benchmark) ---

func checkoutWorktree(workRoot, role, ref string) (string, error) {
	path := filepath.Join(workRoot, "worktree-"+role+"-"+sanitize(ref))
	removeWorktree(path)
	if err := runCmd(".", "git", "worktree", "add", "--detach", path, ref); err != nil {
		return "", fmt.Errorf("failed to checkout worktree for ref %q: %w", ref, err)
	}
	return path, nil
}

func removeWorktree(path string) {
	_ = runCmdSilent(".", "git", "worktree", "remove", "--force", path)
}

// buildInterpreter builds `bal` from the worktree and returns its absolute path.
func buildInterpreter(worktree string) (string, error) {
	out := "bal"
	if runtime.GOOS == "windows" {
		out = "bal.exe"
	}
	if err := runCmd(worktree, "go", "build", "-o", out, "./cli/cmd"); err != nil {
		return "", fmt.Errorf("failed to build interpreter in %q: %w", worktree, err)
	}
	abs, err := filepath.Abs(filepath.Join(worktree, out))
	if err != nil {
		return "", err
	}
	return abs, nil
}

func sanitize(ref string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':':
			return '-'
		default:
			return r
		}
	}, ref)
}

func runCmd(dir, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stdout = os.Stderr // build/log noise goes to stderr, keeping stdout clean
	c.Stderr = os.Stderr
	return c.Run()
}

func runCmdSilent(dir, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	return c.Run()
}

// --- service lifecycle + load ---

// measureOnce launches `bal run hello.bal`, times startup, drives a warmup and
// a measured wrk run, captures peak RSS, and tears the service down.
func measureOnce(balPath, helloFile string, cfg config) (sample, error) {
	var s sample
	// Fail fast if the port is already taken: otherwise waitForPort would latch
	// onto a stale/foreign listener and we would measure the wrong service.
	if portOpen(servicePort) {
		return s, fmt.Errorf("port %d is already in use before launch; a prior service may not have shut down", servicePort)
	}
	cmd := exec.Command(balPath, "run", helloFile)
	setProcAttr(cmd) // own process group (unix) so the whole tree can be killed
	cmd.Stdout = nil
	cmd.Stderr = nil
	start := time.Now()
	if err := cmd.Start(); err != nil {
		return s, fmt.Errorf("failed to start service: %w", err)
	}
	defer func() {
		killGroup(cmd)
		waitPortClose(servicePort, 15*time.Second)
	}()

	if !waitForPort(servicePort, 60*time.Second) {
		return s, fmt.Errorf("service did not open port %d within 60s", servicePort)
	}
	s.startupSec = time.Since(start).Seconds()

	threads := cfg.conns
	if n := runtime.NumCPU(); n < threads {
		threads = n
	}

	if _, err := runWrk(threads, cfg.conns, cfg.warmup); err != nil {
		return s, fmt.Errorf("warmup wrk failed: %w", err)
	}
	out, err := runWrk(threads, cfg.conns, cfg.duration)
	if err != nil {
		return s, fmt.Errorf("measured wrk failed: %w", err)
	}
	m, err := parseWrk(out)
	if err != nil {
		return s, fmt.Errorf("parsing wrk output: %w\n---\n%s", err, out)
	}
	s.wrkMetrics = m
	s.rssMB = readPeakRSS(cmd.Process.Pid)
	return s, nil
}

func runWrk(threads, conns int, dur string) (string, error) {
	d, err := time.ParseDuration(dur)
	if err != nil {
		return "", fmt.Errorf("invalid wrk duration %q: %w", dur, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), d+30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wrk",
		"-t", strconv.Itoa(threads),
		"-c", strconv.Itoa(conns),
		"-d", dur,
		"--latency",
		serviceURL,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// portOpen reports whether something is already accepting TCP connections on
// the port.
func portOpen(port int) bool {
	c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func waitForPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if portOpen(port) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func waitPortClose(port int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !portOpen(port) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// readPeakRSS reads VmHWM (peak resident set) for pid from /proc, in MB.
// Nutcracker's `bal run` serves in-process, so pid is the server. Returns 0
// when /proc is unavailable (e.g. a macOS dev host).
func readPeakRSS(pid int) float64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmHWM:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				if kb, err := strconv.ParseFloat(f[1], 64); err == nil {
					return kb / 1024
				}
			}
		}
	}
	return 0
}
