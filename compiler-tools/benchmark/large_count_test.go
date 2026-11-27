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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Counts and measurement failures are host tool inputs, outside Ballerina corpus execution.
func TestLargeRunCountDoesNotAllocateBeforeMeasurement(t *testing.T) {
	t.Parallel()
	maxInt := int(^uint(0) >> 1)
	path := filepath.Join(t.TempDir(), "counts.json")
	data := fmt.Sprintf(`{"version":1,"counts":{"case":%d}}`, maxInt)
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	counts, err := loadRunCounts(path, []string{"case"}, 10)
	if err != nil {
		t.Fatalf("positive representable count rejected: %v", err)
	}
	sentinel := errors.New("measurement stopped immediately")
	calls := 0
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("accepted count panicked before controlled measurement failure: panic=%v, calls=%d", recovered, calls)
		}
	}()
	b := &benchmark{config: config{warmup: 0}}
	_, err = b.acquirePairs([]string{"base", "head"}, counts["case"], "case", "s", nil, func(int) (float64, error) {
		calls++
		return 0, sentinel
	})
	if !errors.Is(err, sentinel) || calls != 1 {
		t.Fatalf("got err=%v calls=%d; want immediate sentinel failure", err, calls)
	}
}
