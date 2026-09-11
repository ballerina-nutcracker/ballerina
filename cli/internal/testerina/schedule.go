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

import (
	"fmt"
	"strings"
)

// Step is one test to execute, together with the module that declares it.
type Step struct {
	Module ModuleKey
	Test   TestFunction
}

// Label is the `module:name` form used by --list and the reporters.
func (s Step) Label() string {
	return s.Module.displayName() + ":" + s.Test.Name
}

// Schedule is the flattened, filtered execution order.
type Schedule struct {
	Steps []Step
}

// Filter holds the parsed `--tests` entries. An empty Filter matches every test.
type Filter []string

// ParseFilter parses a comma-separated `--tests` value. Each entry is either
// `namePattern` or `modulePattern:namePattern`, where `*` matches any run of
// characters.
func ParseFilter(spec string) (Filter, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	entries := strings.Split(spec, ",")
	filter := make(Filter, 0, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			return nil, fmt.Errorf("invalid --tests value %q: empty entry", spec)
		}
		// A qualified entry must be exactly modulePattern:namePattern. Without
		// this, "app:", ":testOne" and "app:testOne:extra" parse into patterns
		// that can never match, so the command would succeed having silently
		// run no tests.
		if modulePattern, namePattern, qualified := strings.Cut(trimmed, ":"); qualified {
			switch {
			case modulePattern == "":
				return nil, fmt.Errorf(
					"invalid --tests entry %q: module pattern is empty, want modulePattern:namePattern", trimmed)
			case namePattern == "":
				return nil, fmt.Errorf(
					"invalid --tests entry %q: test name pattern is empty, want modulePattern:namePattern", trimmed)
			case strings.Contains(namePattern, ":"):
				return nil, fmt.Errorf(
					"invalid --tests entry %q: want at most one \":\", as modulePattern:namePattern", trimmed)
			}
		}
		filter = append(filter, trimmed)
	}
	return filter, nil
}

// Matches reports whether the named test in module is selected.
func (f Filter) Matches(module ModuleKey, name string) bool {
	if len(f) == 0 {
		return true
	}
	for _, entry := range f {
		modulePattern, namePattern, qualified := strings.Cut(entry, ":")
		if !qualified {
			if matchesPattern(entry, name) {
				return true
			}
			continue
		}
		if matchesPattern(modulePattern, module.displayName()) && matchesPattern(namePattern, name) {
			return true
		}
	}
	return false
}

// matchesPattern matches value against a pattern whose only metacharacter is
// `*`, standing for any run of characters.
func matchesPattern(pattern, value string) bool {
	segments := strings.Split(pattern, "*")
	if len(segments) == 1 {
		return pattern == value
	}
	if !strings.HasPrefix(value, segments[0]) {
		return false
	}
	value = value[len(segments[0]):]
	last := segments[len(segments)-1]
	for _, segment := range segments[1 : len(segments)-1] {
		index := strings.Index(value, segment)
		if index < 0 {
			return false
		}
		value = value[index+len(segment):]
	}
	return strings.HasSuffix(value, last) && len(value) >= len(last)
}

// NewSchedule flattens plan into steps, dropping tests the filter rejects.
// Matching but disabled tests stay in the schedule and are reported skipped.
func NewSchedule(plan TestPlan, filter Filter) Schedule {
	var schedule Schedule
	for _, module := range plan.Modules {
		for _, test := range module.Tests {
			if !filter.Matches(module.Key, test.Name) {
				continue
			}
			schedule.Steps = append(schedule.Steps, Step{Module: module.Key, Test: test})
		}
	}
	return schedule
}
