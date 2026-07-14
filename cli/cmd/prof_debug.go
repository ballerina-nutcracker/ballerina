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
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Auto-registers pprof handlers
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"
)

type enabledProfiler struct {
	server  *http.Server
	cpuFile *os.File
	enabled bool
	addr    string
	cpuProf string
	memProf string
}

func init() {
	profiler = &enabledProfiler{}
	// Register profiler flags on the global packCmd; the createPackCmd factory
	// intentionally omits them so test-instantiated commands stay flag-free.
	profiler.RegisterFlags(packCmd)
}

func (p *enabledProfiler) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&p.enabled, "prof", false, "Enable profiling")
	cmd.Flags().StringVar(&p.addr, "prof-addr", ":6060", "Profiling server address")
	cmd.Flags().StringVar(&p.cpuProf, "cpuprofile", "", "Write CPU profile to file")
	cmd.Flags().StringVar(&p.memProf, "memprofile", "", "Write memory allocation profile to file")
}

func (p *enabledProfiler) Start() error {
	if p.cpuProf != "" {
		f, err := os.Create(p.cpuProf)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %w", err)
		}
		p.cpuFile = f
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return fmt.Errorf("could not start CPU profile: %w", err)
		}
		fmt.Fprintf(os.Stderr, "CPU profiling to %s\n", p.cpuProf)
	}

	if !p.enabled {
		return nil
	}

	// Create HTTP server (pprof handlers auto-registered by import)
	p.server = &http.Server{
		Addr:         p.addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		fmt.Fprintf(os.Stderr, "Profiling enabled at http://%s/debug/pprof/\n", p.addr)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Profiling server error: %v\n", err)
		}
	}()

	// Brief delay to catch immediate startup errors (port conflicts)
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (p *enabledProfiler) Stop() error {
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
		fmt.Fprintf(os.Stderr, "CPU profile written to %s\n", p.cpuProf)
	}

	if p.memProf != "" {
		f, err := os.Create(p.memProf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create memory profile: %v\n", err)
		} else {
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "could not write memory profile: %v\n", err)
			}
			f.Close()
			fmt.Fprintf(os.Stderr, "Memory profile written to %s\n", p.memProf)
		}
	}

	if p.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.server.Shutdown(ctx)
}
