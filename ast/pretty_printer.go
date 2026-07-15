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
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"ballerina-lang-go/model"
	"ballerina-lang-go/values"
)

// TODO: may be we should rewrite this on top of a visitor.

type PrettyPrinter struct {
	indentLevel        int
	beginningPrinted   bool
	addSpaceBeforeNode bool
	buffer             strings.Builder
	// Fallback handles node types this printer doesn't know about (e.g.
	// desugar-only nodes).
	Fallback func(p *PrettyPrinter, node BLangNode)
}

func (p *PrettyPrinter) Print(node BLangNode) string {
	p.PrintInner(node)
	return p.buffer.String()
}

func (p *PrettyPrinter) PrintInner(node BLangNode) {
	switch t := node.(type) {
	case *BLangPackage:
		p.printPackage(t)
	case *BLangCompilationUnit:
		p.printCompilationUnit(t)
	case *BLangImportPackage:
		p.printImportPackage(t)
	case *BLangFunction:
		p.printFunction(t)
	case *BLangResourceMethod:
		p.printResourceMethod(t)
	case *BLangReturnTypeDescriptor:
		p.printReturnTypeDescriptor(t)
	case *BLangBlockFunctionBody:
		p.printBlockFunctionBody(t)
	case *BLangSimpleVariable:
		p.printSimpleVariable(t)
	case *BLangIf:
		p.printIf(t)
	case *BLangBlockStmt:
		p.printBlockStmt(t)
	case *BLangExpressionStmt:
		p.printExpressionStmt(t)
	case *BLangReturn:
		p.printReturn(t)
	case *BLangSimpleVarRef:
		p.printSimpleVarRef(t)
	case *BLangLiteral:
		p.printLiteral(t)
	case *BLangNumericLiteral:
		p.printNumericLiteral(t)
	case *BLangBinaryExpr:
		p.printBinaryExpr(t)
	case *BLangInvocation:
		p.printInvocation(t)
	case *BLangRemoteMethodCallAction:
		p.printRemoteMethodCallAction(t)
	case *BLangClientResourceAccessAction:
		p.printClientResourceAccessAction(t)
	case *BLangNamedArgsExpression:
		p.printNamedArgsExpression(t)
	case *BLangValueType:
		p.printValueType(t)
	case *BLangBuiltInRefTypeNode:
		p.printBuiltInRefTypeNode(t)
	case *BLangUnaryExpr:
		p.printUnaryExpr(t)
	case *BLangSimpleVariableDef:
		p.printSimpleVariableDef(t)
	case *BLangGroupExpr:
		p.printGroupExpr(t)
	case *BLangWhile:
		p.printWhile(t)
	case *BLangLock:
		p.printLock(t)
	case *BLangForeach:
		p.printForeach(t)
	case *BLangArrayType:
		p.printArrayType(t)
	case *BLangConstant:
		p.printConstant(t)
	case *BLangBreak:
		p.printBreak(t)
	case *BLangContinue:
		p.printContinue(t)
	case *BLangAssignment:
		p.printAssignment(t)
	case *BLangIndexBasedAccess:
		p.printIndexBasedAccess(t)
	case *BLangWildCardBindingPattern:
		p.printWildCardBindingPattern(t)
	case *BLangCompoundAssignment:
		p.printCompoundAssignment(t)
	case *BLangUnionTypeNode:
		p.printUnionTypeNode(t)
	case *BLangIntersectionTypeNode:
		p.printIntersectionTypeNode(t)
	case *BLangErrorTypeNode:
		p.printErrorTypeNode(t)
	case *BLangConstrainedType:
		p.printConstrainedType(t)
	case *BLangStreamType:
		p.printStreamType(t)
	case *BLangTypeDefinition:
		p.printTypeDefinition(t)
	case *BLangUserDefinedType:
		p.printUserDefinedType(t)
	case *BLangFiniteTypeNode:
		p.printFiniteTypeNode(t)
	case *BLangListConstructorExpr:
		p.printListConstructorExpr(t)
	case *BLangMappingConstructorExpr:
		p.printMappingConstructor(t)
	case *BLangAnnotation:
		p.printAnnotation(t)
	case *BLangAnnotationAttachment:
		p.printAnnotationAttachment(t)
	case *BLangAnnotAccessExpr:
		p.printAnnotAccessExpr(t)
	case *BLangTypedescExpr:
		p.printTypedescExpr(t)
	case *BLangTypeConversionExpr:
		p.printTypeConversionExpr(t)
	case *BLangTypeTestExpr:
		p.printTypeTestExpr(t)
	case *BLangTupleTypeNode:
		p.printTupleTypeNode(t)
	case *BLangRecordType:
		p.printRecordType(t)
	case *BLangObjectType:
		p.printObjectType(t)
	case *BObjectField:
		p.printObjectField(t)
	case *BMethodDecl:
		p.printMethodDecl(t)
	case *BLangClassDefinition:
		p.printClassDefinition(t)
	case *BLangService:
		p.printService(t)
	case *BLangNewExpression:
		p.printNewExpression(t)
	case *BLangFieldBaseAccess:
		p.printFieldBaseAccess(t)
	case *BLangErrorConstructorExpr:
		p.printErrorConstructorExpr(t)
	case *BLangQueryExpr:
		p.printQueryExpr(t)
	case *BLangFromClause:
		p.printFromClause(t)
	case *BLangJoinClause:
		p.printJoinClause(t)
	case *BLangLetClause:
		p.printLetClause(t)
	case *BLangOnClause:
		p.printOnClause(t)
	case *BLangWhereClause:
		p.printWhereClause(t)
	case *BLangGroupByClause:
		p.printGroupByClause(t)
	case *BLangGroupingKey:
		p.printGroupingKey(t)
	case *BLangLimitClause:
		p.printLimitClause(t)
	case *BLangOrderByClause:
		p.printOrderByClause(t)
	case *BLangOrderKey:
		p.printOrderKey(t)
	case *BLangSelectClause:
		p.printSelectClause(t)
	case *BLangOnConflictClause:
		p.printOnConflictClause(t)
	case *BLangCollectClause:
		p.printCollectClause(t)
	case *BLangCheckedExpr:
		p.printCheckedExpr(t)
	case *BLangCheckPanickedExpr:
		p.printCheckPanickedExpr(t)
	case *BLangTrapExpr:
		p.printTrapExpr(t)
	case *BLangPanic:
		p.printPanic(t)
	case *BLangMatchStatement:
		p.printMatchStatement(t)
	case *BLangConstPattern:
		p.printConstPattern(t)
	case *BLangWildCardMatchPattern:
		p.printWildCardMatchPattern(t)
	case *BLangMatchClause:
		p.printMatchClause(t)
	case *BLangFunctionType:
		p.printFunctionType(t)
	case *BLangFunctionTypeParam:
		p.printFunctionTypeParam(t)
	case *BLangLambdaFunction:
		p.printLambdaFunction(t)
	case *BLangMarkdownDocumentation:
		p.printMarkdownDocumentation(t)
	case *BLangMarkdownDocumentationLine:
		p.printMarkdownDocumentationLine(t)
	case *BLangMarkdownParameterDocumentation:
		p.printMarkdownParameterDocumentation(t)
	case *BLangMarkdownReturnParameterDocumentation:
		p.printMarkdownReturnParameterDocumentation(t)
	case *BLangMarkDownDeprecationDocumentation:
		p.printMarkDownDeprecationDocumentation(t)
	case *BLangMarkDownDeprecatedParametersDocumentation:
		p.printMarkDownDeprecatedParametersDocumentation(t)
	case *BLangMarkdownReferenceDocumentation:
		p.printMarkdownReferenceDocumentation(t)
	case *BLangXMLSequenceLiteral:
		p.printXMLSequenceLiteral(t)
	case *BLangTemplateExpr:
		p.printTemplateExpr(t)
	case *BLangXMLTemplateExpr:
		p.printXMLTemplateExpr(t)
	case *BLangXMLElementLiteral:
		p.printXMLElementLiteral(t)
	case *BLangXMLAttribute:
		p.printXMLAttribute(t)
	case *BLangXMLPILiteral:
		p.printXMLPILiteral(t)
	case *BLangXMLCommentLiteral:
		p.printXMLCommentLiteral(t)
	case *BLangXMLTextLiteral:
		p.printXMLTextLiteral(t)
	case *BLangXMLNS:
		p.printXMLNS(t)
	default:
		if p.Fallback != nil {
			p.Fallback(p, node)
			return
		}
		fmt.Println(p.buffer.String())
		panic("Unsupported node type: " + reflect.TypeOf(t).String())
	}
}

func (p *PrettyPrinter) printXMLNS(node *BLangXMLNS) {
	p.StartNode()
	p.PrintString("xmlns")
	p.indentLevel++
	if uri := node.GetNamespaceURI(); uri != nil {
		p.PrintInner(uri.(BLangNode))
	}
	if prefix := node.GetPrefix(); prefix != nil {
		p.PrintString("as")
		p.PrintString(prefix.GetValue())
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printCompoundAssignment(t *BLangCompoundAssignment) {
	p.StartNode()
	p.PrintString("compound-assignment")
	p.printOperatorKind(t.OpKind)
	p.indentLevel++
	p.PrintInner(t.VarRef.(BLangNode))
	p.PrintInner(t.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printImportPackage(node *BLangImportPackage) {
	p.StartNode()
	p.PrintString("import-package")
	p.PrintString(node.OrgName.Value)
	for _, pkgNameComp := range node.PkgNameComps {
		p.PrintString(pkgNameComp.Value)
	}
	if node.Alias != nil && node.Alias.Value != "" {
		p.PrintString("(as")
		p.PrintString(node.Alias.Value)
		p.printSticky(")")
	}
	if node.Version != nil && node.Version.Value != "" {
		p.PrintString("(version")
		p.PrintString(node.Version.Value)
		p.printSticky(")")
	}
	p.EndNode()
}

func (p *PrettyPrinter) printCompilationUnit(node *BLangCompilationUnit) {
	p.StartNode()
	p.PrintString("compilation-unit")
	p.PrintString(node.Name)
	p.printSourceKind(node.sourceKind)
	p.printPackageID(node.packageID)
	p.printBLangNodeBase(&node.bLangNodeBase)
	p.indentLevel++
	for _, topLevelNode := range node.TopLevelNodes {
		p.PrintInner(topLevelNode.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printSourceKind(sourceKind SourceKind) {
	switch sourceKind {
	case SourceKind_REGULAR_SOURCE:
		p.PrintString("regular-source")
	case SourceKind_TEST_SOURCE:
		p.PrintString("test-source")
	default:
		panic(fmt.Sprintf("Unsupported source kind: %d", int(sourceKind)))
	}
}

func (p *PrettyPrinter) printPackage(node *BLangPackage) {
	p.StartNode()
	p.PrintString("package")
	p.indentLevel++
	for i := range node.Imports {
		p.printImportPackage(&node.Imports[i])
	}
	for i := range node.Constants {
		p.printConstant(&node.Constants[i])
	}
	for i := range node.GlobalVars {
		p.printSimpleVariable(&node.GlobalVars[i])
	}
	for i := range node.Annotations {
		p.printAnnotation(&node.Annotations[i])
	}
	for i := range node.TypeDefinitions {
		p.printTypeDefinition(&node.TypeDefinitions[i])
	}
	for i := range node.ClassDefinitions {
		p.printClassDefinition(&node.ClassDefinitions[i])
	}
	for i := range node.Services {
		p.printService(&node.Services[i])
	}
	if node.InitFunction != nil {
		p.printFunction(node.InitFunction)
	}
	for i := range node.Functions {
		p.printFunction(&node.Functions[i])
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printBLangNodeBase(node *bLangNodeBase) {
	// no-op
}

func (p *PrettyPrinter) StartNode() {
	if !p.beginningPrinted {
		p.beginningPrinted = true
	} else {
		p.buffer.WriteString("\n")
	}
	for i := 0; i < p.indentLevel; i++ {
		p.buffer.WriteString("  ")
	}
	p.buffer.WriteString("(")
	p.addSpaceBeforeNode = false
}

func (p *PrettyPrinter) EndNode() {
	p.printSticky(")")
}

func (p *PrettyPrinter) printSticky(str string) {
	p.buffer.WriteString(str)
}

func (p *PrettyPrinter) PrintString(str string) {
	if p.addSpaceBeforeNode {
		p.buffer.WriteString(" ")
	}
	p.buffer.WriteString(str)
	p.addSpaceBeforeNode = true
}

func (p *PrettyPrinter) printPackageID(packageID *model.PackageID) {
	if packageID.IsUnnamed() {
		p.PrintString("(unnamed-package)")
	} else {
		p.StartNode()
		p.PrintString("package-id")
		p.PrintString(string(*packageID.OrgName))
		p.PrintString(string(*packageID.PkgName))
		p.PrintString(string(*packageID.Version))
		p.EndNode()
	}
}

// Helper methods
func (p *PrettyPrinter) printOperatorKind(opKind model.OperatorKind) {
	p.PrintString(string(opKind))
}

func (p *PrettyPrinter) printTypeKind(typeKind TypeKind) {
	p.PrintString(string(typeKind))
}

func (p *PrettyPrinter) printAnnotationAttachments(node AnnotatableNode) {
	for _, attachment := range node.GetAnnotationAttachments() {
		p.PrintInner(attachment.(BLangNode))
	}
}

func (p *PrettyPrinter) printReturnTypeDescriptor(node *BLangReturnTypeDescriptor) {
	p.printAnnotationAttachments(node)
	p.PrintInner(node.TypeDescriptor)
}

func (p *PrettyPrinter) printTemplateExpr(node *BLangTemplateExpr) {
	p.StartNode()
	switch node.Kind {
	case TemplateExprKindString:
		p.PrintString("string-template-literal")
	case TemplateExprKindXML:
		p.PrintString("xml-template-literal")
	default:
		panic("unsupported template expr kind")
	}
	p.indentLevel++
	for i, s := range node.Strings {
		p.StartNode()
		p.PrintString("template-string")
		p.PrintString(fmt.Sprintf("%q", s))
		p.EndNode()
		if i < len(node.Insertions) {
			p.PrintInner(node.Insertions[i].(BLangNode))
		}
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printXMLTemplateExpr(node *BLangXMLTemplateExpr) {
	p.StartNode()
	p.PrintString("xml-template-literal")
	p.indentLevel++
	for i, s := range node.Strings {
		p.StartNode()
		p.PrintString("template-string")
		p.PrintString(fmt.Sprintf("%q", s))
		p.EndNode()
		if i < len(node.Insertions) {
			p.StartNode()
			switch node.InsertionKinds[i] {
			case XMLTemplateInsertionKindAttribute:
				p.PrintString("xml-template-attribute-insertion")
			default:
				p.PrintString("xml-template-content-insertion")
			}
			p.indentLevel++
			p.PrintInner(node.Insertions[i].(BLangNode))
			p.indentLevel--
			p.EndNode()
		}
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printXMLSequenceLiteral(node *BLangXMLSequenceLiteral) {
	p.StartNode()
	p.PrintString("xml-sequence-literal")
	p.indentLevel++
	for _, child := range node.Children {
		p.PrintInner(child.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printXMLElementLiteral(node *BLangXMLElementLiteral) {
	p.StartNode()
	p.PrintString("xml-element-literal")
	p.PrintString(node.Name)
	p.indentLevel++
	for i := range node.Attrs {
		p.PrintInner(&node.Attrs[i])
	}
	if node.Content != nil {
		p.PrintInner(node.Content.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printXMLAttribute(node *BLangXMLAttribute) {
	p.StartNode()
	p.PrintString("xml-attribute")
	p.PrintString(node.Name)
	if node.Value != nil {
		p.indentLevel++
		p.PrintInner(node.Value.(BLangNode))
		p.indentLevel--
	}
	p.EndNode()
}

func (p *PrettyPrinter) printXMLPILiteral(node *BLangXMLPILiteral) {
	p.StartNode()
	p.PrintString("xml-pi-literal")
	p.PrintString(node.Target)
	p.PrintString(node.Data)
	p.EndNode()
}

func (p *PrettyPrinter) printXMLCommentLiteral(node *BLangXMLCommentLiteral) {
	p.StartNode()
	p.PrintString("xml-comment-literal")
	p.PrintString(node.Body)
	p.EndNode()
}

func (p *PrettyPrinter) printXMLTextLiteral(node *BLangXMLTextLiteral) {
	p.StartNode()
	p.PrintString("xml-text-literal")
	p.PrintString(node.Body)
	p.EndNode()
}

// Literal and basic expression printers
func (p *PrettyPrinter) printLiteral(node *BLangLiteral) {
	p.StartNode()
	p.PrintString("literal")
	p.PrintString(literalValueString(node.Value))
	p.EndNode()
}

func literalValueString(value any) string {
	switch value.(type) {
	case *values.Map, *values.List, *values.TypeDesc:
		return values.String(value, make(map[uintptr]bool))
	default:
		return fmt.Sprintf("%v", value)
	}
}

func (p *PrettyPrinter) printNumericLiteral(node *BLangNumericLiteral) {
	p.StartNode()
	p.PrintString("numeric-literal")
	p.PrintString(fmt.Sprintf("%v", node.Value))
	p.EndNode()
}

func (p *PrettyPrinter) printSimpleVarRef(node *BLangSimpleVarRef) {
	p.StartNode()
	p.PrintString("simple-var-ref")
	if node.PkgAlias != nil && node.PkgAlias.GetValue() != "" {
		p.PrintString(node.PkgAlias.GetValue() + " " + node.VariableName.GetValue())
	} else {
		p.PrintString(node.VariableName.GetValue())
	}
	p.EndNode()
}

func (p *PrettyPrinter) printAnnotAccessExpr(node *BLangAnnotAccessExpr) {
	p.StartNode()
	p.PrintString("annot-access-expr")
	if node.AnnotationName != nil {
		if node.PkgAlias != nil && node.PkgAlias.GetValue() != "" {
			p.PrintString(node.PkgAlias.GetValue() + " " + node.AnnotationName.GetValue())
		} else {
			p.PrintString(node.AnnotationName.GetValue())
		}
	}
	p.indentLevel++
	if node.Expr != nil {
		p.PrintInner(node.Expr.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printTypedescExpr(node *BLangTypedescExpr) {
	p.StartNode()
	p.PrintString("typedesc-expr")
	p.indentLevel++
	if node.typeDescriptor != nil {
		p.PrintInner(node.typeDescriptor.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Binary and complex expression printers
func (p *PrettyPrinter) printBinaryExpr(node *BLangBinaryExpr) {
	p.StartNode()
	p.PrintString("binary-expr")
	p.printOperatorKind(node.OpKind)
	p.indentLevel++
	p.PrintInner(node.LhsExpr.(BLangNode))
	p.PrintInner(node.RhsExpr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printInvocation(node *BLangInvocation) {
	p.StartNode()
	p.PrintString("invocation")

	// Print function name with optional package alias
	if node.PkgAlias != nil && node.PkgAlias.GetValue() != "" {
		p.PrintString(node.PkgAlias.GetValue() + " " + node.Name.GetValue())
	} else {
		p.PrintString(node.Name.GetValue())
	}

	// Print expression for method calls if present
	if node.Expr != nil {
		p.PrintString("expr:")
		p.indentLevel++
		p.PrintInner(node.Expr.(BLangNode))
		p.indentLevel--
	}

	// Print arguments if present
	p.PrintString("(")
	if len(node.ArgExprs) > 0 {
		p.indentLevel++
		for _, arg := range node.ArgExprs {
			p.PrintInner(arg.(BLangNode))
		}
		p.indentLevel--
	}
	p.printSticky(")")

	p.EndNode()
}

func (p *PrettyPrinter) printResourcePathParamSegment(kind string, seg *BLangResourcePathSegment) {
	p.StartNode()
	p.PrintString(kind)
	p.PrintString(seg.Name)
	if seg.ParamType != nil {
		p.indentLevel++
		p.PrintInner(seg.ParamType.(BLangNode))
		p.indentLevel--
	}
	p.EndNode()
}

func (p *PrettyPrinter) printResourceMethod(node *BLangResourceMethod) {
	p.StartNode()
	p.PrintString("resource-function")
	p.PrintString(node.Name.GetValue())
	p.indentLevel++
	for i := range node.ResourcePath {
		seg := &node.ResourcePath[i]
		switch seg.Kind {
		case ResourcePathSegmentName:
			p.PrintString("name:" + seg.Name)
		case ResourcePathSegmentParam:
			p.printResourcePathParamSegment("param", seg)
		case ResourcePathSegmentParamRest:
			p.printResourcePathParamSegment("rest", seg)
		}
	}
	for i := range node.RequiredParams {
		p.PrintInner(&node.RequiredParams[i])
	}
	if node.GetReturnTypeDescriptor() != nil {
		p.PrintInner(node.GetReturnTypeDescriptor())
	}
	if node.Body != nil {
		p.PrintInner(node.Body.(BLangNode))
	}
	p.printAnnotationAttachments(node)
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printClientResourceAccessAction(node *BLangClientResourceAccessAction) {
	p.StartNode()
	p.PrintString("client-resource-access")
	p.PrintString(node.MethodName)
	p.indentLevel++
	if node.Expr != nil {
		p.PrintString("expr:")
		p.PrintInner(node.Expr)
	}
	for i := range node.Path {
		seg := &node.Path[i]
		switch seg.Kind {
		case ResourceAccessSegmentName:
			p.PrintString("name:" + seg.Name)
		case ResourceAccessSegmentComputed:
			p.PrintString("computed:")
			p.PrintInner(seg.Expr)
		}
	}
	for _, arg := range node.ArgExprs {
		p.PrintInner(arg)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printRemoteMethodCallAction(node *BLangRemoteMethodCallAction) {
	p.StartNode()
	p.PrintString("remote-method-call")
	p.PrintString(node.Name.GetValue())

	if node.Expr != nil {
		p.PrintString("expr:")
		p.indentLevel++
		p.PrintInner(node.Expr.(BLangNode))
		p.indentLevel--
	}

	p.PrintString("(")
	if len(node.ArgExprs) > 0 {
		p.indentLevel++
		for _, arg := range node.ArgExprs {
			p.PrintInner(arg.(BLangNode))
		}
		p.indentLevel--
	}
	p.printSticky(")")

	p.EndNode()
}

func (p *PrettyPrinter) printNamedArgsExpression(node *BLangNamedArgsExpression) {
	p.StartNode()
	p.PrintString("named-arg")
	p.PrintString(node.Name.GetValue())
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Statement printers
func (p *PrettyPrinter) printExpressionStmt(node *BLangExpressionStmt) {
	p.StartNode()
	p.PrintString("expression-stmt")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printReturn(node *BLangReturn) {
	p.StartNode()
	p.PrintString("return")
	if node.Expr != nil {
		p.indentLevel++
		p.PrintInner(node.Expr.(BLangNode))
		p.indentLevel--
	}
	p.EndNode()
}

func (p *PrettyPrinter) printPanic(node *BLangPanic) {
	p.StartNode()
	p.PrintString("panic")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printBlockStmt(node *BLangBlockStmt) {
	p.StartNode()
	p.PrintString("block-stmt")
	p.indentLevel++
	for _, stmt := range node.Stmts {
		p.PrintInner(stmt.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printIf(node *BLangIf) {
	p.StartNode()
	p.PrintString("if")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.PrintInner(&node.Body)
	p.PrintString("(")
	if node.ElseStmt != nil {
		p.PrintInner(node.ElseStmt.(BLangNode))
	}
	p.printSticky(")")
	p.indentLevel--
	p.EndNode()
}

// Type node printers
func (p *PrettyPrinter) printValueType(node *BLangValueType) {
	p.StartNode()
	p.PrintString("value-type")
	p.printTypeKind(node.TypeKind)
	p.EndNode()
}

func (p *PrettyPrinter) printBuiltInRefTypeNode(node *BLangBuiltInRefTypeNode) {
	p.StartNode()
	p.PrintString("builtin-ref-type")
	p.printTypeKind(node.TypeKind)
	p.EndNode()
}

// Variable and function body printers
func (p *PrettyPrinter) printSimpleVariable(node *BLangSimpleVariable) {
	p.StartNode()
	p.PrintString("variable")
	p.PrintString(node.Name.GetValue())
	if node.TypeNode() != nil {
		p.PrintString("(type")
		p.indentLevel++
		p.PrintInner(node.TypeNode().(BLangNode))
		p.indentLevel--
		p.printSticky(")")
	}
	if node.Expr != nil {
		p.PrintString("(expr")
		p.indentLevel++
		p.PrintInner(node.Expr.(BLangNode))
		p.indentLevel--
		p.printSticky(")")
	}
	p.EndNode()
}

func (p *PrettyPrinter) printBlockFunctionBody(node *BLangBlockFunctionBody) {
	p.StartNode()
	p.PrintString("block-function-body")
	p.indentLevel++
	for _, stmt := range node.Stmts {
		p.PrintInner(stmt.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Function printer
func (p *PrettyPrinter) printFunction(node *BLangFunction) {
	p.StartNode()
	p.PrintString("function")

	// Print function name
	p.PrintString(node.Name.GetValue())

	// Print markdown documentation if present
	if node.MarkdownDocumentationAttachment != nil {
		p.indentLevel++
		p.PrintInner(node.MarkdownDocumentationAttachment)
		p.indentLevel--
	}

	// Print parameters
	p.PrintString("(")
	p.indentLevel++
	for i := range node.RequiredParams {
		p.PrintInner(&node.RequiredParams[i])
	}
	p.indentLevel--
	p.printSticky(")")

	// Print return type
	p.PrintString("(")
	if node.GetReturnTypeDescriptor() != nil {
		p.indentLevel++
		p.PrintInner(node.GetReturnTypeDescriptor())
		p.indentLevel--
	}
	p.printSticky(")")

	// Print function body if present
	if node.Body != nil {
		p.indentLevel++
		p.PrintInner(node.Body.(BLangNode))
		p.indentLevel--
	}
	p.indentLevel++
	p.printAnnotationAttachments(node)
	p.indentLevel--

	p.EndNode()
}

// Unary expression printer
func (p *PrettyPrinter) printUnaryExpr(node *BLangUnaryExpr) {
	p.StartNode()
	p.PrintString("unary-expr")
	p.printOperatorKind(node.Operator)
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Variable definition printer
func (p *PrettyPrinter) printSimpleVariableDef(node *BLangSimpleVariableDef) {
	p.StartNode()
	p.PrintString("var-def")
	p.indentLevel++
	p.PrintInner(node.Var)
	p.indentLevel--
	p.EndNode()
}

// Grouped expression printer
func (p *PrettyPrinter) printGroupExpr(node *BLangGroupExpr) {
	p.StartNode()
	p.PrintString("group-expr")
	p.indentLevel++
	p.PrintInner(node.Expression.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printTypeConversionExpr(node *BLangTypeConversionExpr) {
	p.StartNode()
	p.PrintString("type-conversion-expr")
	p.indentLevel++
	p.PrintInner(node.Expression.(BLangNode))
	if node.TypeDescriptor != nil {
		p.PrintInner(node.TypeDescriptor.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printTypeTestExpr(node *BLangTypeTestExpr) {
	p.StartNode()
	if node.isNegation {
		p.PrintString("type-test-expr !is")
	} else {
		p.PrintString("type-test-expr is")
	}
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	if node.Type.TypeDescriptor != nil {
		p.PrintInner(node.Type.TypeDescriptor.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printQueryExpr(node *BLangQueryExpr) {
	p.StartNode()
	p.PrintString("query-expr")
	p.indentLevel++
	for i := range node.QueryClauseList {
		p.PrintInner(node.QueryClauseList[i])
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printFromClause(node *BLangFromClause) {
	p.StartNode()
	p.PrintString("from-clause")
	p.indentLevel++
	if node.VariableDefinitionNode != nil {
		p.PrintInner(node.VariableDefinitionNode)
	}
	if node.Collection != nil {
		p.PrintInner(node.Collection)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printJoinClause(node *BLangJoinClause) {
	p.StartNode()
	if node.IsOuterJoinFlag {
		p.PrintString("join-clause outer")
	} else {
		p.PrintString("join-clause")
	}
	p.indentLevel++
	if node.VariableDefinitionNode != nil {
		p.PrintInner(node.VariableDefinitionNode)
	}
	if node.Collection != nil {
		p.PrintInner(node.Collection)
	}
	if node.OnClause.OnExpr != nil || node.OnClause.EqualsExpr != nil {
		p.PrintInner(&node.OnClause)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printLetClause(node *BLangLetClause) {
	p.StartNode()
	p.PrintString("let-clause")
	p.indentLevel++
	for i := range node.LetVarDeclarations {
		p.PrintInner(&node.LetVarDeclarations[i])
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printOnClause(node *BLangOnClause) {
	p.StartNode()
	p.PrintString("on-clause")
	p.indentLevel++
	if node.OnExpr != nil {
		p.PrintInner(node.OnExpr)
	}
	if node.EqualsExpr != nil {
		p.PrintInner(node.EqualsExpr)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printWhereClause(node *BLangWhereClause) {
	p.StartNode()
	p.PrintString("where-clause")
	p.indentLevel++
	if node.Expression != nil {
		p.PrintInner(node.Expression)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printGroupByClause(node *BLangGroupByClause) {
	p.StartNode()
	p.PrintString("group-by-clause")
	p.indentLevel++
	for _, groupingKey := range node.GetGroupingKeyList() {
		p.PrintInner(groupingKey.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printGroupingKey(node *BLangGroupingKey) {
	p.StartNode()
	p.PrintString("grouping-key")
	p.indentLevel++
	if groupingKey := node.GetGroupingKey(); groupingKey != nil {
		p.PrintInner(groupingKey.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printLimitClause(node *BLangLimitClause) {
	p.StartNode()
	p.PrintString("limit-clause")
	p.indentLevel++
	if node.Expression != nil {
		p.PrintInner(node.Expression)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printOrderByClause(node *BLangOrderByClause) {
	p.StartNode()
	p.PrintString("order-by-clause")
	p.indentLevel++
	for i := range node.OrderByKeyList {
		p.PrintInner(&node.OrderByKeyList[i])
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printOrderKey(node *BLangOrderKey) {
	p.StartNode()
	p.PrintString("order-key")
	if node.IsDescending {
		p.PrintString("descending")
	} else {
		p.PrintString("ascending")
	}
	p.indentLevel++
	if node.Expression != nil {
		p.PrintInner(node.Expression)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printSelectClause(node *BLangSelectClause) {
	p.StartNode()
	p.PrintString("select-clause")
	p.indentLevel++
	if node.Expression != nil {
		p.PrintInner(node.Expression)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printOnConflictClause(node *BLangOnConflictClause) {
	p.StartNode()
	p.PrintString("on-conflict-clause")
	p.indentLevel++
	if node.Expression != nil {
		p.PrintInner(node.Expression)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printCollectClause(node *BLangCollectClause) {
	p.StartNode()
	p.PrintString("collect-clause")
	p.indentLevel++
	if node.Expression != nil {
		p.PrintInner(node.Expression.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// While loop printer
func (p *PrettyPrinter) printWhile(node *BLangWhile) {
	p.StartNode()
	p.PrintString("while")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.PrintInner(&node.Body)
	// OnFailClause handling can be added if needed in the future
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printLock(node *BLangLock) {
	p.StartNode()
	p.PrintString("lock")
	p.indentLevel++
	p.PrintInner(&node.Body)
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printForeach(node *BLangForeach) {
	p.StartNode()
	p.PrintString("foreach")
	p.indentLevel++
	if node.VariableDef != nil {
		p.PrintInner(node.VariableDef)
	}
	if node.Collection != nil {
		p.PrintInner(node.Collection.(BLangNode))
	}
	p.PrintInner(&node.Body)
	p.indentLevel--
	p.EndNode()
}

// Array type printer
func (p *PrettyPrinter) printArrayType(node *BLangArrayType) {
	p.StartNode()
	p.PrintString("array-type")
	p.indentLevel++
	p.PrintInner(node.Elemtype.TypeDescriptor.(BLangNode))
	if node.Dimensions > 0 {
		p.PrintString(fmt.Sprintf("dimensions: %d", node.Dimensions))
	}
	p.PrintString("(")
	if len(node.Sizes) > 0 {
		for _, size := range node.Sizes {
			p.printSticky("[")
			if size != nil {
				p.PrintInner(size.(BLangNode))
			}
			p.printSticky("]")
		}
	}
	p.printSticky(")")
	p.indentLevel--
	p.EndNode()
}

// Constant declaration printer
func (p *PrettyPrinter) printConstant(node *BLangConstant) {
	p.StartNode()
	p.PrintString("const")
	p.PrintString(node.Name.GetValue())

	// Print markdown documentation if present
	if node.MarkdownDocumentationAttachment != nil {
		if bn, ok := node.MarkdownDocumentationAttachment.(BLangNode); ok {
			p.indentLevel++
			p.PrintInner(bn)
			p.indentLevel--
			p.addSpaceBeforeNode = true
		}
	}

	p.PrintString("(")
	if node.TypeNode() != nil {
		p.indentLevel++
		p.PrintInner(node.TypeNode().(BLangNode))
		p.indentLevel--
	}
	p.printSticky(")")
	p.PrintString("(")
	if node.Expr != nil {
		p.indentLevel++
		p.PrintInner(node.Expr.(BLangNode))
		p.indentLevel--
	}
	p.printSticky(")")
	p.EndNode()
}

// Break statement printer
func (p *PrettyPrinter) printBreak(node *BLangBreak) {
	p.StartNode()
	p.PrintString("break")
	p.EndNode()
}

// Continue statement printer
func (p *PrettyPrinter) printContinue(node *BLangContinue) {
	p.StartNode()
	p.PrintString("continue")
	p.EndNode()
}

// Assignment statement printer
func (p *PrettyPrinter) printAssignment(node *BLangAssignment) {
	p.StartNode()
	p.PrintString("assignment")
	p.indentLevel++
	p.PrintInner(node.VarRef.(BLangNode))
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Index-based access expression printer
func (p *PrettyPrinter) printIndexBasedAccess(node *BLangIndexBasedAccess) {
	p.StartNode()
	p.PrintString("index-based-access")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.PrintInner(node.IndexExpr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// List constructor expression printer
func (p *PrettyPrinter) printListConstructorExpr(node *BLangListConstructorExpr) {
	p.StartNode()
	p.PrintString("list-constructor-expr")
	p.indentLevel++
	for _, expr := range node.Exprs {
		p.PrintInner(expr.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMappingConstructor(node *BLangMappingConstructorExpr) {
	p.StartNode()
	p.PrintString("mapping-constructor-expr")
	p.indentLevel++
	for _, f := range node.Fields {
		if kv, ok := f.(*BLangMappingKeyValueField); ok {
			p.printMappingKeyValueField(kv)
		}
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printAnnotation(node *BLangAnnotation) {
	p.StartNode()
	p.PrintString("annotation")
	if node.Name != nil {
		p.PrintString(node.Name.GetValue())
	}
	if node.IsPublic() {
		p.PrintString("public")
	}
	if node.IsConst() {
		p.PrintString("const")
	}
	p.indentLevel++
	p.printAnnotationAttachments(node)
	if node.typeDescriptor != nil {
		p.PrintInner(node.typeDescriptor.(BLangNode))
	}
	attachPoints := node.AttachPoints()
	slices.SortFunc(attachPoints, func(a, b AttachPoint) int {
		if a.Point != b.Point {
			return cmp.Compare(a.Point.String(), b.Point.String())
		}
		if a.Source == b.Source {
			return 0
		}
		if a.Source {
			return 1
		}
		return -1
	})
	for _, attachPoint := range attachPoints {
		p.StartNode()
		p.PrintString("attach-point")
		if attachPoint.Source {
			p.PrintString("source")
		}
		p.PrintString(attachPoint.Point.String())
		p.EndNode()
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printAnnotationAttachment(node *BLangAnnotationAttachment) {
	p.StartNode()
	p.PrintString("annotation-attachment")
	if node.AnnotationName != nil {
		if node.PkgAlias != nil && node.PkgAlias.GetValue() != "" {
			p.PrintString(node.PkgAlias.GetValue() + " " + node.AnnotationName.GetValue())
		} else {
			p.PrintString(node.AnnotationName.GetValue())
		}
	}
	p.indentLevel++
	if node.Expr != nil {
		p.PrintInner(node.Expr.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Mapping key-value field printer: prints as (key-value (key) (value))
func (p *PrettyPrinter) printMappingKeyValueField(kv *BLangMappingKeyValueField) {
	p.StartNode()
	p.PrintString("key-value")
	p.indentLevel++
	if kv.Key != nil && kv.Key.Expr != nil {
		p.PrintInner(kv.Key.Expr.(BLangNode))
	}
	if kv.ValueExpr != nil {
		p.PrintInner(kv.ValueExpr.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Wildcard binding pattern printer
func (p *PrettyPrinter) printWildCardBindingPattern(node *BLangWildCardBindingPattern) {
	p.StartNode()
	p.PrintString("wildcard-binding-pattern")
	p.EndNode()
}

// Finite type node printer
func (p *PrettyPrinter) printFiniteTypeNode(node *BLangFiniteTypeNode) {
	p.StartNode()
	p.PrintString("finite-type")
	p.indentLevel++
	for _, value := range node.ValueSpace {
		p.PrintInner(value.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Union type node printer
func (p *PrettyPrinter) printUnionTypeNode(node *BLangUnionTypeNode) {
	p.StartNode()
	p.PrintString("union-type")
	p.indentLevel++
	p.PrintInner(node.lhs.TypeDescriptor.(BLangNode))
	p.PrintInner(node.rhs.TypeDescriptor.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Intersection type node printer
func (p *PrettyPrinter) printIntersectionTypeNode(node *BLangIntersectionTypeNode) {
	p.StartNode()
	p.PrintString("intersection-type")
	p.indentLevel++
	p.PrintInner(node.lhs.TypeDescriptor.(BLangNode))
	p.PrintInner(node.rhs.TypeDescriptor.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Markdown documentation printers
func (p *PrettyPrinter) printMarkdownDocumentation(node *BLangMarkdownDocumentation) {
	p.StartNode()
	p.PrintString("md-doc")
	p.indentLevel++

	// Print documentation lines
	if len(node.DocumentationLines) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(doc-lines")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for _, line := range node.DocumentationLines {
			p.PrintInner(&line)
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	// Print parameters
	if len(node.Parameters) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(params")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for i := range node.Parameters {
			p.PrintInner(&node.Parameters[i])
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	// Print return parameter
	if node.ReturnParameter != nil {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(return-param")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		p.PrintInner(node.ReturnParameter)
		p.indentLevel--
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	// Print deprecation documentation
	if node.DeprecationDocumentation != nil {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(deprec-doc")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		p.PrintInner(node.DeprecationDocumentation)
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	if node.DeprecatedParametersDocumentation != nil {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(deprec-params-doc")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		p.PrintInner(node.DeprecatedParametersDocumentation)
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	// Print references
	if len(node.References) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(references")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for i := range node.References {
			p.PrintInner(&node.References[i])
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	p.indentLevel--
	p.buffer.WriteString(")")
	p.addSpaceBeforeNode = true
}

// Error type node printer
func (p *PrettyPrinter) printErrorTypeNode(node *BLangErrorTypeNode) {
	p.StartNode()
	p.PrintString("error-type")
	if !node.IsTop() {
		p.indentLevel++
		p.PrintInner(node.DetailType.TypeDescriptor.(BLangNode))
		p.indentLevel--
	}
	p.EndNode()
}

func (p *PrettyPrinter) printMarkdownDocumentationLine(node *BLangMarkdownDocumentationLine) {
	p.StartNode()
	p.PrintString("md-doc-line")
	p.PrintString(fmt.Sprintf("\"%s\"", strings.ReplaceAll(node.Text, "\"", "\\\"")))
	p.EndNode()
}

func (p *PrettyPrinter) printMarkdownParameterDocumentation(node *BLangMarkdownParameterDocumentation) {
	p.StartNode()
	p.PrintString("md-param-doc")
	p.indentLevel++

	// Print parameter name
	if node.ParameterName != nil {
		p.PrintString("(param-name")
		p.PrintString(node.ParameterName.Value)
		p.printSticky(")")
	}

	// Print parameter documentation lines
	if len(node.ParameterDocumentationLines) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(doc-lines")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for _, line := range node.ParameterDocumentationLines {
			p.buffer.WriteString("\n")
			for i := 0; i < p.indentLevel; i++ {
				p.buffer.WriteString("  ")
			}
			fmt.Fprintf(&p.buffer, "\"%s\"", strings.ReplaceAll(line, "\"", "\\\""))
			p.addSpaceBeforeNode = false
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMarkdownReturnParameterDocumentation(node *BLangMarkdownReturnParameterDocumentation) {
	p.StartNode()
	p.PrintString("md-return-param-doc")
	p.indentLevel++

	// Print return parameter documentation lines
	if len(node.ReturnParameterDocumentationLines) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(doc-lines")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for _, line := range node.ReturnParameterDocumentationLines {
			p.buffer.WriteString("\n")
			for i := 0; i < p.indentLevel; i++ {
				p.buffer.WriteString("  ")
			}
			fmt.Fprintf(&p.buffer, "\"%s\"", strings.ReplaceAll(line, "\"", "\\\""))
			p.addSpaceBeforeNode = false
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	// Print return type if present
	if node.ReturnType != nil {
		p.PrintString("(return-type")
		p.indentLevel++
		p.PrintInner(node.ReturnType)
		p.indentLevel--
		p.printSticky(")")
	}

	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMarkDownDeprecationDocumentation(node *BLangMarkDownDeprecationDocumentation) {
	p.StartNode()
	p.PrintString("md-deprec-doc")
	p.indentLevel++

	if len(node.DeprecationDocumentationLines) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(doc-lines")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for _, line := range node.DeprecationDocumentationLines {
			p.buffer.WriteString("\n")
			for i := 0; i < p.indentLevel; i++ {
				p.buffer.WriteString("  ")
			}
			fmt.Fprintf(&p.buffer, "\"%s\"", strings.ReplaceAll(line, "\"", "\\\""))
			p.addSpaceBeforeNode = false
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	if len(node.DeprecationLines) > 0 {
		p.buffer.WriteString("\n")
		for i := 0; i < p.indentLevel; i++ {
			p.buffer.WriteString("  ")
		}
		p.buffer.WriteString("(deprec-lines")
		p.addSpaceBeforeNode = false
		p.indentLevel++
		for _, line := range node.DeprecationLines {
			p.buffer.WriteString("\n")
			for i := 0; i < p.indentLevel; i++ {
				p.buffer.WriteString("  ")
			}
			fmt.Fprintf(&p.buffer, "\"%s\"", strings.ReplaceAll(line, "\"", "\\\""))
			p.addSpaceBeforeNode = false
		}
		p.indentLevel--
		p.buffer.WriteString(")")
		p.addSpaceBeforeNode = false
	}

	if node.IsCorrectDeprecationLine {
		p.PrintString("is-correct-deprec-line")
	}

	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMarkDownDeprecatedParametersDocumentation(node *BLangMarkDownDeprecatedParametersDocumentation) {
	p.StartNode()
	p.PrintString("md-deprec-params-doc")
	p.indentLevel++

	// Print deprecated parameters
	if len(node.Parameters) > 0 {
		p.PrintString("(params")
		p.indentLevel++
		for i := range node.Parameters {
			p.PrintInner(&node.Parameters[i])
		}
		p.indentLevel--
		p.printSticky(")")
	}

	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMarkdownReferenceDocumentation(node *BLangMarkdownReferenceDocumentation) {
	p.StartNode()
	p.PrintString("md-ref-doc")
	p.indentLevel++

	// Print reference type
	p.PrintString("(type")
	p.PrintString(string(node.Type))
	p.printSticky(")")

	// Print qualifier if present
	if node.Qualifier != "" {
		p.PrintString("(qualifier")
		p.PrintString(fmt.Sprintf("\"%s\"", strings.ReplaceAll(node.Qualifier, "\"", "\\\"")))
		p.printSticky(")")
	}

	// Print type name if present
	if node.TypeName != "" {
		p.PrintString("(type-name")
		p.PrintString(fmt.Sprintf("\"%s\"", strings.ReplaceAll(node.TypeName, "\"", "\\\"")))
		p.printSticky(")")
	}

	// Print identifier if present
	if node.Identifier != "" {
		p.PrintString("(identifier")
		p.PrintString(fmt.Sprintf("\"%s\"", strings.ReplaceAll(node.Identifier, "\"", "\\\"")))
		p.printSticky(")")
	}

	// Print reference name
	if node.ReferenceName != "" {
		p.PrintString("(reference-name")
		p.PrintString(fmt.Sprintf("\"%s\"", strings.ReplaceAll(node.ReferenceName, "\"", "\\\"")))
		p.printSticky(")")
	}

	if node.HasParserWarnings {
		p.PrintString("has-parser-warnings")
	}

	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printStreamType(node *BLangStreamType) {
	p.StartNode()
	p.PrintString("stream-type")
	p.indentLevel++
	if node.ValueType.TypeDescriptor != nil {
		p.PrintInner(node.ValueType.TypeDescriptor.(BLangNode))
	}
	if node.CompletionType.TypeDescriptor != nil {
		p.PrintInner(node.CompletionType.TypeDescriptor.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printConstrainedType(node *BLangConstrainedType) {
	p.StartNode()
	p.PrintString("constrained-type")
	p.indentLevel++
	if node.Type.TypeDescriptor != nil {
		p.PrintInner(node.Type.TypeDescriptor.(BLangNode))
	}
	if node.Constraint.TypeDescriptor != nil {
		p.PrintInner(node.Constraint.TypeDescriptor.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Type definition printer
func (p *PrettyPrinter) printTypeDefinition(node *BLangTypeDefinition) {
	p.StartNode()
	p.PrintString("type-definition")
	if node.Name != nil {
		p.PrintString(node.Name.GetValue())
	}
	p.indentLevel++
	p.printAnnotationAttachments(node)
	if node.GetTypeData().TypeDescriptor != nil {
		p.PrintInner(node.GetTypeData().TypeDescriptor.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

// Tuple type node printer
func (p *PrettyPrinter) printTupleTypeNode(node *BLangTupleTypeNode) {
	p.StartNode()
	p.PrintString("tuple-type")
	p.indentLevel++
	for _, member := range node.Members {
		p.PrintInner(member.TypeDesc.(BLangNode))
	}
	if node.Rest != nil {
		p.PrintString("(rest")
		p.indentLevel++
		p.PrintInner(node.Rest.(BLangNode))
		p.indentLevel--
		p.printSticky(")")
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printRecordType(node *BLangRecordType) {
	p.StartNode()
	p.PrintString("record-type")
	p.indentLevel++
	for name, field := range node.FieldPtrs() {
		p.StartNode()
		p.PrintString("field")
		p.PrintString(name)
		if field.IsReadonly() {
			p.PrintString("readonly")
		}
		if field.IsOptional() {
			p.PrintString("optional")
		}
		p.indentLevel++
		p.printAnnotationAttachments(field)
		p.PrintInner(field.Type.(BLangNode))
		if field.DefaultExpr != nil {
			p.PrintInner(field.DefaultExpr.(BLangNode))
		}
		p.indentLevel--
		p.EndNode()
	}
	if node.RestType != nil {
		p.StartNode()
		p.PrintString("rest")
		p.indentLevel++
		p.PrintInner(node.RestType.(BLangNode))
		p.indentLevel--
		p.EndNode()
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printObjectType(node *BLangObjectType) {
	p.StartNode()
	p.PrintString("object-type")
	if node.Isolated {
		p.PrintString("isolated")
	}
	switch node.NetworkQuals {
	case ObjectNetworkQualsClient:
		p.PrintString("client")
	case ObjectNetworkQualsService:
		p.PrintString("service")
	}
	p.indentLevel++
	members := slices.SortedFunc(node.Members(), func(a, b ObjectMember) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	for _, member := range members {
		p.PrintInner(member.(BLangNode))
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printObjectField(node *BObjectField) {
	p.StartNode()
	p.PrintString("field")
	p.PrintString(node.Name())
	if node.IsPublic() {
		p.PrintString("public")
	}
	p.indentLevel++
	p.printAnnotationAttachments(node)
	p.PrintInner(node.Ty.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMethodDecl(node *BMethodDecl) {
	p.StartNode()
	p.PrintString("method-decl")
	p.PrintString(node.Name())
	if node.IsPublic() {
		p.PrintString("public")
	}
	p.PrintString("(")
	if len(node.RequiredParams) > 0 {
		p.indentLevel++
		for _, param := range node.RequiredParams {
			p.StartNode()
			p.PrintString("param")
			if param.Name != nil {
				p.PrintString(param.Name.GetValue())
			}
			p.indentLevel++
			p.PrintInner(param.TypeDesc.(BLangNode))
			p.indentLevel--
			p.EndNode()
		}
		p.indentLevel--
	}
	p.printSticky(")")
	p.PrintString("(")
	if node.ReturnTypeDescriptor != nil {
		p.indentLevel++
		p.PrintInner(node.ReturnTypeDescriptor.(BLangNode))
		p.indentLevel--
	}
	p.printSticky(")")
	p.EndNode()
}

// Field-based access expression printer
func (p *PrettyPrinter) printFieldBaseAccess(node *BLangFieldBaseAccess) {
	p.StartNode()
	if node.IsOptionalAccess() {
		p.PrintString("optional-field-based-access")
	} else {
		p.PrintString("field-based-access")
	}
	p.PrintString(node.Field.GetValue())
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Error constructor expression printer
func (p *PrettyPrinter) printErrorConstructorExpr(node *BLangErrorConstructorExpr) {
	p.StartNode()
	p.PrintString("error-constructor-expr")
	if node.ErrorTypeRef != nil {
		p.indentLevel++
		p.PrintInner(node.ErrorTypeRef)
		p.indentLevel--
	}
	p.PrintString("(")
	if len(node.PositionalArgs) > 0 {
		p.indentLevel++
		for _, arg := range node.PositionalArgs {
			p.PrintInner(arg.(BLangNode))
		}
		p.indentLevel--
	}
	p.printSticky(")")
	if len(node.NamedArgs) > 0 {
		p.PrintString("(")
		p.indentLevel++
		for _, namedArg := range node.NamedArgs {
			p.StartNode()
			p.PrintString("named-arg")
			p.PrintString(namedArg.Name.GetValue())
			p.indentLevel++
			p.PrintInner(namedArg.Expr.(BLangNode))
			p.indentLevel--
			p.EndNode()
		}
		p.indentLevel--
		p.printSticky(")")
	}
	p.EndNode()
}

// Checked expression printer
func (p *PrettyPrinter) printCheckedExpr(node *BLangCheckedExpr) {
	p.StartNode()
	p.PrintString("checked-expr")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

// Check panicked expression printer
func (p *PrettyPrinter) printCheckPanickedExpr(node *BLangCheckPanickedExpr) {
	p.StartNode()
	p.PrintString("check-panicked-expr")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printTrapExpr(node *BLangTrapExpr) {
	p.StartNode()
	p.PrintString("trap-expr")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printClassDefinition(node *BLangClassDefinition) {
	p.StartNode()
	p.PrintString("class-definition")
	if node.IsPublic() {
		p.PrintString("public")
	}
	if node.IsDistinct() {
		p.PrintString("distinct")
	}
	p.PrintString(node.Name.GetValue())
	p.indentLevel++
	p.printAnnotationAttachments(node)
	// Print fields
	for _, field := range node.Fields {
		p.PrintInner(field.(BLangNode))
	}
	// Print init function
	if node.InitFunction != nil {
		p.printFunction(node.InitFunction)
	}
	// Print methods sorted by name for determinism
	methodNames := slices.SortedFunc(func(yield func(string) bool) {
		for name := range node.Methods {
			if !yield(name) {
				return
			}
		}
	}, cmp.Compare[string])
	for _, name := range methodNames {
		method := node.Methods[name]
		p.printFunction(method)
	}
	for _, rm := range node.ResourceMethods {
		p.printResourceMethod(rm)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printService(node *BLangService) {
	p.StartNode()
	p.PrintString("service")
	if node.IsIsolated() {
		p.PrintString("isolated")
	}
	if node.GetTypeData().TypeDescriptor != nil {
		p.indentLevel++
		p.PrintInner(node.GetTypeData().TypeDescriptor.(BLangNode))
		p.indentLevel--
	}
	if node.AttachPointLiteral != nil {
		p.indentLevel++
		p.PrintInner(node.AttachPointLiteral)
		p.indentLevel--
	} else if len(node.AbsoluteResourcePath) > 0 {
		p.indentLevel++
		p.StartNode()
		p.PrintString("absolute-resource-path")
		for i := range node.AbsoluteResourcePath {
			p.PrintString(node.AbsoluteResourcePath[i].Value)
		}
		p.EndNode()
		p.indentLevel--
	}
	p.indentLevel++
	p.StartNode()
	p.PrintString("on")
	for _, expr := range node.AttachedExprs {
		p.PrintInner(expr.(BLangNode))
	}
	p.EndNode()
	// Print the embedded class members.
	for _, field := range node.Fields {
		p.PrintInner(field.(BLangNode))
	}
	if node.InitFunction != nil {
		p.printFunction(node.InitFunction)
	}
	methodNames := slices.SortedFunc(func(yield func(string) bool) {
		for name := range node.Methods {
			if !yield(name) {
				return
			}
		}
	}, cmp.Compare[string])
	for _, name := range methodNames {
		p.printFunction(node.Methods[name])
	}
	for _, rm := range node.ResourceMethods {
		p.printResourceMethod(rm)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printNewExpression(node *BLangNewExpression) {
	p.StartNode()
	p.PrintString("new")
	if node.TypeDescriptor != nil {
		p.indentLevel++
		p.PrintInner(node.TypeDescriptor)
		p.indentLevel--
	}
	p.PrintString("(")
	if len(node.ArgsExprs) > 0 {
		p.indentLevel++
		for _, arg := range node.ArgsExprs {
			p.PrintInner(arg.(BLangNode))
		}
		p.indentLevel--
	}
	p.printSticky(")")
	p.EndNode()
}

// Function type printer
func (p *PrettyPrinter) printFunctionType(node *BLangFunctionType) {
	p.StartNode()
	p.PrintString("function-type")
	p.PrintString("(")
	if len(node.RequiredParams) > 0 {
		p.indentLevel++
		for i := range node.RequiredParams {
			param := &node.RequiredParams[i]
			if param.TypeDesc != nil {
				p.PrintInner(param.TypeDesc.(BLangNode))
			}
		}
		p.indentLevel--
	}
	if node.RestParam != nil {
		p.indentLevel++
		p.PrintInner(node.RestParam)
		p.indentLevel--
	}
	p.printSticky(")")
	p.PrintString("(")
	if node.ReturnTypeDescriptor != nil {
		p.indentLevel++
		p.PrintInner(node.ReturnTypeDescriptor.(BLangNode))
		p.indentLevel--
	}
	p.printSticky(")")
	p.EndNode()
}

func (p *PrettyPrinter) printLambdaFunction(node *BLangLambdaFunction) {
	p.StartNode()
	p.PrintString("lambda")
	if node.Function != nil {
		p.indentLevel++
		p.PrintInner(node.Function)
		p.indentLevel--
	}
	p.EndNode()
}

func (p *PrettyPrinter) printFunctionTypeParam(node *BLangFunctionTypeParam) {
	p.StartNode()
	p.PrintString("function-type-param")
	if node.Name != nil {
		p.PrintInner(node.Name)
	}
	if node.TypeDesc != nil {
		p.PrintInner(node.TypeDesc.(BLangNode))
	}
	p.EndNode()
}

// User-defined type printer
func (p *PrettyPrinter) printUserDefinedType(node *BLangUserDefinedType) {
	p.StartNode()
	p.PrintString("user-defined-type")
	if node.PkgAlias.GetValue() != "" {
		p.PrintString(node.PkgAlias.GetValue() + " " + node.TypeName.Value)
	} else {
		p.PrintString(node.TypeName.Value)
	}
	p.EndNode()
}

// Match statement printer
func (p *PrettyPrinter) printMatchStatement(node *BLangMatchStatement) {
	p.StartNode()
	p.PrintString("match")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	for i := range node.MatchClauses {
		clause := &node.MatchClauses[i]
		p.printMatchClause(clause)
	}
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printMatchClause(node *BLangMatchClause) {
	p.StartNode()
	p.PrintString("match-clause")
	p.indentLevel++
	// Print patterns
	for _, pattern := range node.Patterns {
		p.PrintInner(pattern.(BLangNode))
	}
	// Print guard if present
	if node.Guard != nil {
		p.StartNode()
		p.PrintString("match-guard")
		p.indentLevel++
		p.PrintInner(node.Guard)
		p.indentLevel--
		p.EndNode()
	}
	p.PrintInner(&node.Body)
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printConstPattern(node *BLangConstPattern) {
	p.StartNode()
	p.PrintString("const-pattern")
	p.indentLevel++
	p.PrintInner(node.Expr.(BLangNode))
	p.indentLevel--
	p.EndNode()
}

func (p *PrettyPrinter) printWildCardMatchPattern(node *BLangWildCardMatchPattern) {
	p.StartNode()
	p.PrintString("wildcard-match-pattern")
	p.EndNode()
}
