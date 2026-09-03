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

// Package ast declares types used to represent Abstract Syntax Tree for Ballerina source
package ast

import (
	"iter"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/values"
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

type BLangNode interface {
	Node
	SetDeterminedType(ty semtypes.SemType)
	SetPosition(pos diagnostics.Location)
}

type (
	bLangNodeBase struct {
		DeterminedType semtypes.SemType
		pos            diagnostics.Location
	}

	BLangAnnotation struct {
		bLangNodeBase
		Name                            IdentifierNode
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
		AnnotationName  IdentifierNode
		PkgAlias        IdentifierNode
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
	}

	BLangImportPackage struct {
		bLangNodeBase
		OrgName      *BLangIdentifier
		PkgNameComps []BLangIdentifier
		Alias        *BLangIdentifier
		Version      *BLangIdentifier
	}

	unresolvedInclusionsBase struct {
		unresolvedInclusions []*BLangUserDefinedType
	}

	classDefnBase struct {
		bLangNodeBase
		unresolvedInclusionsBase
		scope                           model.Scope
		symbol                          model.SymbolRef
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
		InitFunction                    *BLangFunction
		Methods                         map[string]*BLangFunction
		ResourceMethods                 []*BLangResourceMethod
		Fields                          []*BLangVariable
		Inclusions                      []model.SymbolRef      // This needs to be symbol because it could be a class definition as well
		InclusionPositions              []diagnostics.Location // Positions of each inclusion, parallel to Inclusions
		flags                           model.Flag
		typeData                        TypeData
		Definition                      semtypes.Definition
		CycleDepth                      int
	}

	BLangClassDefinition struct {
		classDefnBase
		Name IdentifierNode
	}

	BLangService struct {
		classDefnBase
		AttachedExprs         []BLangExpression
		AttachedExprsPosition diagnostics.Location
		// A nil AbsoluteResourcePath means there is no attach point; an empty,
		// non-nil path represents the root attach point `/`.
		AbsoluteResourcePath []BLangIdentifier
		AttachPointLiteral   *BLangLiteral
		AttachPointType      semtypes.SemType
		ObjectBodyType       semtypes.SemType
	}

	BLangCompilationUnit struct {
		bLangNodeBase
		TopLevelNodes []TopLevelNode
		Name          string
		Scope         model.Scope
		packageID     *model.PackageID
	}

	BLangPackage struct {
		bLangNodeBase
		Imports          []*BLangImportPackage
		XmlnsList        []*BLangXMLNS
		Constants        []*BLangVariable
		GlobalVars       []*BLangVariable
		Services         []*BLangService
		Functions        []*BLangFunction
		TypeDefinitions  []*BLangTypeDefinition
		Annotations      []*BLangAnnotation
		InitFunction     *BLangFunction
		ClassDefinitions []*BLangClassDefinition
		PackageID        *model.PackageID
		Scope            model.Scope
	}
	BLangXMLNS struct {
		bLangNodeBase
		symbol       model.SymbolRef
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

	bLangVariableBase struct {
		bLangNodeBase
		// We are using variable for function paramets and record td fields so we need to have
		// type descriptors here. Not sure this is the best way to do this.
		typeNode                        BType
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
		Expr                            BLangActionOrExpression
		flags                           model.Flag
		IsDeclaredWithVar               bool
		symbol                          model.SymbolRef
	}

	BLangVariable struct {
		bLangVariableBase
		Name IdentifierNode
	}

	ClosureVarSymbol struct {
		DiagnosticLocation diagnostics.Location
	}

	bLangInvokableNodeBase struct {
		bLangNodeBase
		Name                            IdentifierNode
		symbol                          model.SymbolRef
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
		RequiredParams                  []BLangVariable
		RestParam                       *BLangVariable
		returnTypeDescriptor            *BLangReturnTypeDescriptor
		ParamListPos                    Location // range from ( to ) inclusive
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
		Name                            IdentifierNode
		symbol                          model.SymbolRef
		typeData                        TypeData
		annAttachments                  []BLangAnnotationAttachment
		markdownDocumentationAttachment *BLangMarkdownDocumentation
		flags                           model.Flag
		CycleDepth                      int
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

func (b *bLangInvokableNodeBase) SetAttached()      { b.flags |= model.FlagAttached }
func (b *bLangInvokableNodeBase) Flags() model.Flag { return b.flags }

func (b *bLangInvokableNodeBase) FuncSymbolFlags() model.FuncSymbolFlags {
	return model.FuncSymbolFlags(b.flags)
}

func (b *bLangInvokableNodeBase) Parameters() []Param {
	parameters := b.GetParameters()
	params := make([]Param, len(parameters))
	for i := range parameters {
		params[i] = &parameters[i]
	}
	return params
}

func (b *bLangInvokableNodeBase) RestParameter() Param {
	if b.RestParam == nil {
		return nil
	}
	return b.RestParam
}

func (b *bLangInvokableNodeBase) ReturnType() TypeDescriptor {
	if b.returnTypeDescriptor == nil {
		return nil
	}
	return b.returnTypeDescriptor
}

// bLangVariableBase flag methods
func (b *bLangVariableBase) IsPublic() bool           { return b.flags.Has(model.FlagPublic) }
func (b *bLangVariableBase) IsFinal() bool            { return b.flags.Has(model.FlagFinal) }
func (b *bLangVariableBase) IsConfigurable() bool     { return b.flags.Has(model.FlagConfigurable) }
func (b *bLangVariableBase) IsDefaultableParam() bool { return b.flags.Has(model.FlagDefaultableParam) }
func (b *bLangVariableBase) IsRequiredParam() bool    { return b.flags.Has(model.FlagRequiredParam) }
func (b *bLangVariableBase) IsRestParam() bool        { return b.flags.Has(model.FlagRestParam) }
func (b *bLangVariableBase) IsIncludedRecordParam() bool {
	return b.flags.Has(model.FlagIncluded)
}

func (b *bLangVariableBase) SetRequiredParam() { b.flags |= model.FlagRequiredParam }
func (b *bLangVariableBase) IsReadonly() bool  { return b.flags.Has(model.FlagReadonly) }
func (b *bLangVariableBase) IsListener() bool  { return b.flags.Has(model.FlagListener) }
func (b *bLangVariableBase) IsConstant() bool  { return b.flags.Has(model.FlagConstant) }
func (b *bLangVariableBase) Flags() model.Flag { return b.flags }

// classDefnBase flag methods (promoted to BLangClassDefinition / BLangService)
func (b *classDefnBase) IsPublic() bool   { return b.flags.Has(model.FlagPublic) }
func (b *classDefnBase) IsDistinct() bool { return b.flags.Has(model.FlagDistinct) }
func (b *classDefnBase) IsClient() bool   { return b.flags.Has(model.FlagClient) }
func (b *classDefnBase) IsReadonly() bool { return b.flags.Has(model.FlagReadonly) }
func (b *classDefnBase) IsService() bool  { return b.flags.Has(model.FlagService) }
func (b *classDefnBase) IsIsolated() bool { return b.flags.Has(model.FlagIsolated) }

func (b *classDefnBase) Flags() model.Flag { return b.flags }

// BLangTypeDefinition flag methods
func (b *BLangTypeDefinition) IsPublic() bool    { return b.flags.Has(model.FlagPublic) }
func (b *BLangTypeDefinition) IsAnonymous() bool { return b.flags.Has(model.FlagAnonymous) }
func (b *BLangTypeDefinition) IsDistinct() bool  { return b.flags.Has(model.FlagDistinct) }
func (b *BLangAnnotation) IsPublic() bool        { return b.flags.Has(model.FlagPublic) }
func (b *BLangAnnotation) IsConst() bool         { return b.flags.Has(model.FlagConstant) }

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

func (n *BLangXMLNS) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangXMLNS) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *classDefnBase) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *classDefnBase) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *classDefnBase) Scope() model.Scope {
	return n.scope
}

func (n *classDefnBase) SetScope(scope model.Scope) {
	n.scope = scope
}

func (n *bLangVariableBase) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *bLangVariableBase) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *bLangVariableBase) TypeNode() BType {
	return n.typeNode
}

func (n *bLangVariableBase) SetTypeNode(bt BType) {
	n.typeNode = bt
}

func (n *bLangVariableBase) Type() BType {
	return n.typeNode
}

func (n *bLangVariableBase) DefaultExpr() BLangExpression {
	if n.Expr == nil {
		return nil
	}
	return n.Expr.(BLangExpression)
}

func (n *bLangVariableBase) IsDefaultable() bool {
	return n.IsDefaultableParam()
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
	_ BNodeWithSymbol   = &BLangAnnotation{}
	_ BNodeWithSymbol   = &BLangAnnotationAttachment{}
	_ BNodeWithSymbol   = &BLangXMLNS{}
	_ NodeWithScope     = &BLangClassDefinition{}
	_ FunctionBodyNode  = &BLangExternFunctionBody{}
	_ FunctionSignature = &BLangFunction{}
	_ FunctionSignature = &BLangResourceMethod{}
	_ Param             = &BLangVariable{}
)

func (*BLangImportPackage) isTopLevel()   {}
func (*BLangXMLNS) isTopLevel()           {}
func (*BLangAnnotation) isTopLevel()      {}
func (*BLangVariable) isTopLevel()        {}
func (*BLangFunction) isTopLevel()        {}
func (*BLangClassDefinition) isTopLevel() {}
func (*BLangService) isTopLevel()         {}
func (*BLangTypeDefinition) isTopLevel()  {}

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
	_ BLangNode = &BLangXMLNS{}
	_ BLangNode = &BLangMarkdownDocumentation{}
	_ BLangNode = &BLangMarkdownReferenceDocumentation{}
	_ BLangNode = &BLangVariable{}
	_ BLangNode = &BLangFunction{}
	_ BLangNode = &BLangTypeDefinition{}
)

var (
	// Assert that concrete types with symbols implement BNodeWithSymbol
	_ BNodeWithSymbol = &BLangClassDefinition{}
	_ BNodeWithSymbol = &BLangService{}
	_ BNodeWithSymbol = &BLangVariable{}
	_ BNodeWithSymbol = &BLangFunction{}
	_ BNodeWithSymbol = &BLangTypeDefinition{}
)

func NewBLangAnnotation(pos Location, name IdentifierNode, typeDescriptor TypeDescriptor, flags model.Flag) *BLangAnnotation {
	return &BLangAnnotation{
		bLangNodeBase:  bLangNodeBase{pos: pos},
		Name:           name,
		typeDescriptor: typeDescriptor,
		flags:          flags,
	}
}

func (b *BLangAnnotationAttachment) GetPackageAlias() IdentifierNode {
	return b.PkgAlias
}

func (b *BLangAnnotationAttachment) GetAnnotationName() IdentifierNode {
	return b.AnnotationName
}

func (b *BLangAnnotationAttachment) GetExpressionNode() BLangExpression {
	return b.Expr
}

func (b *BLangAnnotation) GetName() IdentifierNode {
	return b.Name
}

func (b *BLangAnnotation) GetTypeDescriptor() TypeDescriptor {
	if b.typeDescriptor == nil {
		return nil
	}
	return b.typeDescriptor
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

func (b *BLangAnnotation) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.AnnAttachments
}

func (b *BLangAnnotation) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.AnnAttachments = append(b.AnnAttachments, annAttachment)
}

func (b *BLangAnnotation) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.MarkdownDocumentationAttachment
}

func (b *BLangExprFunctionBody) GetExpr() BLangExpression {
	return b.Expr
}

func NewBLangIdentifier(pos Location, value, originalValue string) BLangIdentifier {
	return BLangIdentifier{
		bLangNodeBase: bLangNodeBase{pos: pos},
		Value:         value,
		OriginalValue: originalValue,
	}
}

func NewBLangVariable(pos Location, name IdentifierNode, typeNode BType, expr BLangActionOrExpression, isDeclaredWithVar bool, flags model.Flag) *BLangVariable {
	return &BLangVariable{
		bLangVariableBase: bLangVariableBase{
			bLangNodeBase:     bLangNodeBase{pos: pos},
			typeNode:          typeNode,
			Expr:              expr,
			flags:             flags,
			IsDeclaredWithVar: isDeclaredWithVar,
		},
		Name: name,
	}
}

func (b *BLangIdentifier) GetValue() string {
	return b.Value
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

func (b *BLangImportPackage) GetPackageVersion() *BLangIdentifier {
	return b.Version
}

func (b *BLangImportPackage) GetAlias() *BLangIdentifier {
	return b.Alias
}

func newClassDefnBase() classDefnBase {
	b := classDefnBase{}
	b.CycleDepth = -1
	b.Methods = map[string]*BLangFunction{}
	return b
}

func NewBLangClassDefinition() BLangClassDefinition {
	return NewBLangClassDefinitionWithFlags(0)
}

func NewBLangClassDefinitionWithFlags(flags model.Flag) BLangClassDefinition {
	b := BLangClassDefinition{classDefnBase: newClassDefnBase()}
	b.flags = flags | model.FlagClass
	return b
}

func NewBLangService() BLangService {
	return NewBLangServiceWithFlags(0)
}

func NewBLangServiceWithFlags(flags model.Flag) BLangService {
	b := BLangService{classDefnBase: newClassDefnBase()}
	b.flags = flags | model.FlagService
	return b
}

func (b *unresolvedInclusionsBase) AddUnresolvedInclusion(inclusion *BLangUserDefinedType) {
	b.unresolvedInclusions = append(b.unresolvedInclusions, inclusion)
}

func (b *unresolvedInclusionsBase) PopUnresolvedInclusions() []*BLangUserDefinedType {
	inclusions := b.unresolvedInclusions
	b.unresolvedInclusions = nil
	return inclusions
}

func (b *BLangClassDefinition) GetName() IdentifierNode {
	return b.Name
}

func (b *classDefnBase) GetMethods() iter.Seq2[string, *BLangFunction] {
	return func(yield func(string, *BLangFunction) bool) {
		for name, method := range b.Methods {
			if !yield(name, method) {
				return
			}
		}
	}
}

func (b *classDefnBase) GetMethod(name string) *BLangFunction {
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

func (b *classDefnBase) GetInitFunction() *BLangFunction {
	if b.InitFunction == nil {
		return nil
	}
	return b.InitFunction
}

func (b *classDefnBase) AddField(field *BLangVariable) {
	b.Fields = append(b.Fields, field)
}

func (b *classDefnBase) AddInclusion(symbolRef model.SymbolRef) {
	b.Inclusions = append(b.Inclusions, symbolRef)
}

func (b *classDefnBase) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.AnnAttachments
}

func (b *classDefnBase) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.AnnAttachments = append(b.AnnAttachments, annAttachment)
}

func (b *classDefnBase) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.MarkdownDocumentationAttachment
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

func (b *BLangCompilationUnit) GetPackageID() *model.PackageID {
	return b.packageID
}

func (b *BLangCompilationUnit) SetPackageID(packageID *model.PackageID) {
	b.packageID = packageID
}

func (b *BLangVariable) GetName() IdentifierNode {
	return b.Name
}

func (b *BLangVariable) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.GetAnnAttachments()
}

func (b *BLangVariable) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.AnnAttachments = append(b.AnnAttachments, annAttachment)
}

func (b *BLangVariable) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.MarkdownDocumentationAttachment
}

func (b *BLangVariable) GetAssociatedType() semtypes.SemType {
	if b.TypeNode() != nil {
		return b.TypeNode().GetTypeData().Type
	}
	return semtypes.SemType{}
}

func (b *BLangVariable) ParamName() string {
	if b.Name == nil {
		return ""
	}
	return b.Name.GetValue()
}

func (b *BLangVariable) SetName(name IdentifierNode) {
	b.Name = name
}

func (b *BLangMarkdownDocumentation) GetDocumentationLines() []BLangMarkdownDocumentationLine {
	return b.DocumentationLines
}

func (b *BLangMarkdownDocumentation) AddDocumentationLine(documentationText BLangMarkdownDocumentationLine) {
	b.DocumentationLines = append(b.DocumentationLines, documentationText)
}

func (b *BLangMarkdownDocumentation) GetParameters() []BLangMarkdownParameterDocumentation {
	return b.Parameters
}

func (b *BLangMarkdownDocumentation) AddParameter(parameter BLangMarkdownParameterDocumentation) {
	b.Parameters = append(b.Parameters, parameter)
}

func (b *BLangMarkdownDocumentation) GetReturnParameter() *BLangMarkdownReturnParameterDocumentation {
	return b.ReturnParameter
}

func (b *BLangMarkdownDocumentation) GetDeprecationDocumentation() *BLangMarkDownDeprecationDocumentation {
	return b.DeprecationDocumentation
}

func (b *BLangMarkdownDocumentation) GetDeprecatedParametersDocumentation() *BLangMarkDownDeprecatedParametersDocumentation {
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

func (b *BLangMarkdownDocumentation) GetParameterDocumentations() map[string]*BLangMarkdownParameterDocumentation {
	result := make(map[string]*BLangMarkdownParameterDocumentation)
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

func (b *BLangMarkdownDocumentation) GetReferences() []BLangMarkdownReferenceDocumentation {
	return b.References
}

func (b *BLangMarkdownDocumentation) AddReference(reference BLangMarkdownReferenceDocumentation) {
	b.References = append(b.References, reference)
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

type InvokableData struct {
	Position                    Location
	Name                        IdentifierNode
	RequiredParams              []BLangVariable
	RestParam                   *BLangVariable
	ReturnTypeDescriptor        BType
	ReturnAnnotationAttachments []BLangAnnotationAttachment
	ParamListPosition           Location
	Body                        FunctionBodyNode
	Flags                       model.Flag
}

func newBLangInvokableNodeBase(data InvokableData) bLangInvokableNodeBase {
	restParam := data.RestParam
	var returnTypeDescriptor *BLangReturnTypeDescriptor
	if data.ReturnTypeDescriptor != nil {
		returnTypeDescriptor = &BLangReturnTypeDescriptor{
			bLangNodeBase:  bLangNodeBase{pos: data.ReturnTypeDescriptor.GetPosition()},
			TypeDescriptor: data.ReturnTypeDescriptor,
			AnnAttachments: data.ReturnAnnotationAttachments,
		}
	}
	return bLangInvokableNodeBase{
		bLangNodeBase:        bLangNodeBase{pos: data.Position},
		Name:                 data.Name,
		RequiredParams:       data.RequiredParams,
		RestParam:            restParam,
		returnTypeDescriptor: returnTypeDescriptor,
		ParamListPos:         data.ParamListPosition,
		Body:                 data.Body,
		flags:                data.Flags,
	}
}

func NewBLangFunction(data InvokableData) *BLangFunction {
	return &BLangFunction{bLangInvokableNodeBase: newBLangInvokableNodeBase(data)}
}

func NewBLangResourceMethod(data InvokableData, resourcePath []BLangResourcePathSegment) *BLangResourceMethod {
	return &BLangResourceMethod{
		bLangInvokableNodeBase: newBLangInvokableNodeBase(data),
		ResourcePath:           resourcePath,
	}
}

type ResourcePathSegmentKind uint8

const (
	ResourcePathSegmentName ResourcePathSegmentKind = iota
	ResourcePathSegmentParam
	ResourcePathSegmentParamRest
)

func (b *bLangInvokableNodeBase) GetName() IdentifierNode {
	return b.Name
}

func (b *bLangInvokableNodeBase) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.AnnAttachments
}

func (b *bLangInvokableNodeBase) GetAnnAttachments() []BLangAnnotationAttachment {
	return b.GetAnnotationAttachments()
}

func (b *bLangInvokableNodeBase) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.AnnAttachments = append(b.AnnAttachments, annAttachment)
}

func (b *bLangInvokableNodeBase) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.MarkdownDocumentationAttachment
}

// BLangReturnTypeDescriptor is the return type descriptor of an invokable node.
// It holds both the return type and its annotation attachments.
type BLangReturnTypeDescriptor struct {
	bLangNodeBase
	TypeDescriptor BType
	AnnAttachments []BLangAnnotationAttachment
}

func (r *BLangReturnTypeDescriptor) IsPublic() bool { return false }

func (r *BLangReturnTypeDescriptor) AddAnnotationAttachment(ann BLangAnnotationAttachment) {
	r.AnnAttachments = append(r.AnnAttachments, ann)
}

func (r *BLangReturnTypeDescriptor) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return r.AnnAttachments
}

func (r *BLangReturnTypeDescriptor) IsGrouped() bool { return r.innerType().IsGrouped() }

func (r *BLangReturnTypeDescriptor) SetTypeData(ty TypeData) { r.innerType().SetTypeData(ty) }

func (r *BLangReturnTypeDescriptor) GetTypeData() TypeData { return r.innerType().GetTypeData() }

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

func (b *bLangInvokableNodeBase) GetParameters() []BLangVariable {
	return b.RequiredParams
}

func (b *bLangInvokableNodeBase) RequiredParameters() []BLangVariable {
	return b.RequiredParams
}

func (b *bLangInvokableNodeBase) GetRestParam() *BLangVariable {
	return b.RestParam
}

func (b *bLangInvokableNodeBase) HasBody() bool {
	return b.Body != nil
}

func (b *bLangInvokableNodeBase) GetReturnTypeDescriptor() *BLangReturnTypeDescriptor {
	if b.returnTypeDescriptor == nil {
		return nil
	}
	return b.returnTypeDescriptor
}

// ReturnTypeDescriptorNode returns the return type descriptor node, which carries
// the return type's annotation attachments, or nil if there is none.
func (b *bLangInvokableNodeBase) ReturnTypeDescriptorNode() *BLangReturnTypeDescriptor {
	return b.returnTypeDescriptor
}

func (b *bLangInvokableNodeBase) HasExplicitReturnTypeDescriptor() bool {
	return b.flags.Has(model.FlagExplicitReturnTypeDescriptor)
}

func (b *bLangInvokableNodeBase) GetBody() FunctionBodyNode {
	return b.Body
}

func (b *bLangVariableBase) GetAnnAttachments() []BLangAnnotationAttachment {
	return b.AnnAttachments
}

func (b *bLangVariableBase) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.MarkdownDocumentationAttachment
}

func (b *bLangVariableBase) GetExpr() BLangActionOrExpression {
	return b.Expr
}

func (b *bLangVariableBase) GetIsDeclaredWithVar() bool {
	return b.IsDeclaredWithVar
}

func (m *bLangVariableBase) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	m.AnnAttachments = append(m.AnnAttachments, annAttachment)
}

func (m *bLangVariableBase) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return m.GetAnnAttachments()
}

func (m *bLangVariableBase) GetInitialExpression() BLangActionOrExpression {
	return m.Expr
}

func (m *bLangVariableBase) SetInitialExpression(expr BLangActionOrExpression) {
	m.Expr = expr
}

// BLangTypeDefinition methods

func NewBLangTypeDefinition() *BLangTypeDefinition {
	return NewBLangTypeDefinitionWithData(Location{}, nil, TypeData{}, nil, 0)
}

func NewBLangTypeDefinitionWithData(pos Location, name IdentifierNode, typeData TypeData, documentation *BLangMarkdownDocumentation, flags model.Flag) *BLangTypeDefinition {
	return &BLangTypeDefinition{
		bLangNodeBase:                   bLangNodeBase{pos: pos},
		Name:                            name,
		typeData:                        typeData,
		annAttachments:                  []BLangAnnotationAttachment{},
		markdownDocumentationAttachment: documentation,
		flags:                           flags,
		CycleDepth:                      -1,
	}
}

func (b *BLangTypeDefinition) GetName() IdentifierNode {
	return b.Name
}

func (b *BLangTypeDefinition) GetTypeData() TypeData {
	return b.typeData
}

func (b *BLangTypeDefinition) SetTypeData(typeData TypeData) {
	b.typeData = typeData
}

func (b *BLangTypeDefinition) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.annAttachments
}

func (b *BLangTypeDefinition) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.annAttachments = append(b.annAttachments, annAttachment)
}

func (b *BLangTypeDefinition) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.markdownDocumentationAttachment
}

func (b *BLangTypeDefinition) GetCycleDepth() int {
	return b.CycleDepth
}

func (b *BLangTypeDefinition) SetCycleDepth(depth int) {
	b.CycleDepth = depth
}

func NewBLangXMLNS(pos Location, namespaceURI BLangExpression, prefix *BLangIdentifier) *BLangXMLNS {
	return &BLangXMLNS{bLangNodeBase: bLangNodeBase{pos: pos}, namespaceURI: namespaceURI, prefix: prefix}
}

func (b *BLangXMLNS) GetNamespaceURI() BLangExpression {
	return b.namespaceURI
}

func (b *BLangXMLNS) GetPrefix() *BLangIdentifier {
	return b.prefix
}

func (b *BLangPackage) GetImports() []*BLangImportPackage {
	return b.Imports
}

func (b *BLangPackage) AddImport(importPkg *BLangImportPackage) {
	b.Imports = append(b.Imports, importPkg)
}

// AddImportIfAbsent adds an import unless the package already imports the same
// organization and module, possibly under a different alias.
func (b *BLangPackage) AddImportIfAbsent(importPkg *BLangImportPackage) {
	for _, existing := range b.Imports {
		if sameImportPackage(existing, importPkg) {
			return
		}
	}
	b.AddImport(importPkg)
}

func sameImportPackage(left, right *BLangImportPackage) bool {
	identifierValue := func(identifier *BLangIdentifier) string {
		if identifier == nil {
			return ""
		}
		return identifier.GetValue()
	}
	if identifierValue(left.OrgName) != identifierValue(right.OrgName) || len(left.PkgNameComps) != len(right.PkgNameComps) {
		return false
	}
	for i := range left.PkgNameComps {
		if left.PkgNameComps[i].GetValue() != right.PkgNameComps[i].GetValue() {
			return false
		}
	}
	return true
}

func (b *BLangPackage) GetNamespaceDeclarations() []*BLangXMLNS {
	return b.XmlnsList
}

func (b *BLangPackage) AddNamespaceDeclaration(xmlnsDecl *BLangXMLNS) {
	b.XmlnsList = append(b.XmlnsList, xmlnsDecl)
}

func (b *BLangPackage) GetConstants() []*BLangVariable {
	return b.Constants
}

func (b *BLangPackage) GetGlobalVariables() []*BLangVariable {
	return b.GlobalVars
}

func (b *BLangPackage) AddGlobalVariable(globalVar *BLangVariable) {
	b.GlobalVars = append(b.GlobalVars, globalVar)
}

func (b *BLangPackage) GetServices() []*BLangService {
	return b.Services
}

func (b *BLangPackage) AddService(service *BLangService) {
	b.Services = append(b.Services, service)
}

func (b *BLangPackage) GetFunctions() []*BLangFunction {
	return b.Functions
}

func (b *BLangPackage) AddFunction(function *BLangFunction) {
	b.Functions = append(b.Functions, function)
}

func (b *BLangPackage) GetTypeDefinitions() []*BLangTypeDefinition {
	return b.TypeDefinitions
}

func (b *BLangPackage) AddTypeDefinition(typeDefinition *BLangTypeDefinition) {
	b.TypeDefinitions = append(b.TypeDefinitions, typeDefinition)
}

func (b *BLangPackage) GetAnnotations() []*BLangAnnotation {
	return b.Annotations
}

func (b *BLangPackage) AddAnnotation(annotation *BLangAnnotation) {
	b.Annotations = append(b.Annotations, annotation)
}

func (b *BLangPackage) GetClassDefinitions() []*BLangClassDefinition {
	return b.ClassDefinitions
}

func (b *BLangPackage) AddClassDefinition(classDefNode *BLangClassDefinition) {
	b.ClassDefinitions = append(b.ClassDefinitions, classDefNode)
}

func NewBLangPackage() *BLangPackage {
	return &BLangPackage{}
}
