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
	"cmp"
	"sort"
	"strconv"
	"strings"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func (c *Context) addRootDiagnostics(values []diagnostics.Diagnostic) {
	c.addDiagnostics(scopeRoot, PackageDescriptor{}, ModuleDescriptor{}, "", false, 0, values)
}

func (c *Context) addPackageDiagnostics(pkg PackageDescriptor, values []diagnostics.Diagnostic) {
	c.addDiagnostics(scopePackage, pkg, ModuleDescriptor{}, "", false, 0, values)
}

func (c *Context) addModuleDiagnostics(module ModuleDescriptor, stage compilercontext.CompilationStage,
	test bool, document int, values []diagnostics.Diagnostic,
) {
	c.addDiagnostics(scopeModule, PackageDescriptor{}, module, stage, test, document, values)
}

func (c *Context) addDiagnostics(scope diagnosticScope, pkg PackageDescriptor, module ModuleDescriptor,
	stage compilercontext.CompilationStage, test bool, document int, values []diagnostics.Diagnostic,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, diagnostic := range values {
		c.nextOrdinal++
		c.diagnostics = append(c.diagnostics, diagnosticEntry{
			diagnostic: diagnostic, scope: scope, pkg: pkg, module: module, stage: stage,
			test: test, document: document, ordinal: c.nextOrdinal,
		})
	}
}

func (c *Context) sortedDiagnosticEntries() []diagnosticEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := append([]diagnosticEntry(nil), c.diagnostics...)
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.scope != b.scope {
			return a.scope < b.scope
		}
		switch a.scope {
		case scopeRoot:
			return a.ordinal < b.ordinal
		case scopePackage:
			if value := comparePackage(a.pkg, b.pkg); value != 0 {
				return value < 0
			}
		case scopeModule:
			ai, bi := c.moduleIndexLocked(a.module), c.moduleIndexLocked(b.module)
			if ai != bi {
				return ai < bi
			}
			if value := cmp.Compare(stageIndex(a.stage), stageIndex(b.stage)); value != 0 {
				return value < 0
			}
			if a.test != b.test {
				return !a.test
			}
			if a.document != b.document {
				return a.document < b.document
			}
		}
		al, bl := a.diagnostic.Location(), b.diagnostic.Location()
		if al.StartOffset() != bl.StartOffset() {
			return al.StartOffset() < bl.StartOffset()
		}
		if value := cmp.Compare(a.diagnostic.DiagnosticInfo().Code(), b.diagnostic.DiagnosticInfo().Code()); value != 0 {
			return value < 0
		}
		return a.ordinal < b.ordinal
	})
	return entries
}

func comparePackage(a, b PackageDescriptor) int {
	if value := cmp.Compare(a.Org, b.Org); value != 0 {
		return value
	}
	if value := cmp.Compare(a.Name, b.Name); value != 0 {
		return value
	}
	return compareSemanticVersion(a.Version, b.Version)
}

func compareSemanticVersion(a, b string) int {
	aCore, aPre, aOK := parseSemanticVersion(a)
	bCore, bPre, bOK := parseSemanticVersion(b)
	if !aOK || !bOK {
		return cmp.Compare(a, b)
	}
	for index := range aCore {
		if value := cmp.Compare(aCore[index], bCore[index]); value != 0 {
			return value
		}
	}
	if len(aPre) == 0 || len(bPre) == 0 {
		switch {
		case len(aPre) == len(bPre):
			return 0
		case len(aPre) == 0:
			return 1
		default:
			return -1
		}
	}
	for index := 0; index < min(len(aPre), len(bPre)); index++ {
		aNumber, aErr := strconv.Atoi(aPre[index])
		bNumber, bErr := strconv.Atoi(bPre[index])
		switch {
		case aErr == nil && bErr == nil:
			if value := cmp.Compare(aNumber, bNumber); value != 0 {
				return value
			}
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		default:
			if value := cmp.Compare(aPre[index], bPre[index]); value != 0 {
				return value
			}
		}
	}
	return cmp.Compare(len(aPre), len(bPre))
}

func parseSemanticVersion(value string) ([3]int, []string, bool) {
	var core [3]int
	value = strings.SplitN(value, "+", 2)[0]
	parts := strings.SplitN(value, "-", 2)
	coreParts := strings.Split(parts[0], ".")
	if len(coreParts) < 1 || len(coreParts) > len(core) {
		return core, nil, false
	}
	for index, part := range coreParts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return core, nil, false
		}
		core[index] = number
	}
	if len(parts) == 1 {
		return core, nil, true
	}
	if parts[1] == "" {
		return core, nil, false
	}
	return core, strings.Split(parts[1], "."), true
}

func stageIndex(stage compilercontext.CompilationStage) int {
	for index, candidate := range pipelineStages {
		if stage == candidate {
			return index
		}
	}
	return len(pipelineStages)
}
