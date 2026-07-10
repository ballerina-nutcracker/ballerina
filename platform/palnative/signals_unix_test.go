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

//go:build unix

package palnative

import (
	"os"
	"syscall"
	"testing"
	"time"

	"ballerina/platform/pal"
)

func TestSignalSourceMapsCtrlCToGracefulStop(t *testing.T) {
	source, cleanup := newSignalSource()
	defer cleanup()

	signalSelf(t, os.Interrupt)
	assertSignal(t, source.Signals, pal.GracefulStop)
}

func TestSignalSourceMapsUnixSignals(t *testing.T) {
	tests := []struct {
		name string
		os   os.Signal
		pal  pal.Signal
	}{
		{name: "sigterm", os: syscall.SIGTERM, pal: pal.GracefulStop},
		{name: "sigquit", os: syscall.SIGQUIT, pal: pal.ImmediateStop},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, cleanup := newSignalSource()
			defer cleanup()

			signalSelf(t, tt.os)
			assertSignal(t, source.Signals, tt.pal)
		})
	}
}

func signalSelf(t *testing.T, signal os.Signal) {
	t.Helper()
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Signal(signal); err != nil {
		t.Fatal(err)
	}
}

func assertSignal(t *testing.T, signals <-chan pal.Signal, want pal.Signal) {
	t.Helper()
	select {
	case got, ok := <-signals:
		if !ok {
			t.Fatal("signal channel closed before delivering PAL signal")
		}
		if got != want {
			t.Fatalf("expected %v, got %v", want, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for PAL signal")
	}
}
