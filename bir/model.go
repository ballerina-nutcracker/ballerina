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
	"fmt"

	"ballerina/model"
	"ballerina/semtypes"
)

type ConstValue struct {
	Value any
}

type BIRInstruction interface {
	GetKind() InstructionKind
	GetPos() Location
}

func (b BIRNodeBase) GetPos() Location {
	return b.Pos
}

type BIRVariableDcl interface {
	GetType() semtypes.SemType
	GetName() model.Name
}

type (
	BIRNodeBase struct {
		Pos Location
	}

	// BIRAbstractInstruction
	BIRInstructionBase struct {
		BIRNodeBase
		// Kind InstructionKind
		LhsOp *BIROperand
		Scope *BIRScope
	}

	BIRScope struct {
		Id     int
		Parent *BIRScope
	}

	BIRPackage struct {
		BIRNodeBase
		PackageID             *model.PackageID
		GlobalVars            map[string]BIRGlobalVariableDcl
		Functions             []BIRFunction
		InitFunction          *BIRFunction
		ClassDefs             []BIRClassDef
		MainFunction          *BIRFunction
		StartFunction         *BIRFunction
		GracefulStopFunction  *BIRFunction
		ImmediateStopFunction *BIRFunction
	}

	ObjectField struct {
		Name string
		Ty   semtypes.SemType
	}

	BIRClassDef struct {
		Name      model.Name
		LookupKey string
		Fields    []ObjectField
		VTable    map[string]*BIRFunction
		RTable    map[string][]BIRResourceMethod
	}

	BIRResourceMethod struct {
		PathSegments  []ResourcePathSegmentDef
		RestSegmentTy semtypes.SemType
		Fn            *BIRFunction
	}

	ResourcePathSegmentDef struct {
		// Ty is a singleton string type for literal path segments
		// and the parameter type for path-parameter segments.
		Ty semtypes.SemType
	}

	birVariableDclBase struct {
		BIRNodeBase
		Type semtypes.SemType
		Name model.Name
	}

	BIRLocalVariableDcl struct {
		birVariableDclBase
	}

	BIRGlobalVariableDcl struct {
		birVariableDclBase
		Flags              model.Flag
		PkgId              *model.PackageID
		GlobalVarLookupKey string
	}

	BIRFunction struct {
		BIRNodeBase
		Name           model.Name
		OriginalName   model.Name
		Flags          model.Flag
		RequiredParams []BIRParameter
		RestParams     *BIRParameter
		ArgsCount      int
		LocalVars      []BIRLocalVariableDcl
		ReturnVariable *BIRLocalVariableDcl
		Parameters     []BIRFunctionParameter
		BasicBlocks    []BIRBasicBlock
		ErrorTable     []BIRErrorEntry
		// FIXME:
		DependentGlobalVars []BIRGlobalVariableDcl
		FunctionLookupKey   string
	}

	BIRErrorEntry struct {
		Start   int
		End     int
		Target  int
		ErrorOp *BIROperand
	}

	BIRBasicBlock struct {
		BIRNodeBase
		Number       int
		Id           model.Name
		Instructions []BIRNonTerminator
		Terminator   BIRTerminator
	}

	BIRParameter struct {
		BIRNodeBase
		Name  model.Name
		Flags model.Flag
	}

	BIRFunctionParameter struct {
		BIRLocalVariableDcl
		HasDefaultExpr  bool
		IsPathParameter bool
	}

	BIROperand struct {
		BIRNodeBase
		VariableDcl BIRVariableDcl
		Address     Address
	}
)

type Address struct {
	Mode       AddressingMode
	FrameIndex int // Index within the frame
	BaseIndex  int // Number of frames to move up
}

type AddressingMode uint8

const (
	AddressingModeRelative AddressingMode = iota
	AddressingModeAbsolute
)

func RelativeAddress(frameIndex int) Address {
	return Address{Mode: AddressingModeRelative, FrameIndex: frameIndex}
}

func absoluteAddress(baseIndex, frameIndex int) Address {
	return Address{Mode: AddressingModeAbsolute, BaseIndex: baseIndex, FrameIndex: frameIndex}
}

var (
	_ BIRVariableDcl = &BIRLocalVariableDcl{}
	_ BIRVariableDcl = &BIRGlobalVariableDcl{}
)

func (v *birVariableDclBase) GetType() semtypes.SemType {
	return v.Type
}

func (v *birVariableDclBase) GetName() model.Name {
	return v.Name
}

func (v *birVariableDclBase) SetName(name model.Name) {
	v.Name = name
}

func (v *birVariableDclBase) SetPos(pos Location) {
	v.Pos = pos
}

// TODO: add interface asserts

type VarKind uint8

const (
	VAR_KIND_LOCAL VarKind = iota + 1
	VAR_KIND_ARG
	VAR_KIND_TEMP
	VAR_KIND_RETURN
	VAR_KIND_GLOBAL
	VAR_KIND_SELF
	VAR_KIND_CONSTANT
	VAR_KIND_SYNTHETIC
)

type VarScope uint8

const (
	VAR_SCOPE_FUNCTION VarScope = iota + 1
	VAR_SCOPE_GLOBAL
)

type InstructionKind uint8

const (
	INSTRUCTION_KIND_GOTO InstructionKind = iota + 1
	INSTRUCTION_KIND_CALL
	INSTRUCTION_KIND_BRANCH
	INSTRUCTION_KIND_RETURN
	INSTRUCTION_KIND_ASYNC_CALL
	INSTRUCTION_KIND_WAIT
	INSTRUCTION_KIND_FP_CALL
	INSTRUCTION_KIND_WK_RECEIVE
	INSTRUCTION_KIND_WK_SEND
	INSTRUCTION_KIND_FLUSH
	INSTRUCTION_KIND_LOCK
	INSTRUCTION_KIND_FIELD_LOCK
	INSTRUCTION_KIND_UNLOCK
	INSTRUCTION_KIND_WAIT_ALL
	INSTRUCTION_KIND_WK_ALT_RECEIVE
	INSTRUCTION_KIND_WK_MULTIPLE_RECEIVE
	INSTRUCTION_KIND_RESOURCE_CALL
)

const (
	INSTRUCTION_KIND_MOVE InstructionKind = iota + 20
	INSTRUCTION_KIND_CONST_LOAD
	INSTRUCTION_KIND_NEW_STRUCTURE
	INSTRUCTION_KIND_MAP_STORE
	INSTRUCTION_KIND_MAP_LOAD
	INSTRUCTION_KIND_NEW_ARRAY
	INSTRUCTION_KIND_ARRAY_STORE
	INSTRUCTION_KIND_ARRAY_LOAD
	INSTRUCTION_KIND_NEW_ERROR
	INSTRUCTION_KIND_TYPE_CAST
	INSTRUCTION_KIND_IS_LIKE
	INSTRUCTION_KIND_TYPE_TEST
	INSTRUCTION_KIND_NEW_INSTANCE
	INSTRUCTION_KIND_OBJECT_STORE
	INSTRUCTION_KIND_OBJECT_LOAD
	INSTRUCTION_KIND_PANIC
	INSTRUCTION_KIND_FP_LOAD
	INSTRUCTION_KIND_STRING_LOAD
	INSTRUCTION_KIND_NEW_XML_ELEMENT
	INSTRUCTION_KIND_NEW_XML_TEXT
	INSTRUCTION_KIND_NEW_XML_COMMENT
	INSTRUCTION_KIND_NEW_XML_PI
	INSTRUCTION_KIND_NEW_XML_SEQUENCE
	INSTRUCTION_KIND_NEW_XML_QNAME
	INSTRUCTION_KIND_NEW_STRING_XML_QNAME
	INSTRUCTION_KIND_XML_SEQ_STORE
	INSTRUCTION_KIND_XML_SEQ_LOAD
	INSTRUCTION_KIND_XML_LOAD
	INSTRUCTION_KIND_XML_LOAD_ALL
	INSTRUCTION_KIND_XML_ATTRIBUTE_LOAD
	INSTRUCTION_KIND_XML_ATTRIBUTE_STORE
	INSTRUCTION_KIND_NEW_TABLE
	INSTRUCTION_KIND_NEW_TYPEDESC
	INSTRUCTION_KIND_NEW_STREAM
	INSTRUCTION_KIND_STREAM_NEXT
	INSTRUCTION_KIND_STREAM_CLOSE
	INSTRUCTION_KIND_TABLE_STORE
	INSTRUCTION_KIND_TABLE_LOAD
	INSTRUCTION_KIND_ARRAY_FILLING_LOAD
	INSTRUCTION_KIND_MAP_FILLING_LOAD
	INSTRUCTION_KIND_EVAL_TEMPLATE_EXPR
)

const (
	INSTRUCTION_KIND_ADD InstructionKind = iota + 61
	INSTRUCTION_KIND_SUB
	INSTRUCTION_KIND_MUL
	INSTRUCTION_KIND_DIV
	INSTRUCTION_KIND_MOD
	INSTRUCTION_KIND_EQUAL
	INSTRUCTION_KIND_NOT_EQUAL
	INSTRUCTION_KIND_GREATER_THAN
	INSTRUCTION_KIND_GREATER_EQUAL
	INSTRUCTION_KIND_LESS_THAN
	INSTRUCTION_KIND_LESS_EQUAL
	INSTRUCTION_KIND_AND
	INSTRUCTION_KIND_OR
	INSTRUCTION_KIND_REF_EQUAL
	INSTRUCTION_KIND_REF_NOT_EQUAL
	INSTRUCTION_KIND_CLOSED_RANGE
	INSTRUCTION_KIND_HALF_OPEN_RANGE
	INSTRUCTION_KIND_ANNOT_ACCESS
)

const (
	INSTRUCTION_KIND_TYPEOF InstructionKind = iota + 80
	INSTRUCTION_KIND_NOT
	INSTRUCTION_KIND_NEGATE
	INSTRUCTION_KIND_BITWISE_AND
	INSTRUCTION_KIND_BITWISE_OR
	INSTRUCTION_KIND_BITWISE_XOR
	INSTRUCTION_KIND_BITWISE_LEFT_SHIFT
	INSTRUCTION_KIND_BITWISE_RIGHT_SHIFT
	INSTRUCTION_KIND_BITWISE_UNSIGNED_RIGHT_SHIFT

	INSTRUCTION_KIND_NEW_REG_EXP
	INSTRUCTION_KIND_NEW_RE_DISJUNCTION
	INSTRUCTION_KIND_NEW_RE_SEQUENCE
	INSTRUCTION_KIND_NEW_RE_ASSERTION
	INSTRUCTION_KIND_NEW_RE_ATOM_QUANTIFIER
	INSTRUCTION_KIND_NEW_RE_LITERAL_CHAR_ESCAPE
	INSTRUCTION_KIND_NEW_RE_CHAR_CLASS
	INSTRUCTION_KIND_NEW_RE_CHAR_SET
	INSTRUCTION_KIND_NEW_RE_CHAR_SET_RANGE
	INSTRUCTION_KIND_NEW_RE_CAPTURING_GROUP
	INSTRUCTION_KIND_NEW_RE_FLAG_EXPR
	INSTRUCTION_KIND_NEW_RE_FLAG_ON_OFF
	INSTRUCTION_KIND_NEW_RE_QUANTIFIER
	INSTRUCTION_KIND_RECORD_DEFAULT_FP_LOAD
	INSTRUCTION_KIND_BITWISE_COMPLEMENT
	INSTRUCTION_KIND_PUSH_SCOPE
	INSTRUCTION_KIND_POP_SCOPE
)

const (
	INSTRUCTION_KIND_PLATFORM InstructionKind = 128
)

func BB(number int) BIRBasicBlock {
	return BIRBasicBlock{
		Number: number,
		Id:     model.Name(fmt.Sprintf("bb%d", number)),
	}
}
