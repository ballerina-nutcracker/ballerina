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

// Package compilerplugin discovers @test:* annotated functions as a real
// compiler plugin (PR #921) — the sole producer of a testplan.TestPlan
// (cli/cmd/test.go's own direct caller, testrunner.Run, depends only on
// that package's stable contract, not on how a plan was produced). An
// earlier, separate pre-compilation AST walk (cli/internal/testdiscovery)
// existed alongside this package for a time, to prove the plugin produced
// field-identical results before cli/cmd/test.go's wiring switched over to
// it exclusively; it's since been retired (see the implementation plan's
// Phase 3.4 notes) — this package was never able to import it anyway,
// since Go's internal/ visibility rules close cli/internal off to anything
// outside the cli module, which is also why testplan itself lives at the
// repo root rather than under cli/internal.
//
// ProcessTests discovers and validates — it never transforms the AST. It
// stashes what it finds in testplan's process-local store rather than
// returning it, since a PackageTransformer's only output channel is the
// (here, unchanged) AST plus an error; see the design doc's "Go talking to
// Go doesn't need a contract" finding for why that's sufficient.
package compilerplugin

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/testplan"
)

const (
	orgName    = "ballerina"
	moduleName = "test"
)

// ProcessTests is the registered compiler plugin entry point (see
// CompilerPlugin.toml). It discovers every @test:* annotated function in
// pkg, validates dependsOn references, stashes the result in testplan's
// store keyed by pkg's module identity, and returns pkg unchanged.
func ProcessTests(compilerCtx *context.CompilerContext, _ model.ExportedSymbolSpace, pkg *ast.BLangPackage) (*ast.BLangPackage, error) {
	plan := discover(pkg)
	validateDependsOn(compilerCtx, pkg, plan)
	testplan.Store(moduleKey(pkg), plan)
	return pkg, nil
}

// validateDependsOn reports a compile error for any dependsOn entry that
// doesn't name another @test:Config function in the same module.
//
// Confirmed this is a real, not hypothetical, gap: running `bal test`
// against a dependsOn referencing a plain, non-@test:Config function fails
// at runtime with the opaque "The dependent test function is either
// disabled or not included." — register.bal's TestRegistry.getTestFunction
// only resolves against functions that were themselves registered via
// registerTestConfig, which only happens for @test:Config-annotated
// functions. This turns that runtime failure into a compile-time one with a
// message that actually names the function.
func validateDependsOn(compilerCtx *context.CompilerContext, pkg *ast.BLangPackage, plan *testplan.TestPlan) {
	testNames := make(map[string]bool, len(plan.Tests))
	for _, t := range plan.Tests {
		testNames[t.Name] = true
	}
	fnPositions := make(map[string]*ast.BLangFunction, len(pkg.Functions))
	for _, fn := range pkg.Functions {
		fnPositions[identifierValue(fn.Name)] = fn
	}
	for _, t := range plan.Tests {
		for _, dep := range t.DependsOn {
			if testNames[dep] {
				continue
			}
			compilerCtx.SemanticError(
				fmt.Sprintf("dependsOn function '%s' is not a @test:Config function", dep),
				fnPositions[t.Name].GetPosition(),
			)
		}
	}
}

func moduleKey(pkg *ast.BLangPackage) testplan.ModuleKey {
	if pkg.PackageID == nil {
		return testplan.ModuleKey{}
	}
	return testplan.ModuleKey{
		Org:    pkg.PackageID.OrgName.Value(),
		Module: pkg.PackageID.Name.Value(),
	}
}

// discover walks every top-level function in pkg and extracts @test:*
// annotation metadata — unreachable from the now-retired
// cli/internal/testdiscovery.Discover this once mirrored (see the package
// doc comment), so implemented directly here instead.
func discover(pkg *ast.BLangPackage) *testplan.TestPlan {
	if pkg == nil {
		return &testplan.TestPlan{}
	}
	alias := resolveTestAlias(pkg)

	p := &testplan.TestPlan{}
	for _, fn := range pkg.Functions {
		name := identifierValue(fn.Name)
		for _, att := range fn.GetAnnotationAttachments() {
			if !isTestAnnotation(att, alias) {
				continue
			}
			switch identifierValue(att.AnnotationName) {
			case "Config":
				p.Tests = append(p.Tests, parseTestConfig(name, att))
			case "BeforeSuite":
				p.BeforeSuite = append(p.BeforeSuite, testplan.SuiteHook{Name: name})
			case "AfterSuite":
				p.AfterSuite = append(p.AfterSuite, parseAfterSuiteConfig(name, att))
			case "BeforeEach":
				p.BeforeEach = append(p.BeforeEach, testplan.SuiteHook{Name: name})
			case "AfterEach":
				p.AfterEach = append(p.AfterEach, testplan.SuiteHook{Name: name})
			case "BeforeGroups":
				p.BeforeGroups = append(p.BeforeGroups, parseGroupsHook(name, att, false))
			case "AfterGroups":
				p.AfterGroups = append(p.AfterGroups, parseGroupsHook(name, att, true))
			}
		}
	}
	return p
}

// resolveTestAlias returns the alias ballerina/test was imported under in
// pkg, or the default alias "test" if pkg.Imports doesn't say.
func resolveTestAlias(pkg *ast.BLangPackage) string {
	if alias := testImportAlias(pkg); alias != "" {
		return alias
	}
	return moduleName
}

func testImportAlias(pkg *ast.BLangPackage) string {
	for _, imp := range pkg.Imports {
		if imp.OrgName == nil || imp.OrgName.GetValue() != orgName {
			continue
		}
		if len(imp.PkgNameComps) != 1 || imp.PkgNameComps[0].GetValue() != moduleName {
			continue
		}
		if imp.Alias != nil {
			return imp.Alias.GetValue()
		}
		return moduleName
	}
	return ""
}

func isTestAnnotation(att ast.BLangAnnotationAttachment, alias string) bool {
	return att.PkgAlias != nil && att.PkgAlias.GetValue() == alias && att.AnnotationName != nil
}

func identifierValue(id ast.IdentifierNode) string {
	if id == nil {
		return ""
	}
	return id.GetValue()
}

func parseTestConfig(fnName string, att ast.BLangAnnotationAttachment) testplan.TestFunction {
	fields := mappingFields(att.Expr)
	return testplan.TestFunction{
		Name:            fnName,
		Enable:          boolField(fields, "enable", true),
		Groups:          stringArrayField(fields, "groups"),
		DependsOn:       functionRefArrayField(fields, "dependsOn"),
		Before:          functionRefField(fields, "before"),
		After:           functionRefField(fields, "after"),
		SerialExecution: boolField(fields, "serialExecution", false),
		DataProvider:    functionRefField(fields, "dataProvider"),
	}
}

func parseAfterSuiteConfig(fnName string, att ast.BLangAnnotationAttachment) testplan.AfterSuiteHook {
	fields := mappingFields(att.Expr)
	return testplan.AfterSuiteHook{
		Name:      fnName,
		AlwaysRun: boolField(fields, "alwaysRun", false),
	}
}

func parseGroupsHook(fnName string, att ast.BLangAnnotationAttachment, isAfter bool) testplan.GroupHook {
	fields := mappingFields(att.Expr)
	hook := testplan.GroupHook{
		Name:   fnName,
		Groups: stringArrayField(fields, "value"),
	}
	if isAfter {
		hook.AlwaysRun = boolField(fields, "alwaysRun", false)
	}
	return hook
}

func mappingFields(expr ast.BLangExpression) map[string]ast.BLangExpression {
	fields := make(map[string]ast.BLangExpression)
	mapping, ok := expr.(*ast.BLangMappingConstructorExpr)
	if !ok {
		return fields
	}
	for _, field := range mapping.Fields {
		kv, ok := field.(*ast.BLangMappingKeyValueField)
		if !ok {
			continue
		}
		key, ok := mappingKeyName(kv.Key)
		if !ok {
			continue
		}
		fields[key] = kv.ValueExpr
	}
	return fields
}

func mappingKeyName(key *ast.BLangMappingKey) (string, bool) {
	if key == nil || key.Expr == nil {
		return "", false
	}
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		value, ok := expr.Value.(string)
		return value, ok
	case *ast.BLangVarRef:
		return identifierValue(expr.VariableName), true
	default:
		return "", false
	}
}

func boolField(fields map[string]ast.BLangExpression, name string, defaultValue bool) bool {
	expr, ok := fields[name]
	if !ok {
		return defaultValue
	}
	literal, ok := expr.(*ast.BLangLiteral)
	if !ok {
		return defaultValue
	}
	value, ok := literal.Value.(bool)
	if !ok {
		return defaultValue
	}
	return value
}

func stringArrayField(fields map[string]ast.BLangExpression, name string) []string {
	expr, ok := fields[name]
	if !ok {
		return nil
	}
	list, ok := expr.(*ast.BLangListConstructorExpr)
	if !ok {
		return nil
	}
	var result []string
	for _, member := range list.Exprs {
		literal, ok := member.(*ast.BLangLiteral)
		if !ok {
			continue
		}
		value, ok := literal.Value.(string)
		if !ok {
			continue
		}
		result = append(result, value)
	}
	return result
}

func functionRefField(fields map[string]ast.BLangExpression, name string) string {
	expr, ok := fields[name]
	if !ok {
		return ""
	}
	return functionRefName(expr)
}

func functionRefArrayField(fields map[string]ast.BLangExpression, name string) []string {
	expr, ok := fields[name]
	if !ok {
		return nil
	}
	list, ok := expr.(*ast.BLangListConstructorExpr)
	if !ok {
		return nil
	}
	var result []string
	for _, member := range list.Exprs {
		if refName := functionRefName(member); refName != "" {
			result = append(result, refName)
		}
	}
	return result
}

func functionRefName(expr ast.BLangExpression) string {
	varRef, ok := expr.(*ast.BLangVarRef)
	if !ok {
		return ""
	}
	return identifierValue(varRef.VariableName)
}
