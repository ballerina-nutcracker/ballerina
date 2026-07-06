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
	"iter"
	"strings"

	"ballerina-lang-go/common"
	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/parser/tree"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/values"
)

type BNodeWithSymbol interface {
	NodeWithSymbol
	BLangNode
	SetSymbol(symbolRef model.SymbolRef)
}

// SymbolIsSet returns true if the AST node has its symbol set.
func SymbolIsSet(node NodeWithSymbol) bool {
	return !node.Symbol().IsEmpty()
}

type NodeWithScope interface {
	Scope() model.Scope
	SetScope(scope model.Scope)
}

type Location = diagnostics.Location

type SourceKind uint8

const (
	SourceKind_REGULAR_SOURCE SourceKind = iota
	SourceKind_TEST_SOURCE
)

type BLangNode interface {
	Node
	SetDeterminedType(ty semtypes.SemType)
	SetPosition(pos diagnostics.Location)
}

type (
	bLangNodeBase struct {
		DeterminedType semtypes.SemType
		parent         BLangNode
		pos            diagnostics.Location
	}

	BLangAnnotation struct {
		bLangNodeBase
		Name                            *BLangIdentifier
		symbol                          model.SymbolRef
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
		typeDescriptor                  TypeDescriptor
		attachPoints                    common.UnorderedSet[AttachPoint]
		flags                           model.Flag
	}

	BLangAnnotationAttachment struct {
		bLangNodeBase
		Expr            BLangExpression
		HasValue        bool
		AnnotationName  *BLangIdentifier
		PkgAlias        *BLangIdentifier
		symbol          model.SymbolRef
		AnnotationValue values.AnnotationValue
	}

	bLangFunctionBodyBase struct {
		bLangNodeBase
	}
)

func (*bLangFunctionBodyBase) isFunctionBody() {}

type (
	BLangBlockFunctionBody struct {
		bLangFunctionBodyBase
		Stmts []StatementNode
	}

	BLangExprFunctionBody struct {
		bLangFunctionBodyBase
		Expr BLangExpression
	}

	BLangExternFunctionBody struct {
		bLangFunctionBodyBase
		AnnAttachments []BLangAnnotationAttachment
	}

	BLangIdentifier struct {
		bLangNodeBase
		Value         string
		OriginalValue string
		isLiteral     bool
	}

	BLangImportPackage struct {
		bLangNodeBase
		OrgName      *BLangIdentifier
		PkgNameComps []BLangIdentifier
		Alias        *BLangIdentifier
		CompUnit     *BLangIdentifier
		Version      *BLangIdentifier
	}

	classDefnBase struct {
		bLangNodeBase
		scope                           model.Scope
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
		InitFunction                    *BLangFunction
		Methods                         map[string]*BLangFunction
		ResourceMethods                 []*BLangResourceMethod
		Fields                          []SimpleVariableNode
		Inclusions                      []model.SymbolRef       // This needs to be symbol because it could be a class definition as well
		InclusionPositions              []diagnostics.Location  // Positions of each inclusion, parallel to Inclusions
		unresolvedInclusions            []*BLangUserDefinedType // we need this because we can't get symbols before the symbol resolution in node_builder. After symbol resolution this field is cleared out
		flags                           model.Flag
		typeData                        TypeData
		Definition                      semtypes.Definition
		CycleDepth                      int
	}

	BLangClassDefinition struct {
		classDefnBase
		Name   *BLangIdentifier
		symbol model.SymbolRef
	}

	BLangService struct {
		classDefnBase
		AttachedExprs []BLangExpression
		// attach point either AbsoluteResourcePath or AttachPointLiteral
		AbsoluteResourcePath []BLangIdentifier
		AttachPointLiteral   *BLangLiteral
	}

	BLangCompilationUnit struct {
		bLangNodeBase
		TopLevelNodes []TopLevelNode
		Name          string
		Scope         model.Scope
		packageID     *model.PackageID
		sourceKind    SourceKind
	}

	BLangPackage struct {
		bLangNodeBase
		Imports          []BLangImportPackage
		XmlnsList        []BLangXMLNS
		Constants        []BLangConstant
		GlobalVars       []BLangSimpleVariable
		Services         []BLangService
		Functions        []BLangFunction
		TypeDefinitions  []BLangTypeDefinition
		Annotations      []BLangAnnotation
		InitFunction     *BLangFunction
		TestablePkgs     []*BLangTestablePackage
		ClassDefinitions []BLangClassDefinition
		PackageID        *model.PackageID
		Scope            model.Scope
	}
	BLangTestablePackage struct {
		BLangPackage
		Parent               *BLangPackage
		mockFunctionNamesMap map[string]string
		isLegacyMockingMap   map[string]bool
	}
	BLangXMLNS struct {
		bLangNodeBase
		namespaceURI BLangExpression
		prefix       *BLangIdentifier
	}
	BLangMarkdownDocumentation struct {
		bLangNodeBase
		DocumentationLines                []BLangMarkdownDocumentationLine
		Parameters                        []BLangMarkdownParameterDocumentation
		References                        []BLangMarkdownReferenceDocumentation
		ReturnParameter                   *BLangMarkdownReturnParameterDocumentation
		DeprecationDocumentation          *BLangMarkDownDeprecationDocumentation
		DeprecatedParametersDocumentation *BLangMarkDownDeprecatedParametersDocumentation
	}
	BLangMarkdownReferenceDocumentation struct {
		bLangNodeBase
		Qualifier         string
		TypeName          string
		Identifier        string
		ReferenceName     string
		Type              DocumentationReferenceType
		HasParserWarnings bool
	}

	BLangVariableBase struct {
		bLangNodeBase
		// We are using variable for function paramets and record td fields so we need to have
		// type descriptors here. Not sure this is the best way to do this.
		typeNode                        BType
		AnnAttachments                  []AnnotationAttachmentNode
		MarkdownDocumentationAttachment MarkdownDocumentationNode
		Expr                            BLangActionOrExpression
		flags                           model.Flag
		IsDeclaredWithVar               bool
		symbol                          model.SymbolRef
	}

	BLangConstant struct {
		BLangVariableBase
		Name *BLangIdentifier
	}

	BLangSimpleVariable struct {
		BLangVariableBase
		Name *BLangIdentifier
	}

	ClosureVarSymbol struct {
		DiagnosticLocation diagnostics.Location
	}

	bLangInvokableNodeBase struct {
		bLangNodeBase
		Name                            BLangIdentifier
		symbol                          model.SymbolRef
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
		RequiredParams                  []BLangSimpleVariable
		RestParam                       SimpleVariableNode
		returnTypeDescriptor            *BLangReturnTypeDescriptor
		Body                            FunctionBodyNode
		flags                           model.Flag
		scope                           model.Scope
	}

	BLangFunction struct {
		bLangInvokableNodeBase
	}

	BLangResourcePathSegment struct {
		bLangNodeBase
		Kind      ResourcePathSegmentKind
		Name      string
		ParamType BType
	}

	BLangResourceMethod struct {
		bLangInvokableNodeBase
		ResourcePath []BLangResourcePathSegment
	}

	BLangTypeDefinition struct {
		bLangNodeBase
		Name                            *BLangIdentifier
		symbol                          model.SymbolRef
		typeData                        TypeData
		annAttachments                  []BLangAnnotationAttachment
		markdownDocumentationAttachment *BLangMarkdownDocumentation
		flags                           model.Flag
		CycleDepth                      int
		isBuiltinTypeDef                bool
		hasCyclicReference              bool
		referencedFieldsDefined         bool
	}
)

// bLangInvokableNodeBase flag methods
func (b *bLangInvokableNodeBase) IsPublic() bool        { return b.flags.Has(model.FlagPublic) }
func (b *bLangInvokableNodeBase) IsRemote() bool        { return b.flags.Has(model.FlagRemote) }
func (b *bLangInvokableNodeBase) IsTransactional() bool { return b.flags.Has(model.FlagTransactional) }
func (b *bLangInvokableNodeBase) IsResource() bool      { return b.flags.Has(model.FlagResource) }
func (b *bLangInvokableNodeBase) IsIsolated() bool      { return b.flags.Has(model.FlagIsolated) }
func (b *bLangInvokableNodeBase) IsInterface() bool     { return b.flags.Has(model.FlagInterface) }
func (b *bLangInvokableNodeBase) IsNative() bool        { return b.flags.Has(model.FlagNative) }
func (b *bLangInvokableNodeBase) IsAnonymous() bool     { return b.flags.Has(model.FlagLambda) }
func (b *bLangInvokableNodeBase) IsAttached() bool      { return b.flags.Has(model.FlagAttached) }

func (b *bLangInvokableNodeBase) SetPublic()        { b.flags |= model.FlagPublic }
func (b *bLangInvokableNodeBase) SetRemote()        { b.flags |= model.FlagRemote }
func (b *bLangInvokableNodeBase) SetTransactional() { b.flags |= model.FlagTransactional }
func (b *bLangInvokableNodeBase) SetResource()      { b.flags |= model.FlagResource }
func (b *bLangInvokableNodeBase) SetIsolated()      { b.flags |= model.FlagIsolated }
func (b *bLangInvokableNodeBase) SetInterface()     { b.flags |= model.FlagInterface }
func (b *bLangInvokableNodeBase) SetNative()        { b.flags |= model.FlagNative }
func (b *bLangInvokableNodeBase) SetAnonymous()     { b.flags |= model.FlagLambda | model.FlagAnonymous }
func (b *bLangInvokableNodeBase) SetAttached()      { b.flags |= model.FlagAttached }
func (b *bLangInvokableNodeBase) Flags() model.Flag { return b.flags }

func (b *bLangInvokableNodeBase) FuncSymbolFlags() model.FuncSymbolFlags {
	return model.FuncSymbolFlags(b.flags)
}

// BLangVariableBase flag methods
func (b *BLangVariableBase) IsPublic() bool           { return b.flags.Has(model.FlagPublic) }
func (b *BLangVariableBase) IsFinal() bool            { return b.flags.Has(model.FlagFinal) }
func (b *BLangVariableBase) IsConfigurable() bool     { return b.flags.Has(model.FlagConfigurable) }
func (b *BLangVariableBase) IsDefaultableParam() bool { return b.flags.Has(model.FlagDefaultableParam) }
func (b *BLangVariableBase) IsRequiredParam() bool    { return b.flags.Has(model.FlagRequiredParam) }
func (b *BLangVariableBase) IsRestParam() bool        { return b.flags.Has(model.FlagRestParam) }
func (b *BLangVariableBase) IsIncludedRecordParam() bool {
	return b.flags.Has(model.FlagIncluded)
}

func (b *BLangVariableBase) SetPublic()              { b.flags |= model.FlagPublic }
func (b *BLangVariableBase) SetPrivate()             { b.flags &^= model.FlagPublic }
func (b *BLangVariableBase) SetFinal()               { b.flags |= model.FlagFinal }
func (b *BLangVariableBase) SetConfigurable()        { b.flags |= model.FlagConfigurable }
func (b *BLangVariableBase) SetIsolated()            { b.flags |= model.FlagIsolated }
func (b *BLangVariableBase) SetDefaultableParam()    { b.flags |= model.FlagDefaultableParam }
func (b *BLangVariableBase) SetRequiredParam()       { b.flags |= model.FlagRequiredParam }
func (b *BLangVariableBase) SetRestParam()           { b.flags |= model.FlagRestParam }
func (b *BLangVariableBase) SetIncludedRecordParam() { b.flags |= model.FlagIncluded }
func (b *BLangVariableBase) IsReadonly() bool        { return b.flags.Has(model.FlagReadonly) }
func (b *BLangVariableBase) IsListener() bool        { return b.flags.Has(model.FlagListener) }
func (b *BLangVariableBase) SetListener()            { b.flags |= model.FlagListener }
func (b *BLangVariableBase) Flags() model.Flag       { return b.flags }

// classDefnBase flag methods (promoted to BLangClassDefinition / BLangService)
func (b *classDefnBase) IsPublic() bool   { return b.flags.Has(model.FlagPublic) }
func (b *classDefnBase) IsDistinct() bool { return b.flags.Has(model.FlagDistinct) }
func (b *classDefnBase) IsClient() bool   { return b.flags.Has(model.FlagClient) }
func (b *classDefnBase) IsReadonly() bool { return b.flags.Has(model.FlagReadonly) }
func (b *classDefnBase) IsService() bool  { return b.flags.Has(model.FlagService) }
func (b *classDefnBase) IsIsolated() bool { return b.flags.Has(model.FlagIsolated) }

func (b *classDefnBase) SetPublic()        { b.flags |= model.FlagPublic }
func (b *classDefnBase) SetDistinct()      { b.flags |= model.FlagDistinct }
func (b *classDefnBase) SetClient()        { b.flags |= model.FlagClient }
func (b *classDefnBase) SetReadonly()      { b.flags |= model.FlagReadonly }
func (b *classDefnBase) SetService()       { b.flags |= model.FlagService }
func (b *classDefnBase) SetIsolated()      { b.flags |= model.FlagIsolated }
func (b *classDefnBase) SetClass()         { b.flags |= model.FlagClass }
func (b *classDefnBase) Flags() model.Flag { return b.flags }

// BLangTypeDefinition flag methods
func (b *BLangTypeDefinition) IsPublic() bool    { return b.flags.Has(model.FlagPublic) }
func (b *BLangTypeDefinition) IsAnonymous() bool { return b.flags.Has(model.FlagAnonymous) }
func (b *BLangTypeDefinition) IsDistinct() bool  { return b.flags.Has(model.FlagDistinct) }
func (b *BLangTypeDefinition) SetPublic()        { b.flags |= model.FlagPublic }
func (b *BLangTypeDefinition) SetAnonymous()     { b.flags |= model.FlagAnonymous }
func (b *BLangTypeDefinition) SetDistinct()      { b.flags |= model.FlagDistinct }

func (b *BLangAnnotation) IsPublic() bool { return b.flags.Has(model.FlagPublic) }
func (b *BLangAnnotation) IsConst() bool  { return b.flags.Has(model.FlagConstant) }
func (b *BLangAnnotation) SetPublic()     { b.flags |= model.FlagPublic }
func (b *BLangAnnotation) SetConst()      { b.flags |= model.FlagConstant }

// Stub IsPublic for types with no flags
func (b *BLangService) IsPublic() bool        { return false }
func (b *BLangMemberTypeDesc) IsPublic() bool { return false }

func (b *bLangNodeBase) SetDeterminedType(ty semtypes.SemType) {
	b.DeterminedType = ty
}

func (b *bLangNodeBase) GetDeterminedType() semtypes.SemType {
	return b.DeterminedType
}

func (b *bLangNodeBase) GetPosition() diagnostics.Location {
	return b.pos
}

func (b *bLangNodeBase) SetPosition(pos diagnostics.Location) {
	b.pos = pos
}

func (n *BLangClassDefinition) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangClassDefinition) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *classDefnBase) Scope() model.Scope {
	return n.scope
}

func (n *classDefnBase) SetScope(scope model.Scope) {
	n.scope = scope
}

func (n *BLangVariableBase) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangVariableBase) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *BLangVariableBase) TypeNode() BType {
	return n.typeNode
}

func (n *BLangVariableBase) SetTypeNode(bt BType) {
	n.typeNode = bt
}

func (n *bLangInvokableNodeBase) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *bLangInvokableNodeBase) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *BLangTypeDefinition) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangTypeDefinition) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *BLangAnnotation) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangAnnotation) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *BLangAnnotationAttachment) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangAnnotationAttachment) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

var (
	_ AnnotationAttachmentNode                    = &BLangAnnotationAttachment{}
	_ BNodeWithSymbol                             = &BLangAnnotation{}
	_ BNodeWithSymbol                             = &BLangAnnotationAttachment{}
	_ ImportPackageNode                           = &BLangImportPackage{}
	_ ClassDefinition                             = &BLangClassDefinition{}
	_ NodeWithScope                               = &BLangClassDefinition{}
	_ PackageNode                                 = &BLangPackage{}
	_ PackageNode                                 = &BLangTestablePackage{}
	_ AnnotationNode                              = &BLangAnnotation{}
	_ XMLNSDeclarationNode                        = &BLangXMLNS{}
	_ ServiceNode                                 = &BLangService{}
	_ CompilationUnitNode                         = &BLangCompilationUnit{}
	_ ConstantNode                                = &BLangConstant{}
	_ SimpleVariableNode                          = &BLangSimpleVariable{}
	_ MarkdownDocumentationNode                   = &BLangMarkdownDocumentation{}
	_ MarkdownDocumentationReferenceAttributeNode = &BLangMarkdownReferenceDocumentation{}
	_ ExprFunctionBodyNode                        = &BLangExprFunctionBody{}
	_ FunctionNode                                = &BLangFunction{}
	_ FunctionBodyNode                            = &BLangExternFunctionBody{}
)

func (*BLangImportPackage) isTopLevel()   {}
func (*BLangXMLNS) isTopLevel()           {}
func (*BLangAnnotation) isTopLevel()      {}
func (*BLangSimpleVariable) isTopLevel()  {}
func (*BLangFunction) isTopLevel()        {}
func (*BLangClassDefinition) isTopLevel() {}
func (*BLangService) isTopLevel()         {}
func (*BLangTypeDefinition) isTopLevel()  {}
func (*BLangConstant) isTopLevel()        {}

var (
	_ BLangNode = &BLangAnnotation{}
	_ BLangNode = &BLangAnnotationAttachment{}
	_ BLangNode = &BLangBlockFunctionBody{}
	_ BLangNode = &BLangExprFunctionBody{}
	_ BLangNode = &BLangIdentifier{}
	_ BLangNode = &BLangImportPackage{}
	_ BLangNode = &BLangClassDefinition{}
	_ BLangNode = &BLangService{}
	_ BLangNode = &BLangCompilationUnit{}
	_ BLangNode = &BLangPackage{}
	_ BLangNode = &BLangTestablePackage{}
	_ BLangNode = &BLangXMLNS{}
	_ BLangNode = &BLangMarkdownDocumentation{}
	_ BLangNode = &BLangMarkdownReferenceDocumentation{}
	_ BLangNode = &BLangConstant{}
	_ BLangNode = &BLangSimpleVariable{}
	_ BLangNode = &BLangFunction{}
	_ BLangNode = &BLangTypeDefinition{}
)

var (
	// Assert that concrete types with symbols implement BNodeWithSymbol
	_ BNodeWithSymbol = &BLangClassDefinition{}
	_ BNodeWithSymbol = &BLangConstant{}
	_ BNodeWithSymbol = &BLangSimpleVariable{}
	_ BNodeWithSymbol = &BLangFunction{}
	_ BNodeWithSymbol = &BLangTypeDefinition{}
)

func (b *BLangAnnotationAttachment) GetPackageAlias() *BLangIdentifier {
	return b.PkgAlias
}

func (b *BLangAnnotationAttachment) SetPackageAlias(pkgAlias *BLangIdentifier) {
	b.PkgAlias = pkgAlias
}

func (b *BLangAnnotationAttachment) GetAnnotationName() *BLangIdentifier {
	return b.AnnotationName
}

func (b *BLangAnnotationAttachment) SetAnnotationName(name *BLangIdentifier) {
	b.AnnotationName = name
}

func (b *BLangAnnotationAttachment) GetExpressionNode() BLangExpression {
	return b.Expr
}

func (b *BLangAnnotationAttachment) SetExpressionNode(expr BLangExpression) {
	b.Expr = expr
}

func (b *BLangAnnotation) GetName() *BLangIdentifier {
	return b.Name
}

func (b *BLangAnnotation) SetName(name *BLangIdentifier) {
	b.Name = name
}

func (b *BLangAnnotation) GetTypeDescriptor() TypeDescriptor {
	if b.typeDescriptor == nil {
		return nil
	}
	return b.typeDescriptor
}

func (b *BLangAnnotation) SetTypeDescriptor(typeDescriptor TypeDescriptor) {
	if typeDescriptor == nil {
		b.typeDescriptor = nil
		return
	}
	b.typeDescriptor = typeDescriptor.(BType)
}

// AddAttachPoint adds an attachment point and is safe on a zero-value annotation.
func (b *BLangAnnotation) AddAttachPoint(attachPoint AttachPoint) {
	b.attachPoints.Add(attachPoint)
}

func (b *BLangAnnotation) AttachPoints() []AttachPoint {
	result := []AttachPoint{}
	for attachPoint := range b.attachPoints.Values() {
		result = append(result, attachPoint)
	}
	return result
}

func (b *BLangAnnotation) HasSourceAttachPoint() bool {
	for attachPoint := range b.attachPoints.Values() {
		if attachPoint.Source {
			return true
		}
	}
	return false
}

func (b *BLangAnnotation) GetAnnotationAttachments() []AnnotationAttachmentNode {
	attachments := make([]AnnotationAttachmentNode, len(b.AnnAttachments))
	for i := range b.AnnAttachments {
		attachments[i] = &b.AnnAttachments[i]
	}
	return attachments
}

func (b *BLangAnnotation) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	if annAttachment, ok := annAttachment.(*BLangAnnotationAttachment); ok {
		b.AnnAttachments = append(b.AnnAttachments, *annAttachment)
		return
	}
	panic("annAttachment is not a BLangAnnotationAttachment")
}

func (b *BLangAnnotation) GetMarkdownDocumentationAttachment() MarkdownDocumentationNode {
	return b.MarkdownDocumentationAttachment
}

func (b *BLangAnnotation) SetMarkdownDocumentationAttachment(documentationNode MarkdownDocumentationNode) {
	if documentationNode, ok := documentationNode.(*BLangMarkdownDocumentation); ok {
		b.MarkdownDocumentationAttachment = documentationNode
		return
	}
	panic("documentationNode is not a BLangMarkdownDocumentation")
}

func (b *BLangExprFunctionBody) GetExpr() BLangExpression {
	return b.Expr
}

func (b *BLangIdentifier) GetValue() string {
	return b.Value
}

func (b *BLangIdentifier) SetValue(value string) {
	b.Value = value
}

func (b *BLangIdentifier) SetOriginalValue(value string) {
	b.OriginalValue = value
}

func (b *BLangIdentifier) IsLiteral() bool {
	return b.isLiteral
}

func (b *BLangIdentifier) SetLiteral(isLiteral bool) {
	b.isLiteral = isLiteral
}

func (b *BLangImportPackage) GetOrgName() *BLangIdentifier {
	return b.OrgName
}

func (b *BLangImportPackage) GetPackageName() []*BLangIdentifier {
	result := make([]*BLangIdentifier, len(b.PkgNameComps))
	for i := range b.PkgNameComps {
		result[i] = &b.PkgNameComps[i]
	}
	return result
}

func (b *BLangImportPackage) SetPackageName(nameParts []*BLangIdentifier) {
	b.PkgNameComps = make([]BLangIdentifier, 0, len(nameParts))
	for _, namePart := range nameParts {
		b.PkgNameComps = append(b.PkgNameComps, *namePart)
	}
}

func (b *BLangImportPackage) GetPackageVersion() *BLangIdentifier {
	return b.Version
}

func (b *BLangImportPackage) SetPackageVersion(version *BLangIdentifier) {
	b.Version = version
}

func (b *BLangImportPackage) GetAlias() *BLangIdentifier {
	return b.Alias
}

func (b *BLangImportPackage) SetAlias(alias *BLangIdentifier) {
	b.Alias = alias
}

func newClassDefnBase() classDefnBase {
	b := classDefnBase{}
	b.CycleDepth = -1
	b.Methods = map[string]*BLangFunction{}
	return b
}

func NewBLangClassDefinition() BLangClassDefinition {
	b := BLangClassDefinition{classDefnBase: newClassDefnBase()}
	b.SetClass()
	return b
}

func NewBLangService() BLangService {
	b := BLangService{classDefnBase: newClassDefnBase()}
	b.SetService()
	return b
}

func (b *classDefnBase) PopUnresolvedInclusions() []*BLangUserDefinedType {
	inclusions := b.unresolvedInclusions
	b.unresolvedInclusions = nil
	return inclusions
}

func (b *BLangClassDefinition) GetName() *BLangIdentifier {
	return b.Name
}

func (b *BLangClassDefinition) SetName(name *BLangIdentifier) {
	b.Name = name
}

func (b *classDefnBase) GetMethods() iter.Seq2[string, FunctionNode] {
	return func(yield func(string, FunctionNode) bool) {
		for name, method := range b.Methods {
			if !yield(name, method) {
				return
			}
		}
	}
}

func (b *classDefnBase) GetMethod(name string) FunctionNode {
	if method, ok := b.Methods[name]; ok {
		return method
	}
	return nil
}

func (b *classDefnBase) AddMethod(name string, function *BLangFunction) {
	if b.Methods == nil {
		b.Methods = map[string]*BLangFunction{}
	}
	b.Methods[name] = function
}

func (b *classDefnBase) GetInitFunction() FunctionNode {
	if b.InitFunction == nil {
		return nil
	}
	return b.InitFunction
}

func (b *classDefnBase) AddField(field VariableNode) {
	b.Fields = append(b.Fields, field.(*BLangSimpleVariable))
}

func (b *classDefnBase) AddInclusion(symbolRef model.SymbolRef) {
	b.Inclusions = append(b.Inclusions, symbolRef)
}

func (b *classDefnBase) GetAnnotationAttachments() []AnnotationAttachmentNode {
	attachments := make([]AnnotationAttachmentNode, len(b.AnnAttachments))
	for i := range b.AnnAttachments {
		attachments[i] = &b.AnnAttachments[i]
	}
	return attachments
}

func (b *classDefnBase) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	if annAttachment, ok := annAttachment.(*BLangAnnotationAttachment); ok {
		b.AnnAttachments = append(b.AnnAttachments, *annAttachment)
		return
	}
	panic("annAttachment is not a BLangAnnotationAttachment")
}

func (b *classDefnBase) GetMarkdownDocumentationAttachment() MarkdownDocumentationNode {
	if b.MarkdownDocumentationAttachment == nil {
		return nil
	}
	return b.MarkdownDocumentationAttachment
}

func (b *classDefnBase) SetMarkdownDocumentationAttachment(documentationNode MarkdownDocumentationNode) {
	if documentationNode, ok := documentationNode.(*BLangMarkdownDocumentation); ok {
		b.MarkdownDocumentationAttachment = documentationNode
		return
	}
	panic("documentationNode is not a BLangMarkdownDocumentation")
}

func (b *classDefnBase) GetTypeData() TypeData {
	return b.typeData
}

func (b *classDefnBase) SetTypeData(typeData TypeData) {
	b.typeData = typeData
}

func (b *classDefnBase) GetCycleDepth() int {
	return b.CycleDepth
}

func (b *classDefnBase) SetCycleDepth(depth int) {
	b.CycleDepth = depth
}

func (b *BLangCompilationUnit) AddTopLevelNode(node TopLevelNode) {
	b.TopLevelNodes = append(b.TopLevelNodes, node)
}

func (b *BLangCompilationUnit) GetTopLevelNodes() []TopLevelNode {
	return b.TopLevelNodes
}

func (b *BLangCompilationUnit) GetName() string {
	return b.Name
}

func (b *BLangCompilationUnit) SetName(name string) {
	b.Name = name
}

func (b *BLangCompilationUnit) GetPackageID() *model.PackageID {
	return b.packageID
}

func (b *BLangCompilationUnit) SetPackageID(packageID *model.PackageID) {
	b.packageID = packageID
}

func (b *BLangConstant) GetName() *BLangIdentifier {
	return b.Name
}

func (b *BLangConstant) SetName(name *BLangIdentifier) {
	b.Name = name
}

func (b *BLangConstant) GetAnnotationAttachments() []AnnotationAttachmentNode {
	return b.AnnAttachments
}

func (b *BLangConstant) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	if annAttachment, ok := annAttachment.(*BLangAnnotationAttachment); ok {
		b.AnnAttachments = append(b.AnnAttachments, annAttachment)
		return
	}
	panic("annAttachment is not a BLangAnnotationAttachment")
}

func (b *BLangConstant) GetMarkdownDocumentationAttachment() MarkdownDocumentationNode {
	return b.MarkdownDocumentationAttachment
}

func (b *BLangConstant) SetMarkdownDocumentationAttachment(documentationNode MarkdownDocumentationNode) {
	if documentationNode, ok := documentationNode.(*BLangMarkdownDocumentation); ok {
		b.MarkdownDocumentationAttachment = documentationNode
		return
	}
	panic("documentationNode is not a BLangMarkdownDocumentation")
}

func (b *BLangConstant) GetAssociatedType() semtypes.SemType {
	if b.TypeNode() != nil {
		return b.TypeNode().GetTypeData().Type
	}
	return semtypes.SemType{}
}

func (b *BLangSimpleVariable) GetName() *BLangIdentifier {
	return b.Name
}

func (b *BLangSimpleVariable) SetName(name *BLangIdentifier) {
	b.Name = name
}

func (b *BLangMarkdownDocumentation) GetDocumentationLines() []MarkdownDocumentationTextAttributeNode {
	result := make([]MarkdownDocumentationTextAttributeNode, len(b.DocumentationLines))
	for i := range b.DocumentationLines {
		result[i] = &b.DocumentationLines[i]
	}
	return result
}

func (b *BLangMarkdownDocumentation) AddDocumentationLine(documentationText MarkdownDocumentationTextAttributeNode) {
	if line, ok := documentationText.(*BLangMarkdownDocumentationLine); ok {
		b.DocumentationLines = append(b.DocumentationLines, *line)
	} else {
		panic("documentationText is not a BLangMarkdownDocumentationLine")
	}
}

func (b *BLangMarkdownDocumentation) GetParameters() []MarkdownDocumentationParameterAttributeNode {
	result := make([]MarkdownDocumentationParameterAttributeNode, len(b.Parameters))
	for i := range b.Parameters {
		result[i] = &b.Parameters[i]
	}
	return result
}

func (b *BLangMarkdownDocumentation) AddParameter(parameter MarkdownDocumentationParameterAttributeNode) {
	if param, ok := parameter.(*BLangMarkdownParameterDocumentation); ok {
		b.Parameters = append(b.Parameters, *param)
	} else {
		panic("parameter is not a BLangMarkdownParameterDocumentation")
	}
}

func (b *BLangMarkdownDocumentation) GetReturnParameter() MarkdownDocumentationReturnParameterAttributeNode {
	return b.ReturnParameter
}

func (b *BLangMarkdownDocumentation) GetDeprecationDocumentation() MarkDownDocumentationDeprecationAttributeNode {
	return b.DeprecationDocumentation
}

func (b *BLangMarkdownDocumentation) SetReturnParameter(returnParameter MarkdownDocumentationReturnParameterAttributeNode) {
	if param, ok := returnParameter.(*BLangMarkdownReturnParameterDocumentation); ok {
		b.ReturnParameter = param
	} else {
		panic("returnParameter is not a BLangMarkdownReturnParameterDocumentation")
	}
}

func (b *BLangMarkdownDocumentation) SetDeprecationDocumentation(deprecationDocumentation MarkDownDocumentationDeprecationAttributeNode) {
	if doc, ok := deprecationDocumentation.(*BLangMarkDownDeprecationDocumentation); ok {
		b.DeprecationDocumentation = doc
	} else {
		panic("deprecationDocumentation is not a BLangMarkDownDeprecationDocumentation")
	}
}

func (b *BLangMarkdownDocumentation) SetDeprecatedParametersDocumentation(deprecatedParametersDocumentation MarkDownDocumentationDeprecatedParametersAttributeNode) {
	if doc, ok := deprecatedParametersDocumentation.(*BLangMarkDownDeprecatedParametersDocumentation); ok {
		b.DeprecatedParametersDocumentation = doc
	} else {
		panic("deprecatedParametersDocumentation is not a BLangMarkDownDeprecatedParametersDocumentation")
	}
}

func (b *BLangMarkdownDocumentation) GetDeprecatedParametersDocumentation() MarkDownDocumentationDeprecatedParametersAttributeNode {
	return b.DeprecatedParametersDocumentation
}

func (b *BLangMarkdownDocumentation) GetDocumentation() string {
	var lines []string
	for i := range b.DocumentationLines {
		lines = append(lines, b.DocumentationLines[i].GetText())
	}
	result := strings.Join(lines, "\n")
	return strings.ReplaceAll(result, "\r", "")
}

func (b *BLangMarkdownDocumentation) GetParameterDocumentations() map[string]MarkdownDocumentationParameterAttributeNode {
	result := make(map[string]MarkdownDocumentationParameterAttributeNode)
	for i := range b.Parameters {
		paramName := b.Parameters[i].GetParameterName()
		result[paramName.GetValue()] = &b.Parameters[i]
	}
	return result
}

func (b *BLangMarkdownDocumentation) GetReturnParameterDocumentation() *string {
	if b.ReturnParameter == nil {
		return nil
	}
	return new(b.ReturnParameter.GetReturnParameterDocumentation())
}

func (b *BLangMarkdownDocumentation) GetReferences() []MarkdownDocumentationReferenceAttributeNode {
	result := make([]MarkdownDocumentationReferenceAttributeNode, len(b.References))
	for i := range b.References {
		result[i] = &b.References[i]
	}
	return result
}

func (b *BLangMarkdownDocumentation) AddReference(reference MarkdownDocumentationReferenceAttributeNode) {
	if ref, ok := reference.(*BLangMarkdownReferenceDocumentation); ok {
		b.References = append(b.References, *ref)
	} else {
		panic("reference is not a BLangMarkdownReferenceDocumentation")
	}
}

func (b *BLangMarkdownReferenceDocumentation) GetType() DocumentationReferenceType {
	return b.Type
}

func (b *BLangService) GetAttachedExprs() []BLangExpression {
	result := make([]BLangExpression, len(b.AttachedExprs))
	copy(result, b.AttachedExprs)
	return result
}

func (b *BLangService) GetAbsolutePath() []*BLangIdentifier {
	result := make([]*BLangIdentifier, len(b.AbsoluteResourcePath))
	for i := range b.AbsoluteResourcePath {
		result[i] = &b.AbsoluteResourcePath[i]
	}
	return result
}

func (b *BLangService) GetAttachPointLiteral() LiteralNode {
	if b.AttachPointLiteral == nil {
		return nil
	}
	return b.AttachPointLiteral
}

func (b *bLangInvokableNodeBase) Scope() model.Scope {
	return b.scope
}

func (b *bLangInvokableNodeBase) SetScope(scope model.Scope) {
	b.scope = scope
}

var (
	_ NodeWithScope = &BLangFunction{}
	_ NodeWithScope = &BLangResourceMethod{}
)

type ResourcePathSegmentKind uint8

const (
	ResourcePathSegmentName ResourcePathSegmentKind = iota
	ResourcePathSegmentParam
	ResourcePathSegmentParamRest
)

func (b *bLangInvokableNodeBase) GetName() IdentifierNode {
	return &b.Name
}

func (b *bLangInvokableNodeBase) SetName(name IdentifierNode) {
	if id, ok := name.(*BLangIdentifier); ok {
		b.Name = *id
	} else {
		panic("name is not a BLangIdentifier")
	}
}

func (b *bLangInvokableNodeBase) GetAnnotationAttachments() []AnnotationAttachmentNode {
	result := make([]AnnotationAttachmentNode, len(b.AnnAttachments))
	for i := range b.AnnAttachments {
		result[i] = &b.AnnAttachments[i]
	}
	return result
}

func (b *bLangInvokableNodeBase) GetAnnAttachments() []AnnotationAttachmentNode {
	return b.GetAnnotationAttachments()
}

func (b *bLangInvokableNodeBase) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	b.AnnAttachments = append(b.AnnAttachments, *annAttachment.(*BLangAnnotationAttachment))
}

func (b *bLangInvokableNodeBase) SetAnnAttachments(annAttachments []AnnotationAttachmentNode) {
	result := make([]BLangAnnotationAttachment, len(annAttachments))
	for i, attachment := range annAttachments {
		result[i] = *attachment.(*BLangAnnotationAttachment)
	}
	b.AnnAttachments = result
}

func (b *bLangInvokableNodeBase) GetMarkdownDocumentationAttachment() MarkdownDocumentationNode {
	return b.MarkdownDocumentationAttachment
}

func (b *bLangInvokableNodeBase) SetMarkdownDocumentationAttachment(markdownDocumentationAttachment MarkdownDocumentationNode) {
	if doc, ok := markdownDocumentationAttachment.(*BLangMarkdownDocumentation); ok {
		b.MarkdownDocumentationAttachment = doc
	} else {
		panic("markdownDocumentationAttachment is not a BLangMarkdownDocumentation")
	}
}

// BLangReturnTypeDescriptor is the return type descriptor of an invokable node.
// It holds both the return type and its annotation attachments.
type BLangReturnTypeDescriptor struct {
	bLangNodeBase
	TypeDescriptor BType
	AnnAttachments []BLangAnnotationAttachment
}

func (r *BLangReturnTypeDescriptor) IsPublic() bool { return false }

func (r *BLangReturnTypeDescriptor) AddAnnotationAttachment(ann AnnotationAttachmentNode) {
	r.AnnAttachments = append(r.AnnAttachments, *ann.(*BLangAnnotationAttachment))
}

func (r *BLangReturnTypeDescriptor) GetAnnotationAttachments() []AnnotationAttachmentNode {
	result := make([]AnnotationAttachmentNode, len(r.AnnAttachments))
	for i := range r.AnnAttachments {
		result[i] = &r.AnnAttachments[i]
	}
	return result
}

func (r *BLangReturnTypeDescriptor) IsGrouped() bool { return r.innerType().IsGrouped() }

func (r *BLangReturnTypeDescriptor) SetTypeData(ty TypeData) { r.innerType().SetTypeData(ty) }

func (r *BLangReturnTypeDescriptor) GetTypeData() TypeData { return r.innerType().GetTypeData() }

func (r *BLangReturnTypeDescriptor) BTypeGetTag() TypeTags { return r.innerType().BTypeGetTag() }

func (r *BLangReturnTypeDescriptor) BTypeSetTag(tag TypeTags) { r.innerType().BTypeSetTag(tag) }

func (r *BLangReturnTypeDescriptor) bTypeGetName() model.Name { return r.innerType().bTypeGetName() }

func (r *BLangReturnTypeDescriptor) bTypeSetName(name model.Name) { r.innerType().bTypeSetName(name) }

func (r *BLangReturnTypeDescriptor) bTypeGetFlags() model.Flag { return r.innerType().bTypeGetFlags() }

func (r *BLangReturnTypeDescriptor) bTypeSetFlags(flags model.Flag) {
	r.innerType().bTypeSetFlags(flags)
}

func (r *BLangReturnTypeDescriptor) innerType() BType {
	if r.TypeDescriptor == nil {
		panic("BLangReturnTypeDescriptor has nil TypeDescriptor")
	}
	return r.TypeDescriptor
}

func (b *bLangInvokableNodeBase) GetParameters() []SimpleVariableNode {
	result := make([]SimpleVariableNode, len(b.RequiredParams))
	for i := range b.RequiredParams {
		result[i] = &b.RequiredParams[i]
	}
	return result
}

func (b *bLangInvokableNodeBase) AddParameter(param SimpleVariableNode) {
	if blangParam, ok := param.(*BLangSimpleVariable); ok {
		b.RequiredParams = append(b.RequiredParams, *blangParam)
	} else {
		panic("param is not a BLangSimpleVariable")
	}
}

// RequiredParameters returns the concrete required-parameter nodes backing this
// invokable so callers can take the address of an individual parameter.
func (b *bLangInvokableNodeBase) RequiredParameters() []BLangSimpleVariable {
	return b.RequiredParams
}

func (b *bLangInvokableNodeBase) GetRequiredParams() []SimpleVariableNode {
	result := make([]SimpleVariableNode, len(b.RequiredParams))
	for i := range b.RequiredParams {
		result[i] = &b.RequiredParams[i]
	}
	return result
}

func (b *bLangInvokableNodeBase) SetRequiredParams(requiredParams []SimpleVariableNode) {
	b.RequiredParams = make([]BLangSimpleVariable, len(requiredParams))
	for i, param := range requiredParams {
		if blangParam, ok := param.(*BLangSimpleVariable); ok {
			b.RequiredParams[i] = *blangParam
		} else {
			panic("requiredParams contains element that is not a BLangSimpleVariable")
		}
	}
}

func (b *bLangInvokableNodeBase) GetRestParam() SimpleVariableNode {
	return b.RestParam
}

func (b *bLangInvokableNodeBase) SetRestParameter(restParam SimpleVariableNode) {
	b.RestParam = restParam.(*BLangSimpleVariable)
}

func (b *bLangInvokableNodeBase) SetRestParam(restParam SimpleVariableNode) {
	b.SetRestParameter(restParam)
}

func (b *bLangInvokableNodeBase) HasBody() bool {
	return b.Body != nil
}

func (b *bLangInvokableNodeBase) GetReturnTypeDescriptor() ReturnTypeDescriptor {
	if b.returnTypeDescriptor == nil {
		return nil
	}
	return b.returnTypeDescriptor
}

func (b *bLangInvokableNodeBase) SetReturnTypeDescriptor(typeDescriptor TypeDescriptor) {
	if typeDescriptor == nil {
		b.returnTypeDescriptor = nil
		return
	}
	bType := typeDescriptor.(BType)
	if b.returnTypeDescriptor == nil {
		b.returnTypeDescriptor = &BLangReturnTypeDescriptor{}
	}
	b.returnTypeDescriptor.TypeDescriptor = bType
	b.returnTypeDescriptor.SetPosition(bType.GetPosition())
}

// ReturnTypeDescriptorNode returns the return type descriptor node, which carries
// the return type's annotation attachments, or nil if there is none.
func (b *bLangInvokableNodeBase) ReturnTypeDescriptorNode() *BLangReturnTypeDescriptor {
	return b.returnTypeDescriptor
}

func (b *bLangInvokableNodeBase) GetBody() FunctionBodyNode {
	return b.Body
}

func (b *bLangInvokableNodeBase) SetBody(body FunctionBodyNode) {
	b.Body = body
}

func (b *BLangVariableBase) GetAnnAttachments() []AnnotationAttachmentNode {
	return b.AnnAttachments
}

func (b *BLangVariableBase) SetAnnAttachments(annAttachments []AnnotationAttachmentNode) {
	b.AnnAttachments = annAttachments
}

func (b *BLangVariableBase) GetMarkdownDocumentationAttachment() MarkdownDocumentationNode {
	if b.MarkdownDocumentationAttachment == nil {
		return nil
	}
	return b.MarkdownDocumentationAttachment
}

func (b *BLangVariableBase) SetMarkdownDocumentationAttachment(markdownDocumentationAttachment MarkdownDocumentationNode) {
	if markdownDocumentationAttachment == nil {
		b.MarkdownDocumentationAttachment = nil
		return
	}
	b.MarkdownDocumentationAttachment = markdownDocumentationAttachment.(*BLangMarkdownDocumentation)
}

func (b *BLangVariableBase) GetExpr() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangVariableBase) SetExpr(expr BLangActionOrExpression) {
	b.Expr = expr
}

func (b *BLangVariableBase) GetIsDeclaredWithVar() bool {
	return b.IsDeclaredWithVar
}

func (b *BLangVariableBase) SetIsDeclaredWithVar(isDeclaredWithVar bool) {
	b.IsDeclaredWithVar = isDeclaredWithVar
}

func (m *BLangVariableBase) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	m.AnnAttachments = append(m.AnnAttachments, annAttachment)
}

func (m *BLangVariableBase) GetAnnotationAttachments() []AnnotationAttachmentNode {
	return m.AnnAttachments
}

func (m *BLangVariableBase) GetInitialExpression() BLangActionOrExpression {
	return m.Expr
}

func (m *BLangVariableBase) SetInitialExpression(expr BLangActionOrExpression) {
	m.Expr = expr
}

// BLangTypeDefinition methods

func NewBLangTypeDefinition() *BLangTypeDefinition {
	b := &BLangTypeDefinition{}
	b.annAttachments = []BLangAnnotationAttachment{}
	b.CycleDepth = -1
	b.hasCyclicReference = false
	return b
}

func (b *BLangTypeDefinition) GetName() *BLangIdentifier {
	return b.Name
}

func (b *BLangTypeDefinition) SetName(name *BLangIdentifier) {
	b.Name = name
}

func (b *BLangTypeDefinition) GetTypeData() TypeData {
	return b.typeData
}

func (b *BLangTypeDefinition) SetTypeData(typeData TypeData) {
	b.typeData = typeData
}

func (b *BLangTypeDefinition) GetAnnotationAttachments() []AnnotationAttachmentNode {
	result := make([]AnnotationAttachmentNode, len(b.annAttachments))
	for i := range b.annAttachments {
		result[i] = &b.annAttachments[i]
	}
	return result
}

func (b *BLangTypeDefinition) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	if ann, ok := annAttachment.(*BLangAnnotationAttachment); ok {
		b.annAttachments = append(b.annAttachments, *ann)
	} else {
		panic("annAttachment is not a BLangAnnotationAttachment")
	}
}

func (b *BLangTypeDefinition) GetMarkdownDocumentationAttachment() MarkdownDocumentationNode {
	return b.markdownDocumentationAttachment
}

func (b *BLangTypeDefinition) SetMarkdownDocumentationAttachment(documentationNode MarkdownDocumentationNode) {
	if doc, ok := documentationNode.(*BLangMarkdownDocumentation); ok {
		b.markdownDocumentationAttachment = doc
	} else {
		panic("documentationNode is not a BLangMarkdownDocumentation")
	}
}

func (b *BLangTypeDefinition) GetCycleDepth() int {
	return b.CycleDepth
}

func (b *BLangTypeDefinition) SetCycleDepth(depth int) {
	b.CycleDepth = depth
}

func (b *BLangXMLNS) GetNamespaceURI() BLangExpression {
	return b.namespaceURI
}

func (b *BLangXMLNS) GetPrefix() *BLangIdentifier {
	return b.prefix
}

func (b *BLangXMLNS) SetNamespaceURI(namespaceURI BLangExpression) {
	b.namespaceURI = namespaceURI
}

func (b *BLangXMLNS) SetPrefix(prefix *BLangIdentifier) {
	b.prefix = prefix
}

func (b *BLangPackage) GetImports() []ImportPackageNode {
	result := make([]ImportPackageNode, len(b.Imports))
	for i := range b.Imports {
		result[i] = &b.Imports[i]
	}
	return result
}

func (b *BLangPackage) AddImport(importPkg ImportPackageNode) {
	if imp, ok := importPkg.(*BLangImportPackage); ok {
		b.Imports = append(b.Imports, *imp)
	} else {
		panic("importPkg is not a BLangImportPackage")
	}
}

func (b *BLangPackage) GetNamespaceDeclarations() []XMLNSDeclarationNode {
	result := make([]XMLNSDeclarationNode, len(b.XmlnsList))
	for i := range b.XmlnsList {
		result[i] = &b.XmlnsList[i]
	}
	return result
}

func (b *BLangPackage) AddNamespaceDeclaration(xmlnsDecl XMLNSDeclarationNode) {
	if xmlns, ok := xmlnsDecl.(*BLangXMLNS); ok {
		b.XmlnsList = append(b.XmlnsList, *xmlns)
	} else {
		panic("xmlnsDecl is not a BLangXMLNS")
	}
}

func (b *BLangPackage) GetConstants() []ConstantNode {
	result := make([]ConstantNode, len(b.Constants))
	for i := range b.Constants {
		result[i] = &b.Constants[i]
	}
	return result
}

func (b *BLangPackage) GetGlobalVariables() []VariableNode {
	result := make([]VariableNode, len(b.GlobalVars))
	for i := range b.GlobalVars {
		result[i] = &b.GlobalVars[i]
	}
	return result
}

func (b *BLangPackage) AddGlobalVariable(globalVar SimpleVariableNode) {
	if sv, ok := globalVar.(*BLangSimpleVariable); ok {
		b.GlobalVars = append(b.GlobalVars, *sv)
	} else {
		panic("globalVar is not a BLangSimpleVariable")
	}
}

func (b *BLangPackage) GetServices() []ServiceNode {
	result := make([]ServiceNode, len(b.Services))
	for i := range b.Services {
		result[i] = &b.Services[i]
	}
	return result
}

func (b *BLangPackage) AddService(service ServiceNode) {
	if svc, ok := service.(*BLangService); ok {
		b.Services = append(b.Services, *svc)
	} else {
		panic("service is not a BLangService")
	}
}

func (b *BLangPackage) GetFunctions() []FunctionNode {
	result := make([]FunctionNode, len(b.Functions))
	for i := range b.Functions {
		result[i] = &b.Functions[i]
	}
	return result
}

func (b *BLangPackage) AddFunction(function FunctionNode) {
	if fn, ok := function.(*BLangFunction); ok {
		b.Functions = append(b.Functions, *fn)
	} else {
		panic("function is not a BLangFunction")
	}
}

func (b *BLangPackage) GetTypeDefinitions() []*BLangTypeDefinition {
	result := make([]*BLangTypeDefinition, len(b.TypeDefinitions))
	for i := range b.TypeDefinitions {
		result[i] = &b.TypeDefinitions[i]
	}
	return result
}

func (b *BLangPackage) AddTypeDefinition(typeDefinition *BLangTypeDefinition) {
	b.TypeDefinitions = append(b.TypeDefinitions, *typeDefinition)
}

func (b *BLangPackage) GetAnnotations() []AnnotationNode {
	result := make([]AnnotationNode, len(b.Annotations))
	for i := range b.Annotations {
		result[i] = &b.Annotations[i]
	}
	return result
}

func (b *BLangPackage) AddAnnotation(annotation AnnotationNode) {
	if ann, ok := annotation.(*BLangAnnotation); ok {
		b.Annotations = append(b.Annotations, *ann)
	} else {
		panic("annotation is not a BLangAnnotation")
	}
}

func (b *BLangPackage) GetClassDefinitions() []ClassDefinition {
	result := make([]ClassDefinition, len(b.ClassDefinitions))
	for i := range b.ClassDefinitions {
		result[i] = &b.ClassDefinitions[i]
	}
	return result
}

func (b *BLangPackage) AddTestablePkg(testablePkg *BLangTestablePackage) {
	b.TestablePkgs = append(b.TestablePkgs, testablePkg)
}

func (b *BLangPackage) GetTestablePkgs() []*BLangTestablePackage {
	return b.TestablePkgs
}

func (b *BLangPackage) GetTestablePkg() *BLangTestablePackage {
	if len(b.TestablePkgs) > 0 {
		return b.TestablePkgs[0]
	}
	return nil
}

func (b *BLangPackage) ContainsTestablePkg() bool {
	return len(b.TestablePkgs) > 0
}

func (b *BLangPackage) HasTestablePackage() bool {
	return len(b.TestablePkgs) > 0
}

func (b *BLangPackage) AddClassDefinition(classDefNode *BLangClassDefinition) {
	b.ClassDefinitions = append(b.ClassDefinitions, *classDefNode)
}

func NewBLangPackage(env semtypes.Env) *BLangPackage {
	b := &BLangPackage{}
	b.Imports = []BLangImportPackage{}
	b.XmlnsList = []BLangXMLNS{}
	b.Constants = []BLangConstant{}
	b.GlobalVars = []BLangSimpleVariable{}
	b.Services = []BLangService{}
	b.Functions = []BLangFunction{}
	b.TypeDefinitions = []BLangTypeDefinition{}
	b.Annotations = []BLangAnnotation{}
	b.TestablePkgs = []*BLangTestablePackage{}
	b.ClassDefinitions = []BLangClassDefinition{}
	return b
}

func (b *BLangTestablePackage) GetMockFunctionNamesMap() map[string]string {
	return b.mockFunctionNamesMap
}

func (b *BLangTestablePackage) AddMockFunction(id string, function string) {
	if b.mockFunctionNamesMap == nil {
		b.mockFunctionNamesMap = make(map[string]string)
	}
	b.mockFunctionNamesMap[id] = function
}

func (b *BLangTestablePackage) GetIsLegacyMockingMap() map[string]bool {
	return b.isLegacyMockingMap
}

func (b *BLangTestablePackage) AddIsLegacyMockingMap(id string, isLegacy bool) {
	if b.isLegacyMockingMap == nil {
		b.isLegacyMockingMap = make(map[string]bool)
	}
	b.isLegacyMockingMap[id] = isLegacy
}

func createSimpleVariableNodeWithLocationTokenLocation(location diagnostics.Location, identifier tree.Token, identifierPos diagnostics.Location) *BLangSimpleVariable {
	memberVar := createSimpleVariableNode()
	memberVar.pos = location
	name := createIdentifierFromToken(identifierPos, identifier)
	BLangNode(&name).SetPosition(identifierPos)
	memberVar.SetName(&name)
	return memberVar
}

func createSimpleVariableNode() *BLangSimpleVariable {
	return &BLangSimpleVariable{}
}

func createConstantNode() *BLangConstant {
	c := &BLangConstant{}
	c.flags = model.FlagConstant
	return c
}

func GetCompilationUnit(cx *context.CompilerContext, syntaxTree *tree.SyntaxTree) *BLangCompilationUnit {
	nodeBuilder := NewNodeBuilder(cx)
	compilationUnit := nodeBuilder.TransformModulePart(syntaxTree.RootNode.(*tree.ModulePart))
	return compilationUnit.(*BLangCompilationUnit)
}

func ToPackageFromCompilationUnits(compilationUnits []*BLangCompilationUnit) *BLangPackage {
	p := BLangPackage{}
	for _, compilationUnit := range compilationUnits {
		if p.PackageID == nil {
			p.PackageID = compilationUnit.packageID
		} else if compilationUnit.packageID != nil && p.PackageID != compilationUnit.packageID {
			panic("compilation units have different package IDs")
		}
		addCompilationUnitNodesToPackage(&p, compilationUnit)
	}
	return &p
}

func addCompilationUnitNodesToPackage(p *BLangPackage, compilationUnit *BLangCompilationUnit) {
	for _, node := range compilationUnit.TopLevelNodes {
		switch node := node.(type) {
		case *BLangImportPackage:
			p.Imports = append(p.Imports, *node)
		case *BLangConstant:
			p.Constants = append(p.Constants, *node)
		case *BLangService:
			p.Services = append(p.Services, *node)
		case *BLangSimpleVariable:
			p.GlobalVars = append(p.GlobalVars, *node)
		case *BLangFunction:
			if node.Name.Value == "init" {
				p.InitFunction = node
			} else {
				p.Functions = append(p.Functions, *node)
			}
		case *BLangTypeDefinition:
			p.TypeDefinitions = append(p.TypeDefinitions, *node)
		case *BLangAnnotation:
			p.Annotations = append(p.Annotations, *node)
		case *BLangXMLNS:
			p.XmlnsList = append(p.XmlnsList, *node)
		case *BLangClassDefinition:
			p.ClassDefinitions = append(p.ClassDefinitions, *node)
		default:
			panic(fmt.Sprintf("unexpected top-level node type: %T", node))
		}
	}
}
