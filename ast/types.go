//
// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
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
	"iter"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type ObjectNetworkQuals uint8

const (
	ObjectNetworkQualsNone ObjectNetworkQuals = iota
	ObjectNetworkQualsClient
	ObjectNetworkQualsService
)

type ObjectMemberKind uint8

const (
	ObjectMemberKindField ObjectMemberKind = iota
	ObjectMemberKindMethod
	ObjectMemberKindRemoteMethod
	ObjectMemberKindResourceMethod
)

type BType interface {
	BLangNode
	TypeDescriptor
	SetTypeData(ty TypeData)
	GetTypeData() TypeData
	bTypeGetName() model.Name
	bTypeSetName(name model.Name)
	bTypeGetFlags() model.Flag
	bTypeSetFlags(flags model.Flag)
}

type (
	bLangTypeBase struct {
		bLangNodeBase
		ty      TypeData
		Grouped bool
		name    model.Name
		flags   model.Flag
	}

	BLangArrayType struct {
		bLangTypeBase
		Elemtype   TypeData
		Sizes      []BLangExpression
		Dimensions int
		Definition semtypes.Definition
	}
	BLangBuiltInRefTypeNode struct {
		bLangTypeBase
		TypeKind TypeKind
	}

	BLangValueType struct {
		bLangTypeBase
		TypeKind TypeKind
	}

	BLangUserDefinedType struct {
		// TODO: Is this just type reference? if not we need to rethink this when we have actual user defined types.
		//   If the user defined type is recursive we need a way to get the Definition (similar to array type etc) from that.
		bLangTypeBase
		PkgAlias BLangIdentifier
		TypeName BLangIdentifier
		symbol   model.SymbolRef
	}

	bStructureTypeBase struct {
		names          []string
		fields         []BField // This is only directly included fields, not those included via type inclusions
		TypeInclusions []BType
	}

	bFieldAnnotationBase struct {
		AnnAttachments []BLangAnnotationAttachment
	}

	BField struct {
		// TODO: think how to align this with BLangMemberTypeDesc. Ideally this should be an inclusion on that
		bLangNodeBase
		bFieldAnnotationBase
		Name         model.Name
		Type         BType
		flags        model.Flag
		DefaultExpr  BLangExpression
		DefaultFnRef model.SymbolRef
	}

	bObjectFieldBase struct {
		bLangNodeBase
		name  string
		flags model.Flag
	}

	BObjectField struct {
		bObjectFieldBase
		bFieldAnnotationBase
		Ty BType
	}

	BMethodDecl struct {
		bObjectFieldBase
		BLangFunctionType
		memberKind ObjectMemberKind
		symbol     model.SymbolRef
	}

	BLangObjectType struct {
		bLangTypeBase
		unresolvedInclusionsBase
		Inclusions         []model.SymbolRef      // This needs to be symbol because it could be a class definition as well
		InclusionPositions []diagnostics.Location // Positions of each inclusion, parallel to Inclusions
		members            map[string]ObjectMember
		Definition         semtypes.Definition
		Isolated           bool
		NetworkQuals       ObjectNetworkQuals
	}

	BLangFiniteTypeNode struct {
		bLangTypeBase
		ValueSpace []BLangExpression
	}

	BLangUnionTypeNode struct {
		bLangTypeBase
		lhs TypeData
		rhs TypeData
	}

	BLangIntersectionTypeNode struct {
		bLangTypeBase
		lhs TypeData
		rhs TypeData
	}

	BLangErrorTypeNode struct {
		bLangTypeBase
		DetailType TypeData
	}

	BLangConstrainedType struct {
		bLangTypeBase
		Type       TypeData
		Constraint TypeData
		Definition semtypes.Definition
	}

	BLangStreamType struct {
		bLangTypeBase
		ValueType      TypeData
		CompletionType TypeData
		Definition     semtypes.Definition
	}

	BLangTupleTypeNode struct {
		bLangTypeBase
		Definition semtypes.Definition
		// jBallerina uses BLangSimpleVariabl for this but I think it is better to make it explicit
		Members []BLangMemberTypeDesc
		Rest    BType
	}

	BLangMemberTypeDesc struct {
		bLangNodeBase
		TypeDesc                        TypeDescriptor
		AnnAttachments                  []BLangAnnotationAttachment
		MarkdownDocumentationAttachment *BLangMarkdownDocumentation
	}

	BLangRecordType struct {
		bLangTypeBase
		bStructureTypeBase
		Inclusions []model.SymbolRef
		Definition semtypes.Definition
		RestType   BType
		IsOpen     bool
	}

	BLangFunctionType struct {
		bLangTypeBase
		Definition           semtypes.Definition
		RequiredParams       []*BLangFunctionTypeParam
		RestParam            *BLangFunctionTypeParam
		ReturnTypeDescriptor BType
		ParamListPos         Location
		signatureRef         model.FunctionSignatureRef
	}

	BLangFunctionTypeParam struct {
		bLangNodeBase
		Name                IdentifierNode
		TypeDesc            BType
		InitExpr            BLangExpression
		AnnAttachments      []BLangAnnotationAttachment
		SymbolRef           model.SymbolRef
		IncludedRecordParam bool
	}
)

var (
	_ BNodeWithSymbol   = &BLangUserDefinedType{}
	_ BType             = &BLangStreamType{}
	_ BLangNode         = &BLangStreamType{}
	_ TypeDescriptor    = &BLangStreamType{}
	_ ObjectMember      = &BMethodDecl{}
	_ ObjectMember      = &BObjectField{}
	_ BLangNode         = &BObjectField{}
	_ BLangNode         = &BMethodDecl{}
	_ BNodeWithSymbol   = &BMethodDecl{}
	_ FunctionTypeNode  = &BLangFunctionType{}
	_ FunctionSignature = &BMethodDecl{}
	_ FunctionSignature = &BLangFunctionType{}
	_ Param             = &BLangFunctionTypeParam{}
)

var (
	_ BType     = &BLangUserDefinedType{}
	_ BType     = &BLangBuiltInRefTypeNode{}
	_ BType     = &BLangFunctionType{}
	_ BType     = &BLangRecordType{}
	_ BLangNode = &BLangFunctionType{}
)

var (
	_ BLangNode      = &BLangArrayType{}
	_ BLangNode      = &BLangUserDefinedType{}
	_ BLangNode      = &BLangValueType{}
	_ BLangNode      = &BLangConstrainedType{}
	_ TypeDescriptor = &BLangValueType{}
	_ TypeDescriptor = &BLangConstrainedType{}
	_ BLangNode      = &BLangTupleTypeNode{}
)

func (b *BLangArrayType) GetElementType() TypeData {
	return b.Elemtype
}

func (b *BLangArrayType) GetDimensions() int {
	return b.Dimensions
}

func (b *BLangArrayType) GetSizes() []BLangExpression {
	expressionNodes := make([]BLangExpression, len(b.Sizes))
	copy(expressionNodes, b.Sizes)
	return expressionNodes
}

func (b *BLangArrayType) IsOpenArray() bool {
	return b.Dimensions == 0
}

func (b *bLangTypeBase) IsGrouped() bool {
	return b.Grouped
}

func (b *BLangUserDefinedType) GetPackageAlias() *BLangIdentifier {
	return &b.PkgAlias
}

func (b *BLangUserDefinedType) GetTypeName() *BLangIdentifier {
	return &b.TypeName
}

func (b *BLangUserDefinedType) Symbol() model.SymbolRef {
	return b.symbol
}

func (b *BLangUserDefinedType) SetSymbol(symbolRef model.SymbolRef) {
	b.symbol = symbolRef
}

func (b *BField) GetName() model.Name {
	return b.Name
}

func (b *BField) GetType() Type {
	return b.Type
}

func (b *BField) IsPublic() bool   { return b.flags.Has(model.FlagPublic) }
func (b *BField) IsReadonly() bool { return b.flags.Has(model.FlagReadonly) }
func (b *BField) IsOptional() bool { return b.flags.Has(model.FlagOptional) }
func (b *bFieldAnnotationBase) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.AnnAttachments
}

func (b *bFieldAnnotationBase) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.AnnAttachments = append(b.AnnAttachments, annAttachment)
}

func (b *bStructureTypeBase) Fields() iter.Seq2[string, BField] {
	return func(yield func(string, BField) bool) {
		for i, name := range b.names {
			if !yield(name, b.fields[i]) {
				break
			}
		}
	}
}

func (b *bStructureTypeBase) FieldPtrs() iter.Seq2[string, *BField] {
	return func(yield func(string, *BField) bool) {
		for i, name := range b.names {
			if !yield(name, &b.fields[i]) {
				break
			}
		}
	}
}

func (b *bStructureTypeBase) AddField(name string, field BField) {
	b.names = append(b.names, name)
	b.fields = append(b.fields, field)
}

func (b *BLangObjectType) Members() iter.Seq[ObjectMember] {
	return func(yield func(ObjectMember) bool) {
		for _, member := range b.members {
			if !yield(member) {
				return
			}
		}
	}
}

func (b *BLangObjectType) Member(name string) (ObjectMember, bool) {
	member, ok := b.members[name]
	return member, ok
}

func (b *bObjectFieldBase) Name() string {
	return b.name
}

func (b *bObjectFieldBase) IsPublic() bool {
	return b.flags.Has(model.FlagPublic)
}

func (b *bObjectFieldBase) IsReadonly() bool {
	return b.flags.Has(model.FlagReadonly)
}

func NewBObjectField(pos Location, name string, ty BType, isPublic bool) *BObjectField {
	field := &BObjectField{
		bObjectFieldBase: bObjectFieldBase{bLangNodeBase: bLangNodeBase{pos: pos}, name: name},
		Ty:               ty,
	}
	if isPublic {
		field.flags |= model.FlagPublic
	}
	return field
}

func (b *BObjectField) MemberKind() ObjectMemberKind {
	return ObjectMemberKindField
}

func NewBMethodDecl(pos Location, name string, kind ObjectMemberKind, isPublic bool, functionFlags model.Flag) *BMethodDecl {
	method := &BMethodDecl{
		bObjectFieldBase:  bObjectFieldBase{bLangNodeBase: bLangNodeBase{pos: pos}, name: name},
		BLangFunctionType: BLangFunctionType{bLangTypeBase: bLangTypeBase{flags: functionFlags}},
		memberKind:        kind,
	}
	if isPublic {
		method.flags |= model.FlagPublic
	}
	return method
}

func (b *BMethodDecl) MemberKind() ObjectMemberKind {
	return b.memberKind
}

func (b *BMethodDecl) Symbol() model.SymbolRef {
	return b.symbol
}

func (b *BMethodDecl) SetSymbol(ref model.SymbolRef) {
	b.symbol = ref
}

func (b *bLangTypeBase) GetTypeData() TypeData {
	return b.ty
}

func (b *bLangTypeBase) SetTypeData(ty TypeData) {
	b.ty = ty
}

func (b *bLangTypeBase) bTypeGetName() model.Name {
	return b.name
}

func (b *bLangTypeBase) bTypeSetName(name model.Name) {
	b.name = name
}

func (b *bLangTypeBase) bTypeGetFlags() model.Flag {
	return b.flags
}

func (b *bLangTypeBase) bTypeSetFlags(flags model.Flag) {
	b.flags = flags
}

func NewBField(pos Location, name model.Name, ty BType, defaultExpr BLangExpression, flags model.Flag) BField {
	return BField{
		bLangNodeBase: bLangNodeBase{pos: pos},
		Name:          name,
		Type:          ty,
		flags:         flags,
		DefaultExpr:   defaultExpr,
	}
}

func NewBLangUnionTypeNode(pos Location, lhs, rhs TypeData) *BLangUnionTypeNode {
	return &BLangUnionTypeNode{bLangTypeBase: bLangTypeBase{bLangNodeBase: bLangNodeBase{pos: pos}}, lhs: lhs, rhs: rhs}
}

func NewBLangIntersectionTypeNode(pos Location, lhs, rhs TypeData) *BLangIntersectionTypeNode {
	return &BLangIntersectionTypeNode{bLangTypeBase: bLangTypeBase{bLangNodeBase: bLangNodeBase{pos: pos}}, lhs: lhs, rhs: rhs}
}

func NewBLangErrorTypeNode(pos Location, detailType TypeData, distinct bool) *BLangErrorTypeNode {
	node := &BLangErrorTypeNode{
		bLangTypeBase: bLangTypeBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		DetailType:    detailType,
	}
	if distinct {
		node.flags |= model.FlagDistinct
	}
	return node
}

func (b *BLangFiniteTypeNode) GetValueSet() []BLangExpression {
	values := make([]BLangExpression, len(b.ValueSpace))
	copy(values, b.ValueSpace)
	return values
}

func (b *BLangFiniteTypeNode) AddValue(value BLangExpression) {
	b.ValueSpace = append(b.ValueSpace, value)
}

func (b *BLangUnionTypeNode) Lhs() *TypeData {
	return &b.lhs
}

func (b *BLangUnionTypeNode) Rhs() *TypeData {
	return &b.rhs
}

func (b *BLangIntersectionTypeNode) Lhs() *TypeData {
	return &b.lhs
}

func (b *BLangIntersectionTypeNode) Rhs() *TypeData {
	return &b.rhs
}

func (b *BLangErrorTypeNode) GetDetailType() TypeData {
	return b.DetailType
}

func (b *BLangErrorTypeNode) IsTop() bool {
	return b.DetailType.TypeDescriptor == nil
}

func (b *BLangErrorTypeNode) IsDistinct() bool {
	return b.bTypeGetFlags().Has(model.FlagDistinct)
}

func NewBLangFunctionType(pos Location, requiredParams []*BLangFunctionTypeParam, restParam *BLangFunctionTypeParam, returnType BType, paramListPos Location, flags model.Flag) *BLangFunctionType {
	return &BLangFunctionType{
		bLangTypeBase:        bLangTypeBase{bLangNodeBase: bLangNodeBase{pos: pos}, flags: flags},
		RequiredParams:       requiredParams,
		RestParam:            restParam,
		ReturnTypeDescriptor: returnType,
		ParamListPos:         paramListPos,
	}
}

func (b *BLangFunctionType) SignatureRef() model.FunctionSignatureRef {
	return b.signatureRef
}

func (b *BLangFunctionType) SetSignatureRef(ref model.FunctionSignatureRef) {
	b.signatureRef = ref
}

func (b *BLangFunctionType) Parameters() []Param {
	params := make([]Param, len(b.RequiredParams))
	for i := range b.RequiredParams {
		params[i] = b.RequiredParams[i]
	}
	return params
}

func (b *BLangFunctionType) RestParameter() Param {
	if b.RestParam == nil {
		return nil
	}
	return b.RestParam
}

func (b *BLangFunctionType) ReturnType() TypeDescriptor {
	return b.ReturnTypeDescriptor
}

func (b *BLangFunctionType) IsAnyFunction() bool {
	return b.bTypeGetFlags().Has(model.FlagAnyFunction)
}

func (b *BLangFunctionType) HasExplicitReturnTypeDescriptor() bool {
	return b.bTypeGetFlags().Has(model.FlagExplicitReturnTypeDescriptor)
}

func (b *BLangFunctionType) IsIsolated() bool { return b.bTypeGetFlags().Has(model.FlagIsolated) }
func (b *BLangFunctionType) IsTransactional() bool {
	return b.bTypeGetFlags().Has(model.FlagTransactional)
}

func (b *BLangConstrainedType) GetType() TypeData {
	return b.Type
}

func (b *BLangConstrainedType) GetConstraint() TypeData {
	return b.Constraint
}

// ConstraintKind returns the kind of the constrained type's base (the head
// before the type parameter), e.g. TypeKindMap for `map<T>` or
// TypeKindTypeDesc for `typedesc<T>`.
func (b *BLangConstrainedType) ConstraintKind() TypeKind {
	switch t := b.Type.TypeDescriptor.(type) {
	case *BLangBuiltInRefTypeNode:
		return t.TypeKind
	case *BLangValueType:
		return t.TypeKind
	}
	panic("BLangConstrainedType.Type has unexpected type descriptor")
}

func NewBLangStreamType(valueType, completionType TypeData) *BLangStreamType {
	return &BLangStreamType{
		ValueType:      valueType,
		CompletionType: completionType,
	}
}

func (b *BLangStreamType) GetTypeKind() TypeKind {
	return TypeKindStream
}

func (b *BLangTupleTypeNode) GetMembers() []*BLangMemberTypeDesc {
	members := make([]*BLangMemberTypeDesc, len(b.Members))
	for i := range b.Members {
		members[i] = &b.Members[i]
	}
	return members
}

func (b *BLangTupleTypeNode) GetRest() TypeDescriptor {
	if b.Rest == nil {
		return nil
	}
	return b.Rest
}

func (b *BLangMemberTypeDesc) GetTypeDesc() TypeDescriptor {
	return b.TypeDesc
}

func (b *BLangMemberTypeDesc) GetAnnotationAttachments() []BLangAnnotationAttachment {
	return b.AnnAttachments
}

func (b *BLangMemberTypeDesc) AddAnnotationAttachment(annAttachment BLangAnnotationAttachment) {
	b.AnnAttachments = append(b.AnnAttachments, annAttachment)
}

func (b *BLangMemberTypeDesc) GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation {
	return b.MarkdownDocumentationAttachment
}

func (b *BLangFunctionTypeParam) GetName() *string {
	if b.Name == nil {
		return nil
	}
	name := b.Name.GetValue()
	return &name
}

func (b *BLangFunctionTypeParam) ParamName() string {
	if b.Name == nil {
		return ""
	}
	return b.Name.GetValue()
}

func (b *BLangFunctionTypeParam) Type() BType {
	return b.TypeDesc
}

func (b *BLangFunctionTypeParam) DefaultExpr() BLangExpression {
	return b.InitExpr
}

func (b *BLangFunctionTypeParam) Symbol() model.SymbolRef {
	return b.SymbolRef
}

func (b *BLangFunctionTypeParam) IsDefaultable() bool {
	return b.InitExpr != nil
}

func (b *BLangFunctionTypeParam) GetTypeDesc() Type {
	return b.TypeDesc
}

func (b *BLangFunctionTypeParam) IsIncludedRecordParam() bool {
	return b.IncludedRecordParam
}

func (b *BLangFunctionTypeParam) SetIncludedRecordParam() {
	b.IncludedRecordParam = true
}

func (b *BLangFunctionType) GetParams() []*BLangFunctionTypeParam {
	params := make([]*BLangFunctionTypeParam, len(b.RequiredParams))
	copy(params, b.RequiredParams)
	return params
}

func (b *BLangFunctionType) GetRestParam() *BLangFunctionTypeParam {
	if b.RestParam == nil {
		return nil
	}
	return b.RestParam
}

func (b *BLangFunctionType) GetReturnTypeNode() TypeDescriptor {
	return b.ReturnTypeDescriptor
}

func (b *BLangRecordType) GetRestFieldType() TypeData {
	if b.RestType == nil {
		return TypeData{}
	}
	return b.RestType.GetTypeData()
}

func (b *BLangRecordType) GetFields() iter.Seq2[string, *BField] {
	return func(yield func(string, *BField) bool) {
		for i, name := range b.names {
			if !yield(name, &b.fields[i]) {
				return
			}
		}
	}
}

func NewBLangObjectType(pos Location) *BLangObjectType {
	return &BLangObjectType{
		bLangTypeBase: bLangTypeBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		members:       make(map[string]ObjectMember),
	}
}

// AddMember insert a new member. If there was already a member by the same name return true
func (b *BLangObjectType) AddMember(member ObjectMember) bool {
	name := member.Name()
	if _, hadValue := b.members[name]; hadValue {
		return true
	}
	b.members[name] = member
	return false
}
