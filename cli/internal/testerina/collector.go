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
	"slices"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/compilerplugin"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	testOrg        = "ballerina"
	testModule     = "test"
	configAnnName  = "Config"
	enableField    = "enable"
	beforeField    = "before"
	afterField     = "after"
	mainFunction   = "main"
	initFunction   = "init"
	configAnnLabel = "@test:Config"
)

// Collector owns the injected compiler plugin and accumulates the tests
// discovered in each module it runs on.
type Collector struct {
	mu      sync.Mutex
	modules map[ModuleKey]ModuleTests
}

func NewCollector() *Collector {
	return &Collector{modules: make(map[ModuleKey]ModuleTests)}
}

// Plugin returns the compiler plugin the CLI injects into the compilation.
func (c *Collector) Plugin() compilerplugin.InjectedPlugin {
	return compilerplugin.InjectedPlugin{
		Provider: compilerplugin.Provider{Org: testOrg, Package: testModule},
		Plugin: compilerplugin.CompilerPlugin{
			After:              compilerplugin.AfterSemantics,
			PackageTransformer: c.collect,
		},
	}
}

// Plan returns the collected tests with modules in order, dropping modules
// with no tests and sorting each module's tests into source order.
func (c *Collector) Plan(order []ModuleKey) TestPlan {
	c.mu.Lock()
	defer c.mu.Unlock()
	plan := TestPlan{}
	for _, key := range order {
		tests, ok := c.modules[key]
		if !ok || len(tests.Tests) == 0 {
			continue
		}
		slices.SortFunc(tests.Tests, func(a, b TestFunction) int {
			if a.Position.FileIndex() != b.Position.FileIndex() {
				return a.Position.FileIndex() - b.Position.FileIndex()
			}
			return a.Position.StartOffset() - b.Position.StartOffset()
		})
		plan.Modules = append(plan.Modules, tests)
	}
	return plan
}

// collect runs as the injected plugin's package transformer. It is called once
// per qualifying module, concurrently with the other modules.
func (c *Collector) collect(
	compilerCtx *compilercontext.CompilerContext,
	_ model.ExportedSymbolSpace,
	pkg *ast.BLangPackage,
) (*ast.BLangPackage, error) {
	key := ModuleKey{Org: pkg.PackageID.OrgName.Value(), Module: pkg.PackageID.PkgName.Value()}
	var tests []TestFunction
	for _, fn := range pkg.Functions {
		attachment, ok := configAnnotation(compilerCtx, fn)
		if !ok {
			continue
		}
		test, ok := testFunctionFrom(compilerCtx, fn, attachment)
		if !ok {
			continue
		}
		tests = append(tests, test)
	}
	if len(tests) > 0 {
		c.mu.Lock()
		defer c.mu.Unlock()
		existing := c.modules[key]
		existing.Key = key
		existing.Tests = append(existing.Tests, tests...)
		c.modules[key] = existing
	}
	return pkg, nil
}

func configAnnotation(
	compilerCtx *compilercontext.CompilerContext, fn *ast.BLangFunction,
) (ast.BLangAnnotationAttachment, bool) {
	for _, attachment := range fn.GetAnnotationAttachments() {
		if !ast.SymbolIsSet(&attachment) {
			continue
		}
		pkg := compilerCtx.SymbolPackage(attachment.Symbol())
		if pkg.Organization == testOrg && pkg.Package == testModule &&
			compilerCtx.SymbolName(attachment.Symbol()) == configAnnName {
			return attachment, true
		}
	}
	return ast.BLangAnnotationAttachment{}, false
}

func testFunctionFrom(
	compilerCtx *compilercontext.CompilerContext,
	fn *ast.BLangFunction,
	attachment ast.BLangAnnotationAttachment,
) (TestFunction, bool) {
	name := fn.GetName().GetValue()
	if !validTestSignature(compilerCtx, fn, name) {
		return TestFunction{}, false
	}
	test := TestFunction{Name: name, Enabled: true, Position: fn.GetPosition()}
	fields, ok := configFields(compilerCtx, attachment)
	if !ok {
		return TestFunction{}, false
	}
	for _, field := range fields {
		switch field.name {
		case enableField:
			enabled, ok := boolFieldValue(compilerCtx, attachment, field)
			if !ok {
				return TestFunction{}, false
			}
			test.Enabled = enabled
		case beforeField:
			ref, ok := functionFieldValue(compilerCtx, field)
			if !ok {
				return TestFunction{}, false
			}
			test.Before = ref
		case afterField:
			ref, ok := functionFieldValue(compilerCtx, field)
			if !ok {
				return TestFunction{}, false
			}
			test.After = ref
		}
	}
	return test, true
}

func validTestSignature(
	compilerCtx *compilercontext.CompilerContext, fn *ast.BLangFunction, name string,
) bool {
	// Defaultable params live in RequiredParams (flagged by IsDefaultableParam),
	// so this rejects required, defaultable and rest parameters alike.
	if len(fn.RequiredParams) != 0 || fn.RestParam != nil {
		compilerCtx.SemanticError(
			configAnnLabel+" function '"+name+"' must not have parameters", fn.GetPosition())
		return false
	}
	if name == mainFunction || name == initFunction {
		compilerCtx.SemanticError(
			configAnnLabel+" cannot be applied to '"+name+"'", fn.GetPosition())
		return false
	}
	return true
}

// configField is one `key: value` entry of the annotation, sourced either from
// the attachment AST or from the evaluated constant value.
type configField struct {
	name string
	expr ast.BLangExpression
}

// configFields returns the fields written in the attachment. The attachment
// expression is not desugared at this stage, so a mapping constructor is the
// only shape a `@test:Config { ... }` attachment can take.
func configFields(
	compilerCtx *compilercontext.CompilerContext, attachment ast.BLangAnnotationAttachment,
) ([]configField, bool) {
	if attachment.Expr == nil {
		return nil, true
	}
	mapping, ok := attachment.Expr.(*ast.BLangMappingConstructorExpr)
	if !ok {
		compilerCtx.SemanticError(
			configAnnLabel+" must be a mapping constructor", attachment.GetPosition())
		return nil, false
	}
	fields := make([]configField, 0, len(mapping.Fields))
	for _, field := range mapping.Fields {
		keyValue, ok := field.(*ast.BLangMappingKeyValueField)
		if !ok {
			continue
		}
		name, ok := fieldName(keyValue)
		if !ok {
			continue
		}
		fields = append(fields, configField{name: name, expr: keyValue.ValueExpr})
	}
	return fields, true
}

func fieldName(field *ast.BLangMappingKeyValueField) (string, bool) {
	if field.Key == nil || field.Key.Kind == ast.MappingKeyComputed {
		return "", false
	}
	literal, ok := field.Key.Expr.(*ast.BLangLiteral)
	if ok {
		value, ok := literal.Value.(string)
		return value, ok
	}
	if ref, ok := field.Key.Expr.(*ast.BLangVarRef); ok {
		return ref.VariableName.GetValue(), true
	}
	return "", false
}

// boolFieldValue reads `enable`. The attachment AST carries the written
// expression; a fully constant attachment additionally has an evaluated value,
// which is consulted when the expression itself is not a literal.
func boolFieldValue(
	compilerCtx *compilercontext.CompilerContext,
	attachment ast.BLangAnnotationAttachment,
	field configField,
) (bool, bool) {
	if literal, ok := field.expr.(*ast.BLangLiteral); ok {
		if value, ok := literal.Value.(bool); ok {
			return value, true
		}
	}
	if evaluated, ok := attachment.AnnotationValue.(*values.Map); ok && evaluated != nil {
		if value, ok := evaluated.Get(field.name); ok {
			if enabled, ok := value.(bool); ok {
				return enabled, true
			}
		}
	}
	compilerCtx.SemanticError(
		configAnnLabel+" '"+enableField+"' must be a boolean literal", field.expr.GetPosition())
	return false, false
}

// functionFieldValue reads `before`/`after`, which must name a top-level
// function. Visibility is already enforced by the type checker.
func functionFieldValue(
	compilerCtx *compilercontext.CompilerContext, field configField,
) (FunctionRef, bool) {
	ref, ok := field.expr.(*ast.BLangVarRef)
	if ok && ast.SymbolIsSet(ref) && compilerCtx.SymbolKind(ref.Symbol()) == model.SymbolKindFunction {
		pkg := compilerCtx.SymbolPackage(ref.Symbol())
		return FunctionRef{
			Org:    pkg.Organization,
			Module: pkg.Package,
			Name:   compilerCtx.SymbolName(ref.Symbol()),
		}, true
	}
	compilerCtx.SemanticError(
		configAnnLabel+" '"+field.name+"' must reference a function", field.expr.GetPosition())
	return FunctionRef{}, false
}
