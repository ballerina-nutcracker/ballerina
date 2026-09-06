// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"errors"
	"io/fs"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

func TestModuleSchedulingCancellationJoinsStartedReaders(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var active atomic.Int64
	fsys := fstest.MapFS{"main.bal": &fstest.MapFile{Data: []byte("function main() {}")}}
	modules := []*ModuleSources{{}, {}}
	checks := 0
	result := make(chan error, 1)
	go func() {
		result <- runModuleParsers(modules, func() error {
			checks++
			if checks == 1 {
				return nil
			}
			<-started
			return context.Canceled
		}, func(_ int, _ *ModuleSources) {
			active.Add(1)
			close(started)
			<-release
			_, _ = fs.ReadFile(fsys, "main.bal")
			active.Add(-1)
		})
	}()
	<-started
	select {
	case err := <-result:
		t.Fatalf("scheduler returned before joining the blocked reader: %v", err)
	default:
	}
	close(release)
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got := active.Load(); got != 0 {
		t.Fatalf("active readers after return = %d, want 0", got)
	}
}
