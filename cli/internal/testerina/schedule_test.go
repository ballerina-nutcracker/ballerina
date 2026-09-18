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

package testerina

import "testing"

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    Filter
		wantErr bool
	}{
		{name: "empty matches all", spec: "", want: nil},
		{name: "blank matches all", spec: "   ", want: nil},
		{name: "entries are trimmed", spec: " a , b:c ", want: Filter{"a", "b:c"}},
		{name: "empty entry rejected", spec: "a,,b", wantErr: true},
		{name: "trailing comma rejected", spec: "a,", wantErr: true},
		{name: "empty name pattern rejected", spec: "app:", wantErr: true},
		{name: "empty module pattern rejected", spec: ":testOne", wantErr: true},
		{name: "extra colon rejected", spec: "app:testOne:extra", wantErr: true},
		{name: "malformed entry rejects whole spec", spec: "good,app:", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilter(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseFilter(%q) = %v, want an error", tt.spec, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFilter(%q) failed: %v", tt.spec, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseFilter(%q) = %v, want %v", tt.spec, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ParseFilter(%q) = %v, want %v", tt.spec, got, tt.want)
				}
			}
		})
	}
}

func TestFilterMatches(t *testing.T) {
	defaultModule := ModuleKey{Org: "acme", Module: "app"}
	subModule := ModuleKey{Org: "acme", Module: "app.sub"}

	tests := []struct {
		name   string
		spec   string
		module ModuleKey
		test   string
		want   bool
	}{
		{name: "empty filter matches", spec: "", module: defaultModule, test: "testOne", want: true},
		{name: "bare name matches any module", spec: "testOne", module: subModule, test: "testOne", want: true},
		{name: "bare name rejects others", spec: "testOne", module: defaultModule, test: "testTwo", want: false},
		{name: "trailing wildcard", spec: "test*", module: defaultModule, test: "testOne", want: true},
		{name: "leading wildcard", spec: "*One", module: defaultModule, test: "testOne", want: true},
		{name: "interior wildcard", spec: "test*One", module: defaultModule, test: "testTheOne", want: true},
		{name: "interior wildcard rejects", spec: "test*One", module: defaultModule, test: "testTwo", want: false},
		{name: "bare star matches", spec: "*", module: subModule, test: "anything", want: true},
		{name: "star colon star matches", spec: "*:*", module: subModule, test: "anything", want: true},
		{name: "module qualified on default module", spec: "app:testOne", module: defaultModule, test: "testOne", want: true},
		{name: "submodule uses bare name", spec: "sub:testOne", module: subModule, test: "testOne", want: true},
		{name: "submodule not matched by package name", spec: "app:testOne", module: subModule, test: "testOne", want: false},
		{name: "module wildcard", spec: "s*b:test*", module: subModule, test: "testOne", want: true},
		{name: "any entry may match", spec: "nope,testOne", module: defaultModule, test: "testOne", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := ParseFilter(tt.spec)
			if err != nil {
				t.Fatal(err)
			}
			if got := filter.Matches(tt.module, tt.test); got != tt.want {
				t.Fatalf("Matches(%v, %q) with %q = %v, want %v", tt.module, tt.test, tt.spec, got, tt.want)
			}
		})
	}
}

func TestNewScheduleKeepsDisabledMatchesAndDropsRest(t *testing.T) {
	plan := TestPlan{Modules: []ModuleTests{
		{Key: ModuleKey{Org: "acme", Module: "app"}, Tests: []TestFunction{
			{Name: "testOne", Enabled: true},
			{Name: "testTwo", Enabled: false},
		}},
		{Key: ModuleKey{Org: "acme", Module: "app.sub"}, Tests: []TestFunction{
			{Name: "testThree", Enabled: true},
		}},
	}}

	filter, err := ParseFilter("test*o,sub:*")
	if err != nil {
		t.Fatal(err)
	}
	schedule := NewSchedule(plan, filter)
	if len(schedule.Steps) != 2 {
		t.Fatalf("steps = %#v, want testTwo and testThree", schedule.Steps)
	}
	if schedule.Steps[0].Test.Name != "testTwo" || schedule.Steps[0].Test.Enabled {
		t.Fatalf("first step = %#v, want the disabled testTwo", schedule.Steps[0])
	}
	if schedule.Steps[1].Test.Name != "testThree" {
		t.Fatalf("second step = %#v, want testThree", schedule.Steps[1])
	}
}
