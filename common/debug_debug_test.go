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

package common

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// Corpus tests cannot control overlapping debug scopes or recover their Go panics.
func TestWithSuppressedDebugOverlapping(t *testing.T) {
	// Debug configuration is global, so this test must not run in parallel.
	flags, writer := DebugFlags(), DebugWriter()
	t.Cleanup(func() { InitDebug(flags, writer) })
	var output bytes.Buffer
	InitDebug(DUMP_TOKENS, &output)

	firstReady, finishFirst := make(chan struct{}), make(chan struct{})
	firstDone := make(chan any, 1)
	nestedFlags := make(chan uint16, 1)
	go func() {
		defer func() { firstDone <- recover() }()
		WithSuppressedDebug(func() {
			close(firstReady)
			<-finishFirst
			WithSuppressedDebug(func() {
				nestedFlags <- DebugFlags()
				panic("suppressed panic")
			})
		})
	}()
	<-firstReady

	secondReady, finishSecond, secondDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(secondDone)
		WithSuppressedDebug(func() {
			close(secondReady)
			<-finishSecond
		})
	}()
	<-secondReady
	close(finishFirst)
	recovered := <-firstDone
	flagsWhileSecondActive := DebugFlags()
	enabledWhileSecondActive := DebugEnabled(DUMP_TOKENS)
	calledWhileSecondActive := false
	DebugWriteLazy(DUMP_TOKENS, func() string {
		calledWhileSecondActive = true
		return "suppressed"
	})
	close(finishSecond)
	<-secondDone

	if recovered != "suppressed panic" {
		t.Errorf("recovered %v, want suppressed panic", recovered)
	}
	if got := <-nestedFlags; got != 0 {
		t.Errorf("nested debug flags = %d, want 0", got)
	}
	if flagsWhileSecondActive != 0 || enabledWhileSecondActive || calledWhileSecondActive {
		t.Error("debug output became enabled while the second suppression was active")
	}
	if got := DebugFlags(); got != DUMP_TOKENS {
		t.Errorf("restored flags = %d, want %d", got, DUMP_TOKENS)
	}
	DebugWriteLazy(DUMP_TOKENS, func() string { return "restored" })
	if got := output.String(); got != "restored\n" {
		t.Errorf("debug output = %q, want restored newline", got)
	}
}

// Corpus tests cannot supply concurrent Go writers or reentrant debug callbacks.
func TestDebugWriteLazyConcurrent(t *testing.T) {
	// Debug configuration is global, so this test must not run in parallel.
	flags, writer := DebugFlags(), DebugWriter()
	t.Cleanup(func() { InitDebug(flags, writer) })
	var output bytes.Buffer
	InitDebug(DUMP_ST, &output)
	DebugWriteLazy(DUMP_ST, func() string {
		InitDebug(DUMP_ST, &output)
		DebugWriteLazy(DUMP_ST, func() string { return "inner" })
		return "outer"
	})
	if got := output.String(); got != "inner\nouter\n" {
		t.Fatalf("reentrant debug output = %q", got)
	}
	output.Reset()
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 20 {
				InitDebug(DUMP_ST, &output)
				DebugWriteLazy(DUMP_ST, func() string { return "message" })
			}
		})
	}
	workers.Wait()
	if got := output.String(); got != strings.Repeat("message\n", 160) {
		t.Errorf("concurrent output has %d messages, want 160", strings.Count(got, "message\n"))
	}
}
