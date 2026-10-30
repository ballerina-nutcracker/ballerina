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

// Package testplan is the stable, producer-independent contract describing
// every @test:* annotated function found in one module: what tests exist,
// their configuration, and the suite/group/each hooks that apply to them.
//
// This package deliberately has no dependency on the AST or any other
// discovery mechanism, and lives as its own module at the repo root (not
// under cli/internal) specifically so both cli and a root-module compiler
// plugin can import it — see lib/stdlibs/ballerina/test's compilerplugin
// package, the sole producer of a TestPlan (an earlier cli/internal
// AST-walk producer, testdiscovery, was retired once the plugin proved
// itself — see the implementation plan's Phase 3.4 notes). Every consumer
// of a TestPlan depends only on this package, not on how a TestPlan was
// produced, so swapping the producer again later wouldn't require changing
// them.
package testplan

// TestFunction is a discovered @test:Config-annotated function.
type TestFunction struct {
	Name            string
	Enable          bool
	Groups          []string
	DependsOn       []string // referenced function names
	Before          string   // referenced function name, "" if none
	After           string   // referenced function name, "" if none
	SerialExecution bool
	DataProvider    string // referenced function name, "" if none
}

// SuiteHook is a discovered @test:BeforeSuite/@test:BeforeEach/@test:AfterEach
// function — these are marker annotations with no configurable fields.
type SuiteHook struct {
	Name string
}

// AfterSuiteHook is a discovered @test:AfterSuite function.
type AfterSuiteHook struct {
	Name      string
	AlwaysRun bool
}

// GroupHook is a discovered @test:BeforeGroups/@test:AfterGroups function.
// AlwaysRun is only meaningful for AfterGroups.
type GroupHook struct {
	Name      string
	Groups    []string
	AlwaysRun bool
}

// TestPlan holds every @test:* annotated function found in one module.
type TestPlan struct {
	Tests        []TestFunction
	BeforeSuite  []SuiteHook
	AfterSuite   []AfterSuiteHook
	BeforeEach   []SuiteHook
	AfterEach    []SuiteHook
	BeforeGroups []GroupHook
	AfterGroups  []GroupHook
}

// HasAny reports whether anything was discovered at all — a module with no
// @test:* annotations anywhere (e.g. no test files, or test files with none)
// needs no ballerina/test registration glue at all.
func (p *TestPlan) HasAny() bool {
	return len(p.Tests) > 0 || len(p.BeforeSuite) > 0 || len(p.AfterSuite) > 0 ||
		len(p.BeforeEach) > 0 || len(p.AfterEach) > 0 || len(p.BeforeGroups) > 0 || len(p.AfterGroups) > 0
}
