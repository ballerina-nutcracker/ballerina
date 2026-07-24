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

package test_util

import (
	"sync"
	"time"

	"ballerina/platform/pal"
)

// TestSignalTimeout bounds how long a test PAL waits before forcing a
// graceful shutdown. Tests that never reach Listen() simply ignore it.
const TestSignalTimeout = 10 * time.Minute

// FailReporter is the subset of testing.TB needed to surface a forced
// shutdown back to the owning test. Decoupling from testing.TB keeps the
// PAL package import-graph trim.
type FailReporter interface {
	Errorf(format string, args ...any)
}

// NewTestSignalSource returns a SignalSource paired with the underlying
// channel and a cleanup function for the test harness. After `timeout` the
// source pushes a GracefulStop and reports the timeout via `reporter` (if
// non-nil). Cleanup stops the watchdog and closes the channel to release the
// runtime's signal goroutine.
func NewTestSignalSource(reporter FailReporter, timeout time.Duration) (pal.SignalSource, chan pal.Signal, func()) {
	ch := make(chan pal.Signal, 2)
	var mu sync.Mutex
	var closeOnce sync.Once
	var closed bool
	var timer *time.Timer
	cleanup := func() {
		closeOnce.Do(func() {
			mu.Lock()
			closed = true
			if timer != nil {
				timer.Stop()
			}
			close(ch)
			mu.Unlock()
		})
	}
	if timeout <= 0 {
		return pal.SignalSource{Signals: ch}, ch, cleanup
	}
	timer = time.AfterFunc(timeout, func() {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		if reporter != nil {
			reporter.Errorf("test PAL: forcing graceful shutdown after %s", timeout)
		}
		select {
		case ch <- pal.GracefulStop:
		default:
		}
	})
	return pal.SignalSource{Signals: ch}, ch, cleanup
}
