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

package types

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type defaultBody struct {
	owner           ast.CaptureGroupOwner
	expr            ast.BLangActionOrExpression
	expectedType    semtypes.SemType
	scope           model.Scope
	isolated        bool
	isolatedXMLStep bool
}

type recordDefaultDeclaration struct {
	body       defaultBody
	typeNode   ast.BType
	bodyStatus resolutionStatus
}

type declaredMappingDefaults struct {
	atom     *semtypes.MappingAtomicType
	defaults []model.FieldDefault
}

type constantOutcomeKind uint8

const (
	constantValue constantOutcomeKind = iota
	constantRuntime
	constantFailed
)

type constantOutcome struct {
	kind  constantOutcomeKind
	value values.BalValue
}

func collectDefaultBody(t typeResolver, owner ast.CaptureGroupOwner, expr ast.BLangActionOrExpression, expectedType semtypes.SemType, xmlStep bool) defaultBody {
	body := defaultBody{owner: owner, expr: expr, expectedType: expectedType, scope: t.currentScope(), isolated: isolatedContext(t), isolatedXMLStep: xmlStep}
	t.resolutionContext().defaults = append(t.resolutionContext().defaults, body)
	return body
}

func detachDefaultBodies(t typeResolver, mark int) []defaultBody {
	ctx := t.resolutionContext()
	if ctx.collectingModuleTypes && !t.isEphemeral() {
		return nil
	}
	bodies := append([]defaultBody(nil), ctx.defaults[mark:]...)
	ctx.defaults = ctx.defaults[:mark]
	return bodies
}

func resolveInlineType(t typeResolver, chain *binding, node ast.BType) (semtypes.SemType, *binding, []defaultBody, bool) {
	mark := len(t.resolutionContext().defaults)
	ty, chain, ok := resolveBType(t, chain, node)
	return ty, chain, detachDefaultBodies(t, mark), ok
}

func resolveInlineSignature(t typeResolver, chain *binding, fn *ast.BLangFunction) (semtypes.SemType, *binding, []defaultBody, bool) {
	mark := len(t.resolutionContext().defaults)
	ty, chain, ok := resolveFunctionSignature(t, chain, fn)
	return ty, chain, detachDefaultBodies(t, mark), ok
}

func checkDefaultBody(t typeResolver, chain *binding, body defaultBody) bool {
	scope := t.currentScope()
	t.setCurrentScope(body.scope)
	defer t.setCurrentScope(scope)
	restoreIsolation := setIsolatedContext(t, body.isolated)
	defer restoreIsolation()
	restoreXML := setIsolatedXMLStepContext(t, body.isolatedXMLStep)
	defer restoreXML()
	_, ok := resolveDefaultExpression(t, chain, body.owner, body.expr, body.expectedType)
	return ok
}

func recordDefaultDeclarationFor(t typeResolver, ref model.SymbolRef) (typeResolver, *recordDefaultDeclaration) {
	for current := t; current != nil; current = current.parent() {
		if declaration, found := current.resolutionContext().recordDefaults[ref]; found {
			return current, declaration
		}
	}
	return nil, nil
}

func checkRecordDefaultBody(t typeResolver, chain *binding, declaration *recordDefaultDeclaration) bool {
	switch declaration.bodyStatus {
	case resolutionDone:
		return true
	case resolutionFailed:
		return false
	case resolutionInProgress:
		return true
	case resolutionPending:
	}
	declaration.bodyStatus = resolutionInProgress
	ok := checkDefaultBody(t, chain, declaration.body)
	if ok {
		declaration.bodyStatus = resolutionDone
	} else {
		declaration.bodyStatus = resolutionFailed
	}
	return ok
}

func checkDefaultBodies(t typeResolver, chain *binding, bodies []defaultBody) bool {
	ok := true
	for _, body := range bodies {
		field, record := body.owner.(*ast.BField)
		if record && !t.isEphemeral() {
			declaration := t.resolutionContext().recordDefaults[field.Default.FnRef]
			if _, classified := classifyFieldDefault(t, chain, "", field); !classified {
				ok = false
			}
			if !checkRecordDefaultBody(t, chain, declaration) {
				context := t.resolutionContext()
				context.constantDefaults[field.Default.FnRef] = constantOutcome{kind: constantFailed}
				context.valueStatus[field.Default.FnRef] = resolutionFailed
				ok = false
			}
		} else if !checkDefaultBody(t, chain, body) {
			ok = false
		}
	}
	if !t.isEphemeral() {
		finalizeMappingDefaults(t)
	}
	return ok
}

func finalizeMappingDefaults(t typeResolver) {
	for _, declaration := range t.resolutionContext().mappingDeclarations {
		for i := range declaration.defaults {
			field := &declaration.defaults[i]
			owner, provider := recordDefaultDeclarationFor(t, field.FnRef)
			if provider == nil {
				continue
			}
			if outcome, found := owner.resolutionContext().constantDefaults[field.FnRef]; found && outcome.kind == constantValue {
				field.IsConst = true
				field.Value = outcome.value
			}
		}
		t.setMappingDefaults(declaration.atom, declaration.defaults)
	}
}

func constantDefaultValue(t typeResolver, chain *binding, ref model.SymbolRef) constantOutcome {
	owner, declaration := recordDefaultDeclarationFor(t, ref)
	ctx := owner.resolutionContext()
	if outcome, found := ctx.constantDefaults[ref]; found {
		return outcome
	}
	if ctx.valueStatus[ref] == resolutionInProgress {
		owner.semanticError("cyclic dependency in record field default", declaration.body.expr.GetPosition())
		ctx.valueStatus[ref] = resolutionFailed
		outcome := constantOutcome{kind: constantFailed}
		ctx.constantDefaults[ref] = outcome
		return outcome
	}
	ctx.valueStatus[ref] = resolutionInProgress
	outcome := evaluateConstantDefault(owner, chain, ref, declaration)
	if outcome.kind == constantFailed {
		ctx.valueStatus[ref] = resolutionFailed
	} else {
		ctx.valueStatus[ref] = resolutionDone
	}
	ctx.constantDefaults[ref] = outcome
	return outcome
}

func evaluateConstantDefault(t typeResolver, chain *binding, ref model.SymbolRef, declaration *recordDefaultDeclaration) constantOutcome {
	expr := declaration.body.expr.(ast.BLangExpression)
	if _, eligible := isConstantForm(t, expr); !eligible {
		return constantOutcome{kind: constantRuntime}
	}
	if !validateRecordDefaultDependencies(t, ref, declaration.typeNode, expr, expr.GetPosition()) {
		declaration.bodyStatus = resolutionFailed
		return constantOutcome{kind: constantFailed}
	}
	if !checkRecordDefaultBody(t, chain, declaration) {
		return constantOutcome{kind: constantFailed}
	}
	// Folding a default is an optimisation: a default that cannot be folded is
	// evaluated at runtime, but one whose evaluation fails is an error.
	value, unfoldable, ok := tryFoldConstant(t, chain, expr)
	switch {
	case ok:
		return constantOutcome{kind: constantValue, value: value}
	case unfoldable != nil:
		return constantOutcome{kind: constantRuntime}
	default:
		return constantOutcome{kind: constantFailed}
	}
}
