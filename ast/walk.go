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

package ast

import (
	"fmt"
)

// A Visitor's Visit method is invoked for each node encountered by [Walk].
// If the result visitor w is not nil, [Walk] visits each of the children
// of node with the visitor w, followed by a call of w.Visit(nil).
type Visitor interface {
	Visit(node BLangNode) (w Visitor)
	VisitTypeData(typeData *TypeData) (w Visitor)
}

// Walk traverses an AST in depth-first order: It starts by calling
// v.Visit(node); node must not be nil. If the visitor w returned by
// v.Visit(node) is not nil, Walk is invoked recursively with visitor
// w for each of the non-nil children of node, followed by a call of
// w.Visit(nil).
func Walk(v Visitor, node BLangNode) {
	if v = v.Visit(node); v == nil {
		return
	}
	switch node := node.(type) {
	// Section 1: Top-Level/Package Declarations
	case *BLangPackage:
		for i := range node.Imports {
			Walk(v, &node.Imports[i])
		}
		for i := range node.XmlnsList {
			Walk(v, &node.XmlnsList[i])
		}
		for i := range node.Constants {
			Walk(v, &node.Constants[i])
		}
		for i := range node.GlobalVars {
			Walk(v, &node.GlobalVars[i])
		}
		for i := range node.Services {
			Walk(v, &node.Services[i])
		}
		for i := range node.Functions {
			Walk(v, &node.Functions[i])
		}
		for i := range node.TypeDefinitions {
			Walk(v, &node.TypeDefinitions[i])
		}
		for i := range node.Annotations {
			Walk(v, &node.Annotations[i])
		}
		if node.InitFunction != nil {
			Walk(v, node.InitFunction)
		}
		for i := range node.ClassDefinitions {
			Walk(v, &node.ClassDefinitions[i])
		}

	case *BLangTestablePackage:
		for i := range node.Imports {
			Walk(v, &node.Imports[i])
		}
		for i := range node.XmlnsList {
			Walk(v, &node.XmlnsList[i])
		}
		for i := range node.Constants {
			Walk(v, &node.Constants[i])
		}
		for i := range node.GlobalVars {
			Walk(v, &node.GlobalVars[i])
		}
		for i := range node.Services {
			Walk(v, &node.Services[i])
		}
		for i := range node.Functions {
			Walk(v, &node.Functions[i])
		}
		for i := range node.TypeDefinitions {
			Walk(v, &node.TypeDefinitions[i])
		}
		for i := range node.Annotations {
			Walk(v, &node.Annotations[i])
		}
		if node.InitFunction != nil {
			Walk(v, node.InitFunction)
		}
		for i := range node.ClassDefinitions {
			Walk(v, &node.ClassDefinitions[i])
		}

	case *BLangCompilationUnit:
		for _, topLevelNode := range node.TopLevelNodes {
			Walk(v, topLevelNode.(BLangNode))
		}

	case *BLangImportPackage:
		if node.OrgName != nil {
			Walk(v, node.OrgName)
		}
		for i := range node.PkgNameComps {
			Walk(v, &node.PkgNameComps[i])
		}
		if node.Alias != nil {
			Walk(v, node.Alias)
		}
		if node.CompUnit != nil {
			Walk(v, node.CompUnit)
		}
		if node.Version != nil {
			Walk(v, node.Version)
		}

	case *BLangService:
		for _, expr := range node.AttachedExprs {
			Walk(v, expr.(BLangNode))
		}
		if node.AttachPointLiteral != nil {
			Walk(v, node.AttachPointLiteral)
		}
		for i := range node.AbsoluteResourcePath {
			Walk(v, &node.AbsoluteResourcePath[i])
		}
		walkClassDefnBody(v, &node.classDefnBase)

	case *BLangClassDefinition:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		walkClassDefnBody(v, &node.classDefnBase)

	case *BLangAnnotation:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		for i := range node.AnnAttachments {
			Walk(v, &node.AnnAttachments[i])
		}
		walkTypeDescriptor(v, node.typeDescriptor)

	case *BLangAnnotationAttachment:
		Walk(v, node.Expr.(BLangNode))
		if node.AnnotationName != nil {
			Walk(v, node.AnnotationName)
		}
		if node.PkgAlias != nil {
			Walk(v, node.PkgAlias)
		}

	// Section 2: Type Definitions & Variables
	case *BLangTypeDefinition:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		WalkTypeData(v, &node.typeData)
		for i := range node.annAttachments {
			Walk(v, &node.annAttachments[i])
		}

	case *BLangConstant:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		if tn := node.TypeNode(); tn != nil {
			Walk(v, tn.(BLangNode))
		}

	case *BLangSimpleVariable:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		if tn := node.TypeNode(); tn != nil {
			Walk(v, tn.(BLangNode))
		}
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangXMLNS:
		Walk(v, node.namespaceURI.(BLangNode))
		if node.prefix != nil {
			Walk(v, node.prefix)
		}

	// Section 3: Function & Body
	case *BLangFunction:
		Walk(v, &node.Name)
		for i := range node.RequiredParams {
			Walk(v, &node.RequiredParams[i])
		}
		if node.RestParam != nil {
			Walk(v, node.RestParam.(BLangNode))
		}
		if node.returnTypeDescriptor != nil {
			walkTypeDescriptor(v, node.returnTypeDescriptor)
		}
		if node.Body != nil {
			Walk(v, node.Body.(BLangNode))
		}

	case *BLangResourceMethod:
		Walk(v, &node.Name)
		for i := range node.ResourcePath {
			if tn := node.ResourcePath[i].ParamType; tn != nil {
				walkTypeDescriptor(v, tn)
			}
		}
		for i := range node.RequiredParams {
			Walk(v, &node.RequiredParams[i])
		}
		if node.RestParam != nil {
			Walk(v, node.RestParam.(BLangNode))
		}
		if node.returnTypeDescriptor != nil {
			walkTypeDescriptor(v, node.returnTypeDescriptor)
		}
		if node.Body != nil {
			Walk(v, node.Body.(BLangNode))
		}

	case *BLangBlockFunctionBody:
		for _, stmt := range node.Stmts {
			Walk(v, stmt.(BLangNode))
		}

	case *BLangExprFunctionBody:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangExternFunctionBody:
		// Leaf node - no children to walk

	// Section 4: Statements
	case *BLangBlockStmt:
		for _, stmt := range node.Stmts {
			Walk(v, stmt.(BLangNode))
		}

	case *BLangAssignment:
		if node.VarRef != nil {
			Walk(v, node.VarRef.(BLangNode))
		}
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangCompoundAssignment:
		if node.VarRef != nil {
			Walk(v, node.VarRef.(BLangNode))
		}
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangExpressionStmt:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangIf:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		Walk(v, &node.Body)
		if node.ElseStmt != nil {
			Walk(v, node.ElseStmt.(BLangNode))
		}

	case *BLangWhile:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		Walk(v, &node.Body)
		Walk(v, &node.OnFailClause)

	case *BLangForeach:
		if node.Collection != nil {
			Walk(v, node.Collection.(BLangNode))
		}
		if node.VariableDef != nil {
			Walk(v, node.VariableDef)
		}
		Walk(v, &node.Body)
		if node.OnFailClause != nil {
			Walk(v, node.OnFailClause)
		}

	case *BLangDo:
		Walk(v, &node.Body)
		Walk(v, &node.OnFailClause)

	case *BLangLock:
		Walk(v, &node.Body)

	case *BLangMatchStatement:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		for i := range node.MatchClauses {
			Walk(v, &node.MatchClauses[i])
		}

	case *BLangMatchClause:
		for _, pattern := range node.Patterns {
			Walk(v, pattern.(BLangNode))
		}
		if node.Guard != nil {
			Walk(v, node.Guard)
		}
		Walk(v, &node.Body)

	case *BLangSimpleVariableDef:
		Walk(v, node.Var)

	case *BLangReturn:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangPanic:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangBreak:
		// Leaf node

	case *BLangContinue:
		// Leaf node

	// Section 5: Expressions - Basic
	case *BLangBinaryExpr:
		if node.LhsExpr != nil {
			Walk(v, node.LhsExpr.(BLangNode))
		}
		if node.RhsExpr != nil {
			Walk(v, node.RhsExpr.(BLangNode))
		}

	case *BLangQueryExpr:
		for i := range node.QueryClauseList {
			Walk(v, node.QueryClauseList[i])
		}

	case *BLangUnaryExpr:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangElvisExpr:
		if node.LhsExpr != nil {
			Walk(v, node.LhsExpr.(BLangNode))
		}
		if node.RhsExpr != nil {
			Walk(v, node.RhsExpr.(BLangNode))
		}

	case *BLangCheckedExpr:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangCheckPanickedExpr:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangTrapExpr:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangGroupExpr:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	case *BLangIndexBasedAccess:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		if node.IndexExpr != nil {
			Walk(v, node.IndexExpr.(BLangNode))
		}

	case *BLangFieldBaseAccess:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangListConstructorExpr:
		for _, expr := range node.Exprs {
			Walk(v, expr.(BLangNode))
		}
	case *BLangMappingConstructorExpr:
		for _, f := range node.Fields {
			if kv, ok := f.(*BLangMappingKeyValueField); ok {
				if kv.Key != nil && kv.Key.Expr != nil {
					Walk(v, kv.Key.Expr.(BLangNode))
				}
				if kv.ValueExpr != nil {
					Walk(v, kv.ValueExpr.(BLangNode))
				}
			}
		}
	case *BLangErrorConstructorExpr:
		if node.ErrorTypeRef != nil {
			Walk(v, node.ErrorTypeRef)
		}
		for _, arg := range node.PositionalArgs {
			Walk(v, arg.(BLangNode))
		}
		for i := range node.NamedArgs {
			Walk(v, &node.NamedArgs[i])
		}

	case *BLangInvocation:
		if node.PkgAlias != nil {
			Walk(v, node.PkgAlias)
		}
		if node.Name != nil {
			Walk(v, node.Name)
		}
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		for _, arg := range node.ArgExprs {
			if arg != nil {
				Walk(v, arg.(BLangNode))
			}
		}
	case *BLangLambdaFunction:
		if node.Function != nil {
			Walk(v, node.Function)
		}

	case *BLangArrowFunction:
		for i := range node.Params {
			Walk(v, &node.Params[i])
		}
		if node.Body != nil {
			Walk(v, node.Body)
		}

	case *BLangAnnotAccessExpr:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		if node.PkgAlias != nil {
			Walk(v, node.PkgAlias)
		}
		if node.AnnotationName != nil {
			Walk(v, node.AnnotationName)
		}

	case *BLangTypedescExpr:
		walkTypeDescriptor(v, node.typeDescriptor)

	case *BLangInferredTypedescDefault:
		// leaf — no children

	case *BLangTypeConversionExpr:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}
		if node.TypeDescriptor != nil {
			Walk(v, node.TypeDescriptor.(BLangNode))
		}

	case *BLangTypeTestExpr:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		WalkTypeData(v, &node.Type)

	case *BLangCommitExpr:
		panic("unimplemented")

	// Section 6: Expressions - Variable Refs
	case *BLangSimpleVarRef:
		if node.PkgAlias != nil {
			Walk(v, node.PkgAlias)
		}
		if node.VariableName != nil {
			Walk(v, node.VariableName)
		}

	case *BLangLocalVarRef:
		if node.PkgAlias != nil {
			Walk(v, node.PkgAlias)
		}
		if node.VariableName != nil {
			Walk(v, node.VariableName)
		}

	case *BLangConstRef:
		if node.PkgAlias != nil {
			Walk(v, node.PkgAlias)
		}
		if node.VariableName != nil {
			Walk(v, node.VariableName)
		}

	case *BLangLiteral:
		// Leaf node

	case *BLangNumericLiteral:
		// Leaf node

	case *BLangXMLSequenceLiteral:
		for _, child := range node.Children {
			Walk(v, child)
		}

	case *BLangTemplateExpr:
		for _, ins := range node.Insertions {
			Walk(v, ins)
		}

	case *BLangXMLTemplateExpr:
		for _, ins := range node.Insertions {
			Walk(v, ins)
		}

	case *BLangXMLElementLiteral:
		for i := range node.Attrs {
			Walk(v, &node.Attrs[i])
		}
		if node.Content != nil {
			Walk(v, node.Content)
		}

	case *BLangXMLAttribute:
		if node.Value != nil {
			Walk(v, node.Value)
		}

	case *BLangXMLPILiteral:
		// Leaf node

	case *BLangXMLCommentLiteral:
		// Leaf node

	case *BLangXMLTextLiteral:
		// Leaf node

	// Section 7: Expressions - Worker
	case *BLangWorkerReceive:
		if node.WorkerIdentifier != nil {
			Walk(v, node.WorkerIdentifier)
		}

	case *BLangAlternateWorkerReceive:
		for i := range node.workerReceives {
			Walk(v, &node.workerReceives[i])
		}

	// Section 8: Type Nodes
	case *BLangArrayType:
		WalkTypeData(v, &node.Elemtype)
		for i := range node.Sizes {
			if node.Sizes[i] != nil {
				Walk(v, node.Sizes[i].(BLangNode))
			}
		}

	case *BLangUserDefinedType:
		Walk(v, &node.PkgAlias)
		Walk(v, &node.TypeName)

	case *BLangValueType:
		// No children to walk

	case *BLangBuiltInRefTypeNode:
		// No children to walk

	case *BLangFiniteTypeNode:
		for i := range node.ValueSpace {
			Walk(v, node.ValueSpace[i].(BLangNode))
		}

	case *BLangUnionTypeNode:
		WalkTypeData(v, &node.lhs)
		WalkTypeData(v, &node.rhs)

	case *BLangIntersectionTypeNode:
		WalkTypeData(v, &node.lhs)
		WalkTypeData(v, &node.rhs)

	case *BLangErrorTypeNode:
		WalkTypeData(v, &node.DetailType)

	case *BLangConstrainedType:
		WalkTypeData(v, &node.Type)
		WalkTypeData(v, &node.Constraint)
	case *BLangStreamType:
		WalkTypeData(v, &node.ValueType)
		WalkTypeData(v, &node.CompletionType)
	case *BLangTupleTypeNode:
		for i := range node.Members {
			Walk(v, node.Members[i].TypeDesc.(BLangNode))
		}
		if node.Rest != nil {
			Walk(v, node.Rest.(BLangNode))
		}

	case *BLangRecordType:
		for _, inclusion := range node.TypeInclusions {
			Walk(v, inclusion.(BLangNode))
		}
		for _, field := range node.fields {
			Walk(v, field.Type.(BLangNode))
			if field.DefaultExpr != nil {
				Walk(v, field.DefaultExpr.(BLangNode))
			}
		}
		if node.RestType != nil {
			Walk(v, node.RestType.(BLangNode))
		}

	case *BLangObjectType:
		for member := range node.Members() {
			Walk(v, member.(BLangNode))
		}

	case *BObjectField:
		Walk(v, node.Ty.(BLangNode))

	case *BMethodDecl:
		for _, param := range node.RequiredParams {
			Walk(v, param.TypeDesc.(BLangNode))
		}
		if node.ReturnTypeDescriptor != nil {
			Walk(v, node.ReturnTypeDescriptor.(BLangNode))
		}

	case *BLangFunctionType:
		for i := range node.RequiredParams {
			Walk(v, &node.RequiredParams[i])
		}
		if node.RestParam != nil {
			Walk(v, node.RestParam)
		}
		if node.ReturnTypeDescriptor != nil {
			Walk(v, node.ReturnTypeDescriptor.(BLangNode))
		}

	case *BLangFunctionTypeParam:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		if node.TypeDesc != nil {
			Walk(v, node.TypeDesc.(BLangNode))
		}
		if node.InitExpr != nil {
			Walk(v, node.InitExpr.(BLangNode))
		}

	// Section 9: Binding Patterns
	case *BLangCaptureBindingPattern:
		Walk(v, &node.Identifier)

	case *BLangWildCardBindingPattern:
		// Leaf node

	case *BLangSimpleBindingPattern:
		if node.CaptureBindingPattern != nil {
			Walk(v, node.CaptureBindingPattern)
		}
		if node.WildCardBindingPattern != nil {
			Walk(v, node.WildCardBindingPattern)
		}

	case *BLangErrorBindingPattern:
		if node.ErrorTypeReference != nil {
			Walk(v, node.ErrorTypeReference)
		}
		if node.ErrorMessageBindingPattern != nil {
			Walk(v, node.ErrorMessageBindingPattern)
		}
		if node.ErrorCauseBindingPattern != nil {
			Walk(v, node.ErrorCauseBindingPattern)
		}
		if node.ErrorFieldBindingPatterns != nil {
			Walk(v, node.ErrorFieldBindingPatterns)
		}

	case *BLangErrorMessageBindingPattern:
		if node.SimpleBindingPattern != nil {
			Walk(v, node.SimpleBindingPattern)
		}

	case *BLangErrorCauseBindingPattern:
		if node.SimpleBindingPattern != nil {
			Walk(v, node.SimpleBindingPattern)
		}
		if node.ErrorBindingPattern != nil {
			Walk(v, node.ErrorBindingPattern)
		}

	case *BLangErrorFieldBindingPatterns:
		for i := range node.NamedArgBindingPatterns {
			Walk(v, &node.NamedArgBindingPatterns[i])
		}
		if node.RestBindingPattern != nil {
			Walk(v, node.RestBindingPattern)
		}

	case *BLangNamedArgBindingPattern:
		if node.ArgName != nil {
			Walk(v, node.ArgName)
		}
		if node.BindingPattern != nil {
			Walk(v, node.BindingPattern.(BLangNode))
		}

	case *BLangRestBindingPattern:
		if node.VariableName != nil {
			Walk(v, node.VariableName)
		}

	// Section 10: Clauses
	case *BLangFromClause:
		if node.Collection != nil {
			Walk(v, node.Collection.(BLangNode))
		}
		if node.VariableDefinitionNode != nil {
			Walk(v, node.VariableDefinitionNode)
		}

	case *BLangJoinClause:
		if node.Collection != nil {
			Walk(v, node.Collection.(BLangNode))
		}
		if node.OnClause.OnExpr != nil {
			Walk(v, node.OnClause.OnExpr.(BLangNode))
		}
		if node.VariableDefinitionNode != nil {
			Walk(v, node.VariableDefinitionNode)
		}
		if node.OnClause.EqualsExpr != nil {
			Walk(v, node.OnClause.EqualsExpr.(BLangNode))
		}

	case *BLangSelectClause:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	case *BLangLetClause:
		for i := range node.LetVarDeclarations {
			Walk(v, &node.LetVarDeclarations[i])
		}

	case *BLangWhereClause:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	case *BLangGroupByClause:
		for _, groupingKey := range node.GetGroupingKeyList() {
			Walk(v, groupingKey.(BLangNode))
		}

	case *BLangGroupingKey:
		if groupingKey := node.GetGroupingKey(); groupingKey != nil {
			Walk(v, groupingKey.(BLangNode))
		}

	case *BLangOnClause:
		if node.OnExpr != nil {
			Walk(v, node.OnExpr.(BLangNode))
		}
		if node.EqualsExpr != nil {
			Walk(v, node.EqualsExpr.(BLangNode))
		}

	case *BLangLimitClause:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	case *BLangOrderByClause:
		for i := range node.OrderByKeyList {
			Walk(v, &node.OrderByKeyList[i])
		}

	case *BLangOrderKey:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	case *BLangOnConflictClause:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	case *BLangOnFailClause:
		if node.Body != nil {
			Walk(v, node.Body)
		}
		if node.VariableDefinitionNode != nil {
			Walk(v, node.VariableDefinitionNode)
		}

	case *BLangDoClause:
		if node.Body != nil {
			Walk(v, node.Body)
		}

	case *BLangCollectClause:
		if node.Expression != nil {
			Walk(v, node.Expression.(BLangNode))
		}

	// Section 11: Documentation
	case *BLangMarkdownDocumentation:
		for i := range node.DocumentationLines {
			Walk(v, &node.DocumentationLines[i])
		}
		for i := range node.Parameters {
			Walk(v, &node.Parameters[i])
		}
		for i := range node.References {
			Walk(v, &node.References[i])
		}
		if node.ReturnParameter != nil {
			Walk(v, node.ReturnParameter)
		}
		if node.DeprecationDocumentation != nil {
			Walk(v, node.DeprecationDocumentation)
		}
		if node.DeprecatedParametersDocumentation != nil {
			Walk(v, node.DeprecatedParametersDocumentation)
		}

	case *BLangMarkdownDocumentationLine:
		// Leaf node

	case *BLangMarkdownParameterDocumentation:
		if node.ParameterName != nil {
			Walk(v, node.ParameterName)
		}

	case *BLangMarkdownReturnParameterDocumentation:
		// Leaf node

	case *BLangMarkDownDeprecationDocumentation:
		// Leaf node

	case *BLangMarkDownDeprecatedParametersDocumentation:
		for i := range node.Parameters {
			Walk(v, &node.Parameters[i])
		}

	// Section 12: Pattern Matching
	case *BLangConstPattern:
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}

	case *BLangWildCardMatchPattern:
		// Leaf node

	// Section 13: Misc Leaf
	case *BLangIdentifier:
		// Leaf node

	case *BLangMarkdownReferenceDocumentation:
		// Leaf node

	case *BLangNamedArgsExpression:
		Walk(v, &node.Name)
		Walk(v, node.Expr.(BLangNode))

	case *BLangNewExpression:
		if node.TypeDescriptor != nil {
			Walk(v, node.TypeDescriptor)
		}
		for _, arg := range node.ArgsExprs {
			if arg != nil {
				Walk(v, arg.(BLangNode))
			}
		}

	case *BLangRemoteMethodCallAction:
		if node.Name != nil {
			Walk(v, node.Name)
		}
		if node.Expr != nil {
			Walk(v, node.Expr.(BLangNode))
		}
		for _, arg := range node.ArgExprs {
			if arg != nil {
				Walk(v, arg.(BLangNode))
			}
		}

	case *BLangClientResourceAccessAction:
		if node.Expr != nil {
			Walk(v, node.Expr)
		}
		for i := range node.Path {
			if e := node.Path[i].Expr; e != nil {
				Walk(v, e)
			}
		}
		for _, arg := range node.ArgExprs {
			if arg != nil {
				Walk(v, arg)
			}
		}

	default:
		panic(fmt.Sprintf("unexpected node type %T", node))
	}
	v.Visit(nil)
}

// We need to do this because TypeData is not a ast node but within it there can be ast nodes. Need to think if this is
// the correct appraoch.
func WalkTypeData(v Visitor, typeData *TypeData) {
	if w := v.VisitTypeData(typeData); w != nil {
		v = w
	}
	if typeData.TypeDescriptor == nil {
		v.Visit(nil)
		return
	}
	td := typeData.TypeDescriptor
	if tdNode, ok := td.(BLangNode); ok {
		Walk(v, tdNode)
	}
	v.Visit(nil)
}

func walkClassDefnBody(v Visitor, b *classDefnBase) {
	for i := range b.AnnAttachments {
		Walk(v, &b.AnnAttachments[i])
	}
	if b.InitFunction != nil {
		Walk(v, b.InitFunction)
	}
	for _, method := range b.Methods {
		Walk(v, method)
	}
	for _, method := range b.ResourceMethods {
		Walk(v, method)
	}
	for _, field := range b.Fields {
		Walk(v, field.(BLangNode))
	}
	WalkTypeData(v, &b.typeData)
}

func walkTypeDescriptor(v Visitor, td TypeDescriptor) {
	if td == nil {
		return
	}
	if tdNode, ok := td.(BLangNode); ok {
		Walk(v, tdNode)
	}
}
