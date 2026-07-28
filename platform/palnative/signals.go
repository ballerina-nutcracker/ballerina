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

package palnative

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"ballerina/platform/pal"
)

// newSignalSource installs an os/signal.Notify handler and returns a
// SignalSource exposing the OS-signal -> pal.Signal mapping for the
// native CLI runtime:
//
//	SIGINT, SIGTERM -> pal.GracefulStop
//	SIGQUIT         -> pal.ImmediateStop
//
// SIGKILL is intentionally not handled: it cannot be trapped by user
// processes.
func newSignalSource() (pal.SignalSource, func()) {
	osCh := make(chan os.Signal, 4)
	signal.Notify(osCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	out := make(chan pal.Signal, 2)
	done := make(chan struct{})
	var cleanupOnce sync.Once
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case sig := <-osCh:
				var palSig pal.Signal
				switch sig {
				case syscall.SIGINT, syscall.SIGTERM:
					palSig = pal.GracefulStop
				case syscall.SIGQUIT:
					palSig = pal.ImmediateStop
				default:
					continue
				}
				select {
				case out <- palSig:
				case <-done:
					return
				}
			}
		}
	}()
	return pal.SignalSource{Signals: out}, func() {
		cleanupOnce.Do(func() {
			signal.Stop(osCh)
			close(done)
		})
	}
}
