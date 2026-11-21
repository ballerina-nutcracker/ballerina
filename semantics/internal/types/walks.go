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
	"sort"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
)

type symbolDepCollector struct {
	t       typeResolver
	nodeSet map[model.SymbolRef]int
	deps    map[int]struct{}
}

func (c *symbolDepCollector) depends(ref model.SymbolRef) {
	unnarrowed := c.t.unnarrowedSymbol(ref)
	if idx, ok := c.nodeSet[unnarrowed]; ok {
		c.deps[idx] = struct{}{}
	}
}

func (c *symbolDepCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangVarRef:
		c.depends(n.Symbol())
	case *ast.BLangConstRef:
		c.depends(n.Symbol())
	}
	return c
}

func (c *symbolDepCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

// symbolDeps returns the sorted indices in nodeSet of the symbols expr refers to.
func symbolDeps(t typeResolver, nodeSet map[model.SymbolRef]int, expr ast.BLangNode) []int {
	c := &symbolDepCollector{t: t, nodeSet: nodeSet, deps: make(map[int]struct{})}
	ast.Walk(c, expr)
	var deps []int
	for d := range c.deps {
		deps = append(deps, d)
	}
	sort.Ints(deps)
	return deps
}

type trialDependencyRefCollector struct {
	refs []model.SymbolRef
}

func (c *trialDependencyRefCollector) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	switch ref := node.(type) {
	case *ast.BLangVarRef, *ast.BLangConstRef, *ast.BLangUserDefinedType:
		if symbol := ref.(ast.BNodeWithSymbol).Symbol(); !symbol.IsEmpty() {
			c.refs = append(c.refs, symbol)
		}
	case *ast.BLangInvocation:
		// A method call still carries a deferred method symbol; its receiver's type is reached through other refs.
		if symbol, ok := ref.RawSymbol.(*model.SymbolRef); ok && symbol != nil && !symbol.IsEmpty() {
			c.refs = append(c.refs, *symbol)
		}
	case *ast.BLangObjectType:
		// Object type inclusions are held as symbols, not as walked type descriptor nodes.
		c.refs = append(c.refs, ref.Inclusions...)
	}
	return c
}

func (c *trialDependencyRefCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

// collectTrialDependencyRefs returns the symbols roots refer to or call, in pre-order, keeping duplicates.
func collectTrialDependencyRefs(roots []ast.BLangExpression) []model.SymbolRef {
	c := &trialDependencyRefCollector{}
	for _, root := range roots {
		ast.Walk(c, root)
	}
	return c.refs
}

type compileTimeDependencyValidator struct {
	p       *packageTypeResolver
	root    model.SymbolRef
	visited map[model.SymbolRef]bool
	cyclic  bool
	failed  bool
}

func (v *compileTimeDependencyValidator) reference(ref model.SymbolRef) {
	if ref.IsEmpty() || v.cyclic || v.failed {
		return
	}
	p := v.p
	if ref == v.root {
		v.cyclic = true
		return
	}
	if constant, found := p.packageConstants[ref]; found {
		switch p.lazyResolutionStatus[ref] {
		case resolutionInProgress:
			v.cyclic = true
			return
		case resolutionFailed:
			v.failed = true
			return
		case resolutionPending, resolutionDone:
		}
		if v.visited[ref] {
			return
		}
		v.visited[ref] = true
		v.typeDescriptor(constant.TypeNode())
		if expr, ok := constant.Expr.(ast.BLangExpression); ok {
			v.expression(expr)
		}
		return
	}
	if v.visited[ref] {
		return
	}
	v.visited[ref] = true
	if definition, found := p.typeDefnNodes[ref]; found {
		v.typeDescriptor(definition.GetTypeData().TypeDescriptor.(ast.BType))
	}
	if class, found := p.classDefnNodes[ref]; found {
		for _, inclusion := range class.Inclusions {
			v.reference(inclusion)
		}
		for _, field := range class.Fields {
			v.typeDescriptor(field.TypeNode())
		}
		for _, method := range class.Methods {
			v.signature(method)
		}
		for _, method := range class.ResourceMethods {
			v.signature(method)
			for _, segment := range method.ResourcePath {
				v.typeDescriptor(segment.ParamType)
			}
		}
		if class.InitFunction != nil {
			v.signature(class.InitFunction)
		}
	}
}

func (v *compileTimeDependencyValidator) signature(fn common.FunctionDecl) {
	for _, parameter := range fn.GetParameters() {
		v.typeDescriptor(parameter.TypeNode())
	}
	if rest := fn.GetRestParam(); rest != nil {
		v.typeDescriptor(rest.TypeNode())
	}
	if result := fn.GetReturnTypeDescriptor(); result != nil {
		v.typeDescriptor(result)
	}
}

func (v *compileTimeDependencyValidator) typeData(data *ast.TypeData) {
	if data.TypeDescriptor != nil {
		v.typeDescriptor(data.TypeDescriptor.(ast.BType))
	}
}

func (v *compileTimeDependencyValidator) typeDescriptor(node ast.BType) {
	if node == nil || v.cyclic || v.failed {
		return
	}
	switch node := node.(type) {
	case *ast.BLangUserDefinedType:
		v.reference(node.Symbol())
	case *ast.BLangReturnTypeDescriptor:
		v.typeDescriptor(node.TypeDescriptor)
	case *ast.BLangArrayType:
		v.typeData(&node.Elemtype)
		for _, size := range node.Sizes {
			if size != nil {
				v.expression(size)
			}
		}
	case *ast.BLangUnionTypeNode:
		v.typeData(node.Lhs())
		v.typeData(node.Rhs())
	case *ast.BLangIntersectionTypeNode:
		v.typeData(node.Lhs())
		v.typeData(node.Rhs())
	case *ast.BLangErrorTypeNode:
		v.typeData(&node.DetailType)
	case *ast.BLangConstrainedType:
		v.typeData(&node.BaseType)
		v.typeData(&node.Constraint)
	case *ast.BLangStreamType:
		v.typeData(&node.ValueType)
		v.typeData(&node.CompletionType)
	case *ast.BLangTupleTypeNode:
		for _, member := range node.Members {
			v.typeDescriptor(member.TypeDesc.(ast.BType))
		}
		v.typeDescriptor(node.Rest)
	case *ast.BLangFiniteTypeNode:
		for _, value := range node.ValueSpace {
			v.expression(value)
		}
	case *ast.BLangRecordType:
		for _, inclusion := range node.Inclusions {
			v.reference(inclusion)
		}
		for _, inclusion := range node.TypeInclusions {
			v.typeDescriptor(inclusion)
		}
		for _, field := range node.FieldPtrs() {
			v.typeDescriptor(field.Type)
		}
		v.typeDescriptor(node.RestType)
	case *ast.BLangFunctionType:
		for _, parameter := range node.RequiredParams {
			v.typeDescriptor(parameter.TypeDesc)
		}
		if node.RestParam != nil {
			v.typeDescriptor(node.RestParam.TypeDesc)
		}
		v.typeDescriptor(node.ReturnTypeDescriptor)
	case *ast.BLangObjectType:
		for _, inclusion := range node.Inclusions {
			v.reference(inclusion)
		}
		for member := range node.Members() {
			switch member := member.(type) {
			case *ast.BObjectField:
				v.typeDescriptor(member.Ty)
			case *ast.BMethodDecl:
				v.typeDescriptor(&member.BLangFunctionType)
			}
		}
	case *ast.BLangValueType, *ast.BLangBuiltInRefTypeNode, *ast.BLangBadTypeNode:
	}
}

func (v *compileTimeDependencyValidator) expression(expr ast.BLangExpression) {
	if expr == nil || v.cyclic || v.failed {
		return
	}
	switch expr := expr.(type) {
	case *ast.BLangVarRef:
		v.reference(expr.Symbol())
	case *ast.BLangConstRef:
		v.reference(expr.Symbol())
	case *ast.BLangUnaryExpr:
		v.expression(expr.Expr)
	case *ast.BLangTypeConversionExpr:
		v.typeDescriptor(expr.TypeDescriptor)
		v.expression(expr.Expression)
	case *ast.BLangGroupExpr:
		v.expression(expr.Expression)
	case *ast.BLangBinaryExpr:
		v.expression(expr.LhsExpr)
		v.expression(expr.RhsExpr)
	case *ast.BLangTernaryExpr:
		v.expression(expr.Condition)
		v.expression(expr.ThenExpr)
		v.expression(expr.ElseExpr)
	case *ast.BLangNilConditionalExpr:
		v.expression(expr.LhsExpr)
		v.expression(expr.RhsExpr)
	case *ast.BLangListConstructorExpr:
		for _, member := range expr.Exprs {
			v.expression(member)
		}
	case *ast.BLangMappingConstructorExpr:
		for _, field := range expr.Fields {
			if field, ok := field.(*ast.BLangMappingKeyValueField); ok {
				if field.Key != nil && field.Key.Kind == ast.MappingKeyComputed {
					v.expression(field.Key.Expr)
				}
				v.expression(field.ValueExpr)
			}
		}
	case *ast.BLangTemplateExpr:
		for _, insertion := range expr.Insertions {
			v.expression(insertion)
		}
	case *ast.BLangXMLTemplateExpr:
		for _, insertion := range expr.Insertions {
			v.expression(insertion)
		}
	case *ast.BLangLiteral, *ast.BLangNumericLiteral:
	}
}
