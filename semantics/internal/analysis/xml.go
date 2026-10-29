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

package analysis

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	semcommon "github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func validateXMLElement(cx scope.Context, e *ast.BLangXMLElementLiteral, expected semtypes.SemType) (scope.Facts, bool) {
	for i := range e.Attrs {
		if e.Attrs[i].Value == nil {
			continue
		}
		if _, ok := validateActionOrExpr(cx, e.Attrs[i].Value, semtypes.String); !ok {
			return scope.NoFacts, false
		}
	}
	if e.Content != nil {
		if _, ok := validateActionOrExpr(cx, e.Content, semtypes.SemType{}); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateXMLSequence(cx scope.Context, e *ast.BLangXMLSequenceLiteral, expected semtypes.SemType) (scope.Facts, bool) {
	for _, child := range e.Children {
		if _, ok := validateActionOrExpr(cx, child, semtypes.SemType{}); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateTemplate(cx scope.Context, e *ast.BLangTemplateExpr, expected semtypes.SemType) (scope.Facts, bool) {
	for _, insertion := range e.Insertions {
		if _, ok := validateActionOrExpr(cx, insertion, semcommon.TemplateInsertionAllowedTypes); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateXMLTemplate(cx scope.Context, e *ast.BLangXMLTemplateExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if len(e.InsertionKinds) != len(e.Insertions) {
		cx.InternalErr("xml template insertion kind count mismatch", e.GetPosition())
		return scope.NoFacts, false
	}
	for i, insertion := range e.Insertions {
		if _, ok := validateActionOrExpr(cx, insertion, semcommon.XMLTemplateInsertionAllowedTypes(e.InsertionKinds[i])); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateXMLFilter(cx scope.Context, e *ast.BLangXMLFilterExpression, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.Expression, semtypes.XML); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// validateXMLStep validates the step's operands as written, in the enclosing
// context; its generated lowering is not validated.
func validateXMLStep(cx scope.Context, e *ast.BLangXMLStepExpression, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.Expression, semtypes.XML); !ok {
		return scope.NoFacts, false
	}
	for _, extension := range e.Extensions {
		if !dispatchXMLStepExtend(cx, extension) {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// dispatchXMLStepExtend validates the operands of one step extension; a name
// filter has none.
func dispatchXMLStepExtend(cx scope.Context, extension ast.XMLStepExtend) bool {
	switch extension := extension.(type) {
	case *ast.BLangXMLStepFilterExtend:
		return true
	case *ast.BLangXMLStepIndexExtend:
		_, ok := validateActionOrExpr(cx, extension.Expression, semtypes.Int)
		return ok
	case *ast.BLangXMLStepMethodCallExtend:
		return validateXMLStepMethodCall(cx, extension)
	default:
		cx.InternalErr("unsupported XML step extension", extension.GetPosition())
		return false
	}
}

// validateXMLStepMethodCall validates the arguments of a step method call;
// type resolution checked them against the lang.xml method's parameters.
func validateXMLStepMethodCall(cx scope.Context, extension *ast.BLangXMLStepMethodCallExtend) bool {
	for _, arg := range extension.Invocation.ArgExprs {
		if _, ok := validateActionOrExpr(cx, arg, semtypes.SemType{}); !ok {
			return false
		}
	}
	return true
}
