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

package bir

import (
	"ballerina/semtypes"
	"ballerina/values"
)

type BIRNonTerminator = BIRInstruction

type BIRAssignInstruction interface {
	BIRInstruction
	GetLhsOperand() *BIROperand
}

type MappingConstructorEntry interface {
	IsKeyValuePair() bool
	ValueOp() *BIROperand
}

type (
	Move struct {
		BIRInstructionBase
		RhsOp *BIROperand
	}
	BinaryOp struct {
		BIRInstructionBase
		Kind   InstructionKind
		RhsOp1 BIROperand
		RhsOp2 BIROperand
	}

	UnaryOp struct {
		BIRInstructionBase
		Kind  InstructionKind
		RhsOp *BIROperand
	}

	ConstantLoad struct {
		BIRInstructionBase
		Value any
	}

	FieldAccess struct {
		BIRInstructionBase
		Kind  InstructionKind
		KeyOp *BIROperand
		RhsOp *BIROperand
		// Filler is set for filling load instructions on maps so the
		// runtime can insert a fresh filler value for absent keys.
		Filler values.FillerFactory
	}

	NewArray struct {
		BIRInstructionBase
		SizeOp     *BIROperand
		Type       semtypes.SemType
		Values     []*BIROperand
		Filler     values.FillerFactory
		IsReadonly bool
	}

	// JBallerina call this NewStruct but prints as NewMap
	NewMap struct {
		BIRInstructionBase
		Type       semtypes.SemType
		Values     []MappingConstructorEntry
		Defaults   []MappingConstructorDefaultEntry
		IsReadonly bool
	}

	MappingConstructorDefaultEntry struct {
		FieldName         string
		FunctionLookupKey string
	}

	NewError struct {
		BIRInstructionBase
		Type      semtypes.SemType
		TypeName  string
		MessageOp *BIROperand
		CauseOp   *BIROperand
		DetailOp  *BIROperand
	}

	TypeCast struct {
		BIRInstructionBase
		RhsOp *BIROperand
		// I don't think you need to the type desc part given only way you need to create a new value is with
		// numeric conversions, which can be done with pure types
		Type semtypes.SemType
	}

	TypeTest struct {
		BIRInstructionBase
		RhsOp      *BIROperand
		Type       semtypes.SemType
		IsNegation bool
	}

	NewObject struct {
		BIRInstructionBase
		ClassDefRef string
	}

	NewStream struct {
		BIRInstructionBase
		StreamType semtypes.SemType
		ImplOp     *BIROperand
	}

	StreamNext struct {
		BIRInstructionBase
		StreamOp *BIROperand
	}

	StreamClose struct {
		BIRInstructionBase
		StreamOp *BIROperand
	}

	FPLoad struct {
		BIRInstructionBase
		FunctionLookupKey string
		Type              semtypes.SemType
		IsClosure         bool
	}

	PushScopeFrame struct {
		BIRInstructionBase
		NumLocals int
	}

	PopScopeFrame struct {
		BIRInstructionBase
	}

	NewXMLElement struct {
		BIRInstructionBase
		NameOp       *BIROperand
		ChildrenOp   *BIROperand
		AttrsOp      *BIROperand
		NamespacesOp *BIROperand
	}

	NewXMLPI struct {
		BIRInstructionBase
		TargetOp *BIROperand
		DataOp   *BIROperand
	}

	NewXMLComment struct {
		BIRInstructionBase
		BodyOp *BIROperand
	}

	NewXMLText struct {
		BIRInstructionBase
		BodyOp *BIROperand
	}

	NewXMLSequence struct {
		BIRInstructionBase
		Children []*BIROperand
	}

	EvalTemplateExpr struct {
		BIRInstructionBase
		Kind             TemplateKind
		Strings          []string
		LiteralsTotalLen int
		Insertions       []*BIROperand
	}
)

type TemplateKind uint8

const (
	TemplateKindString TemplateKind = iota
	TemplateKindXML
	TemplateKindRaw
)

type (
	MappingConstructorKeyValueEntry struct {
		keyOp   *BIROperand
		valueOp *BIROperand
	}
)

var (
	_ BIRAssignInstruction    = &Move{}
	_ BIRAssignInstruction    = &BinaryOp{}
	_ BIRAssignInstruction    = &UnaryOp{}
	_ BIRAssignInstruction    = &ConstantLoad{}
	_ BIRInstruction          = &FieldAccess{}
	_ BIRInstruction          = &NewArray{}
	_ BIRInstruction          = &TypeCast{}
	_ BIRAssignInstruction    = &TypeTest{}
	_ BIRInstruction          = &NewMap{}
	_ BIRAssignInstruction    = &NewError{}
	_ BIRAssignInstruction    = &NewObject{}
	_ BIRAssignInstruction    = &NewStream{}
	_ BIRAssignInstruction    = &StreamNext{}
	_ BIRAssignInstruction    = &StreamClose{}
	_ BIRAssignInstruction    = &FPLoad{}
	_ BIRInstruction          = &PushScopeFrame{}
	_ BIRInstruction          = &PopScopeFrame{}
	_ BIRAssignInstruction    = &NewXMLElement{}
	_ BIRAssignInstruction    = &NewXMLPI{}
	_ BIRAssignInstruction    = &NewXMLComment{}
	_ BIRAssignInstruction    = &NewXMLText{}
	_ BIRAssignInstruction    = &NewXMLSequence{}
	_ BIRAssignInstruction    = &EvalTemplateExpr{}
	_ MappingConstructorEntry = &MappingConstructorKeyValueEntry{}
)

func (m *Move) GetLhsOperand() *BIROperand {
	return m.LhsOp
}

func (m *Move) GetKind() InstructionKind {
	return INSTRUCTION_KIND_MOVE
}

func NewMove(fromOperand, toOperand *BIROperand, pos Location) *Move {
	return &Move{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: toOperand,
		},
		RhsOp: fromOperand,
	}
}

func (b *BinaryOp) GetLhsOperand() *BIROperand {
	return b.LhsOp
}

func (b *BinaryOp) GetKind() InstructionKind {
	return b.Kind
}

func NewBinaryOp(kind InstructionKind, lhsOp, rhsOp1, rhsOp2 *BIROperand, pos Location) *BinaryOp {
	return &BinaryOp{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Kind:   kind,
		RhsOp1: *rhsOp1,
		RhsOp2: *rhsOp2,
	}
}

func (u *UnaryOp) GetLhsOperand() *BIROperand {
	return u.LhsOp
}

func (u *UnaryOp) GetKind() InstructionKind {
	return u.Kind
}

func NewUnaryOp(kind InstructionKind, lhsOp, rhsOp *BIROperand, pos Location) *UnaryOp {
	return &UnaryOp{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Kind:  kind,
		RhsOp: rhsOp,
	}
}

func (c *ConstantLoad) GetLhsOperand() *BIROperand {
	return c.LhsOp
}

func (c *ConstantLoad) GetKind() InstructionKind {
	return INSTRUCTION_KIND_CONST_LOAD
}

func NewConstantLoad(lhsOp *BIROperand, value any, pos Location) *ConstantLoad {
	return &ConstantLoad{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Value: value,
	}
}

func (f *FieldAccess) GetLhsOperand() *BIROperand {
	return f.LhsOp
}

func (f *FieldAccess) GetKind() InstructionKind {
	return f.Kind
}

func NewFieldAccess(kind InstructionKind, lhsOp, keyOp, rhsOp *BIROperand, pos Location) *FieldAccess {
	return &FieldAccess{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Kind:  kind,
		KeyOp: keyOp,
		RhsOp: rhsOp,
	}
}

func (n *NewArray) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func (n *NewArray) GetKind() InstructionKind {
	return INSTRUCTION_KIND_NEW_ARRAY
}

func NewArrayConstructor(typ semtypes.SemType, lhsOp, sizeOp *BIROperand, values []*BIROperand, filler values.FillerFactory, isReadonly bool, pos Location) *NewArray {
	return &NewArray{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Type:       typ,
		SizeOp:     sizeOp,
		Values:     values,
		Filler:     filler,
		IsReadonly: isReadonly,
	}
}

func (t *TypeCast) GetLhsOperand() *BIROperand {
	return t.LhsOp
}

func (t *TypeCast) GetKind() InstructionKind {
	return INSTRUCTION_KIND_TYPE_CAST
}

func NewTypeCast(typ semtypes.SemType, lhsOp, rhsOp *BIROperand, pos Location) *TypeCast {
	return &TypeCast{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Type:  typ,
		RhsOp: rhsOp,
	}
}

func (t *TypeTest) GetLhsOperand() *BIROperand {
	return t.LhsOp
}

func (t *TypeTest) GetKind() InstructionKind {
	return INSTRUCTION_KIND_TYPE_TEST
}

func NewTypeTest(typ semtypes.SemType, lhsOp, rhsOp *BIROperand, pos Location) *TypeTest {
	return &TypeTest{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Type:  typ,
		RhsOp: rhsOp,
	}
}

func (f *FPLoad) GetLhsOperand() *BIROperand {
	return f.LhsOp
}

func (f *FPLoad) GetKind() InstructionKind {
	return INSTRUCTION_KIND_FP_LOAD
}

func (n *NewMap) GetKind() InstructionKind {
	return INSTRUCTION_KIND_NEW_STRUCTURE
}

func NewMapConstructor(typ semtypes.SemType, lhsOp *BIROperand, values []MappingConstructorEntry, defaults []MappingConstructorDefaultEntry, isReadonly bool, pos Location) *NewMap {
	return &NewMap{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Type:       typ,
		Values:     values,
		Defaults:   defaults,
		IsReadonly: isReadonly,
	}
}

func (n *NewError) GetKind() InstructionKind {
	return INSTRUCTION_KIND_NEW_ERROR
}

func (n *NewError) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func NewErrorConstructor(typ semtypes.SemType, typeName string, lhsOp, messageOp, causeOp, detailOp *BIROperand, pos Location) *NewError {
	return &NewError{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		Type:      typ,
		TypeName:  typeName,
		MessageOp: messageOp,
		CauseOp:   causeOp,
		DetailOp:  detailOp,
	}
}

func (n *NewMap) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func (n *NewObject) GetKind() InstructionKind {
	return INSTRUCTION_KIND_NEW_INSTANCE
}

func (n *NewObject) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func NewObjectConstructor(classDefRef string, lhsOp *BIROperand, pos Location) *NewObject {
	return &NewObject{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		ClassDefRef: classDefRef,
	}
}

func (n *NewStream) GetKind() InstructionKind {
	return INSTRUCTION_KIND_NEW_STREAM
}

func (n *NewStream) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func NewStreamConstructor(streamType semtypes.SemType, lhsOp, implOp *BIROperand, pos Location) *NewStream {
	return &NewStream{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		StreamType: streamType,
		ImplOp:     implOp,
	}
}

func (n *StreamNext) GetKind() InstructionKind {
	return INSTRUCTION_KIND_STREAM_NEXT
}

func (n *StreamNext) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func NewStreamNext(lhsOp, streamOp *BIROperand, pos Location) *StreamNext {
	return &StreamNext{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		StreamOp: streamOp,
	}
}

func (n *StreamClose) GetKind() InstructionKind {
	return INSTRUCTION_KIND_STREAM_CLOSE
}

func (n *StreamClose) GetLhsOperand() *BIROperand {
	return n.LhsOp
}

func NewStreamClose(lhsOp, streamOp *BIROperand, pos Location) *StreamClose {
	return &StreamClose{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		StreamOp: streamOp,
	}
}

func (p *PushScopeFrame) GetKind() InstructionKind {
	return INSTRUCTION_KIND_PUSH_SCOPE
}

func (p *PushScopeFrame) GetLhsOperand() *BIROperand {
	return nil
}

func (p *PopScopeFrame) GetKind() InstructionKind {
	return INSTRUCTION_KIND_POP_SCOPE
}

func (p *PopScopeFrame) GetLhsOperand() *BIROperand {
	return nil
}

func NewFPLoad(functionLookupKey string, typ semtypes.SemType, lhsOp *BIROperand, pos Location) *FPLoad {
	return &FPLoad{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{
				Pos: pos,
			},
			LhsOp: lhsOp,
		},
		FunctionLookupKey: functionLookupKey,
		Type:              typ,
	}
}

func NewMappingConstructorKeyValueEntry(keyOp, valueOp *BIROperand) *MappingConstructorKeyValueEntry {
	return &MappingConstructorKeyValueEntry{
		keyOp:   keyOp,
		valueOp: valueOp,
	}
}

func (m *MappingConstructorKeyValueEntry) IsKeyValuePair() bool {
	return true
}

func (m *MappingConstructorKeyValueEntry) ValueOp() *BIROperand {
	return m.valueOp
}

func (m *MappingConstructorKeyValueEntry) KeyOp() *BIROperand {
	return m.keyOp
}

func (n *NewXMLElement) GetLhsOperand() *BIROperand { return n.LhsOp }
func (n *NewXMLElement) GetKind() InstructionKind   { return INSTRUCTION_KIND_NEW_XML_ELEMENT }

func NewXMLElementInstr(lhsOp, nameOp, childrenOp, attrsOp, namespacesOp *BIROperand, pos Location) *NewXMLElement {
	return &NewXMLElement{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		NameOp:       nameOp,
		ChildrenOp:   childrenOp,
		AttrsOp:      attrsOp,
		NamespacesOp: namespacesOp,
	}
}

func (n *NewXMLPI) GetLhsOperand() *BIROperand { return n.LhsOp }
func (n *NewXMLPI) GetKind() InstructionKind   { return INSTRUCTION_KIND_NEW_XML_PI }

func NewXMLPIInstr(lhsOp, targetOp, dataOp *BIROperand, pos Location) *NewXMLPI {
	return &NewXMLPI{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		TargetOp: targetOp,
		DataOp:   dataOp,
	}
}

func (n *NewXMLComment) GetLhsOperand() *BIROperand { return n.LhsOp }
func (n *NewXMLComment) GetKind() InstructionKind   { return INSTRUCTION_KIND_NEW_XML_COMMENT }

func NewXMLCommentInstr(lhsOp, bodyOp *BIROperand, pos Location) *NewXMLComment {
	return &NewXMLComment{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		BodyOp: bodyOp,
	}
}

func (n *NewXMLText) GetLhsOperand() *BIROperand { return n.LhsOp }
func (n *NewXMLText) GetKind() InstructionKind   { return INSTRUCTION_KIND_NEW_XML_TEXT }

func NewXMLTextInstr(lhsOp, bodyOp *BIROperand, pos Location) *NewXMLText {
	return &NewXMLText{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		BodyOp: bodyOp,
	}
}

func (n *NewXMLSequence) GetLhsOperand() *BIROperand { return n.LhsOp }
func (n *NewXMLSequence) GetKind() InstructionKind   { return INSTRUCTION_KIND_NEW_XML_SEQUENCE }

func NewXMLSequenceInstr(lhsOp *BIROperand, children []*BIROperand, pos Location) *NewXMLSequence {
	return &NewXMLSequence{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		Children: children,
	}
}

func (e *EvalTemplateExpr) GetLhsOperand() *BIROperand { return e.LhsOp }
func (e *EvalTemplateExpr) GetKind() InstructionKind   { return INSTRUCTION_KIND_EVAL_TEMPLATE_EXPR }

func NewEvalTemplateExpr(kind TemplateKind, strings []string, insertions []*BIROperand, lhsOp *BIROperand, pos Location) *EvalTemplateExpr {
	literalTotalLen := 0
	for _, each := range strings {
		literalTotalLen += len(each)
	}
	return &EvalTemplateExpr{
		BIRInstructionBase: BIRInstructionBase{
			BIRNodeBase: BIRNodeBase{Pos: pos},
			LhsOp:       lhsOp,
		},
		Kind:             kind,
		Strings:          strings,
		LiteralsTotalLen: literalTotalLen,
		Insertions:       insertions,
	}
}
