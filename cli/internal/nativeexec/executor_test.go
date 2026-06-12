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

package nativeexec

import (
	"slices"
	"strings"
	"testing"
)

func TestAppendNativeMode_AddsWhenAbsent(t *testing.T) {
	env := []string{"HOME=/root", "PATH=/usr/bin"}
	result := AppendNativeMode(env)
	if !slices.Contains(result, "BAL_NATIVE=1") {
		t.Errorf("BAL_NATIVE=1 not found in result: %v", result)
	}
	for _, e := range env {
		if strings.HasPrefix(e, "BAL_NATIVE") {
			t.Error("original env slice must not be mutated")
		}
	}
}

func TestAppendNativeMode_ReplacesExisting(t *testing.T) {
	env := []string{"BAL_NATIVE=0", "HOME=/root"}
	result := AppendNativeMode(env)
	count := 0
	for _, e := range result {
		if strings.HasPrefix(e, "BAL_NATIVE=") {
			count++
			if e != "BAL_NATIVE=1" {
				t.Errorf("expected BAL_NATIVE=1, got %q", e)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one BAL_NATIVE entry, got %d in %v", count, result)
	}
}

func TestAppendNativeMode_NilEnv(t *testing.T) {
	result := AppendNativeMode(nil)
	if !slices.Contains(result, "BAL_NATIVE=1") {
		t.Errorf("BAL_NATIVE=1 not found for nil input: %v", result)
	}
}

func TestAppendNativeMode_DoesNotMutateOriginal(t *testing.T) {
	original := []string{"HOME=/root"}
	AppendNativeMode(original)
	for _, e := range original {
		if strings.HasPrefix(e, "BAL_NATIVE") {
			t.Error("original slice was mutated")
		}
	}
}
