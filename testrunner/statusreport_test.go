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

package testrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildModuleStatus(t *testing.T) {
	report := RunReport{
		Results: []Result{
			{Name: "testA", Status: Pass},
			{Name: "testSquare#one", Status: Pass, Type: DataDrivenMapOfTuple},
			{Name: "testB", Status: Fail, Message: "assertion failed"},
			{Name: "testC", Status: Skip},
		},
	}

	status := BuildModuleStatus(report)

	if status.TotalTests != 4 || status.Passed != 2 || status.Failed != 1 || status.Skipped != 1 {
		t.Fatalf("got %+v, want TotalTests=4 Passed=2 Failed=1 Skipped=1", status)
	}

	want := []TestStatusEntry{
		{Name: "testA", Status: "PASSED"},
		{Name: "testSquare#one", Status: "PASSED"}, // "#" is the data-key separator, deliberately not escaped
		{Name: "testB", Status: "FAILURE", FailureMessage: "assertion failed"},
		{Name: "testC", Status: "SKIPPED"},
	}
	if !reflect.DeepEqual(status.Tests, want) {
		t.Errorf("Tests = %+v, want %+v", status.Tests, want)
	}
}

// TestBuildModuleStatus_EscapesDelimiterCharactersInNames confirms a real
// special character (comma, a --tests list separator) does get escaped —
// distinguishing "# isn't special" (above) from "nothing is ever escaped."
func TestBuildModuleStatus_EscapesDelimiterCharactersInNames(t *testing.T) {
	report := RunReport{Results: []Result{{Name: "weird,name", Status: Pass}}}
	status := BuildModuleStatus(report)
	if got := status.Tests[0].Name; got == "weird,name" {
		t.Errorf("expected the comma to be escaped, got unescaped %q", got)
	}
}

func TestBuildModuleStatus_GroupsPassedThenFailedThenSkipped(t *testing.T) {
	// Deliberately out of execution order (fail, pass, skip, pass) — the
	// built Tests slice must still come back grouped pass-then-fail-then-skip,
	// matching ReportData's three separate arrays, not Results' own order.
	report := RunReport{
		Results: []Result{
			{Name: "first", Status: Fail, Message: "x"},
			{Name: "second", Status: Pass},
			{Name: "third", Status: Skip},
			{Name: "fourth", Status: Pass},
		},
	}
	status := BuildModuleStatus(report)

	var order []string
	for _, e := range status.Tests {
		order = append(order, e.Name)
	}
	want := []string{"second", "fourth", "first", "third"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("Tests order = %v, want %v", order, want)
	}
}

func TestWriteModuleStatusFile(t *testing.T) {
	status := ModuleStatus{
		TotalTests: 1, Passed: 1,
		Tests: []TestStatusEntry{{Name: "testA", Status: "PASSED"}},
	}
	path := filepath.Join(t.TempDir(), "cache", "tests_cache", "mymod", "module_status.json")

	if err := WriteModuleStatusFile(path, status); err != nil {
		t.Fatalf("WriteModuleStatusFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got ModuleStatus
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, status) {
		t.Errorf("round-tripped status = %+v, want %+v", got, status)
	}
}

func TestBuildRerunEntry(t *testing.T) {
	report := RunReport{
		Results: []Result{
			{Name: "testA", Status: Pass},
			{Name: "testSquare#one", Status: Fail, Type: DataDrivenMapOfTuple, Message: "x"},
			{Name: "testSquare#three", Status: Fail, Type: DataDrivenMapOfTuple, Message: "y"},
			{Name: "testAdd#0", Status: Fail, Type: DataDrivenTupleOfTuple, Message: "z"},
			{Name: "testB", Status: Fail, Message: "w"},
		},
	}

	entry := BuildRerunEntry(report, "mymod")

	wantNames := []string{"testSquare", "testSquare", "testAdd", "testB"}
	if !reflect.DeepEqual(entry.TestNames, wantNames) {
		t.Errorf("TestNames = %v, want %v (one entry per failed row, duplicates included)", entry.TestNames, wantNames)
	}

	for _, prefix := range []string{"testSquare", "testAdd", "testB"} {
		mod := entry.TestModuleNames[prefix]
		if mod == nil || *mod != "mymod" {
			t.Errorf("TestModuleNames[%q] = %v, want \"mymod\"", prefix, mod)
		}
	}

	wantSub := map[string][]string{
		"testSquare": {"'one'", "'three'"}, // map-shaped provider: single-quoted
		"testAdd":    {"0"},                // tuple-array provider: bare
	}
	if !reflect.DeepEqual(entry.SubTestNames, wantSub) {
		t.Errorf("SubTestNames = %+v, want %+v", entry.SubTestNames, wantSub)
	}
	if _, ok := entry.SubTestNames["testB"]; ok {
		t.Error("testB: expected no SubTestNames entry (not a data-driven test)")
	}
}

func TestWriteRerunEntry_MergesWithoutClobberingOtherModules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rerun_test.json")

	if err := WriteRerunEntry(path, "moduleA", RerunEntry{TestNames: []string{"testA"}}); err != nil {
		t.Fatalf("WriteRerunEntry(moduleA): %v", err)
	}
	if err := WriteRerunEntry(path, "moduleB", RerunEntry{TestNames: []string{"testB"}}); err != nil {
		t.Fatalf("WriteRerunEntry(moduleB): %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var all map[string]RerunEntry
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(all["moduleA"].TestNames, []string{"testA"}) {
		t.Errorf("moduleA entry was clobbered: %+v", all["moduleA"])
	}
	if !reflect.DeepEqual(all["moduleB"].TestNames, []string{"testB"}) {
		t.Errorf("moduleB entry missing or wrong: %+v", all["moduleB"])
	}
}

func TestSplitResultName(t *testing.T) {
	tests := []struct {
		name, wantPrefix, wantSuffix string
	}{
		{"testA", "testA", ""},
		{"testSquare#one", "testSquare", "one"},
	}
	for _, tc := range tests {
		prefix, suffix := splitResultName(tc.name)
		if prefix != tc.wantPrefix || suffix != tc.wantSuffix {
			t.Errorf("splitResultName(%q) = (%q, %q), want (%q, %q)", tc.name, prefix, suffix, tc.wantPrefix, tc.wantSuffix)
		}
	}
}
