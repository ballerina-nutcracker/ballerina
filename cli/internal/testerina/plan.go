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

// Package testerina discovers and executes `@test:Config` functions.
package testerina

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// ModuleKey identifies a module by the org and the full dotted module name the
// runtime dispatches on (e.g. `foo.sub`).
type ModuleKey struct {
	Org    string
	Module string
}

// displayName is the name used for filtering and reporting: the submodule name
// alone. Root-package modules are always `<pkgName>` or `<pkgName>.<rest>`, so
// the default module keeps the package name and a submodule drops it.
func (k ModuleKey) displayName() string {
	if _, rest, found := strings.Cut(k.Module, "."); found {
		return rest
	}
	return k.Module
}

// FunctionRef names a top-level function. The zero value means "none".
type FunctionRef struct {
	Org    string
	Module string
	Name   string
}

func (f FunctionRef) IsZero() bool {
	return f == FunctionRef{}
}

// TestFunction is one discovered `@test:Config` function.
type TestFunction struct {
	Name     string
	Enabled  bool
	Before   FunctionRef
	After    FunctionRef
	Position diagnostics.Location
}

// ModuleTests holds the tests discovered in a single module, in source order.
type ModuleTests struct {
	Key   ModuleKey
	Tests []TestFunction
}

// TestPlan holds every discovered test, with modules in the root package's
// topological module order.
type TestPlan struct {
	Modules []ModuleTests
}
