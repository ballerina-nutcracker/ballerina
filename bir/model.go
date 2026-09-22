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

// Package bir declares types used to represent Ballerina Intermediate Representation
package bir

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
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

	BIRInstructionBase struct {
		BIRNodeBase
		// Kind InstructionKind
		LhsOp *BIROperand
		Scope *BIRScope
	}

	BIRScope struct {
		ID     int
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
		Name        model.Name
		LookupKey   string
		Annotations values.AnnotationValues
		Fields      []ObjectField
		VTable      map[string]*BIRFunction
		RTable      map[string][]BIRResourceMethod
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
		PkgID              *model.PackageID
		GlobalVarLookupKey string
	}

	BIRFunction struct {
		BIRNodeBase
		Name              model.Name
		OriginalName      model.Name
		Flags             model.Flag
		RequiredParams    []BIRParameter
		RestParams        *BIRParameter
		ArgsCount         int
		LocalVars         []BIRLocalVariableDcl
		ReturnVariable    *BIRLocalVariableDcl
		Parameters        []BIRFunctionParameter
		BasicBlocks       []BIRBasicBlock
		ErrorTable        []BIRErrorEntry
		FunctionLookupKey string
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
		ID           model.Name
		Instructions []BIRNonTerminator
		Terminator   BIRTerminator
	}

	BIRParameter struct {
		BIRNodeBase
		Name        model.Name
		Flags       model.Flag
		Annotations values.AnnotationValues
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

// ParamLocalVarOffset returns the local-variable index of a function's first
// declared parameter. Local slot zero is the return value and attached
// functions additionally reserve slot one for self.
func (f *BIRFunction) ParamLocalVarOffset() int {
	if f.Flags.Has(model.FlagAttached) {
		return 2
	}
	return 1
}

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

func AbsoluteAddress(baseIndex, frameIndex int) Address {
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

type VarKind uint8

const (
	VarKindLocal VarKind = iota + 1
	VarKindReturn
	VarKindGlobal
)

type VarScope uint8

const (
	VarScopeFunction VarScope = iota + 1
	VarScopeGlobal
)

type InstructionKind uint8

const (
	InstructionKindGoto InstructionKind = iota + 1
	InstructionKindCall
	InstructionKindBranch
	InstructionKindReturn
	InstructionKindAsyncCall
	InstructionKindWait
	InstructionKindFPCall
	InstructionKindLock
	InstructionKindUnlock
	InstructionKindResourceCall
	InstructionKindAlternateWait
	InstructionKindMove
	InstructionKindConstLoad
	InstructionKindNewStructure
	InstructionKindMapStore
	InstructionKindMapLoad
	InstructionKindNewArray
	InstructionKindArrayStore
	InstructionKindArrayLoad
	InstructionKindNewError
	InstructionKindTypeCast
	InstructionKindTypeTest
	InstructionKindNewInstance
	InstructionKindObjectStore
	InstructionKindObjectLoad
	InstructionKindPanic
	InstructionKindFPLoad
	InstructionKindNewXMLElement
	InstructionKindNewXMLText
	InstructionKindNewXMLComment
	InstructionKindNewXMLPI
	InstructionKindNewXMLSequence
	InstructionKindNewStream
	InstructionKindStreamNext
	InstructionKindStreamClose
	InstructionKindArrayFillingLoad
	InstructionKindMapFillingLoad
	InstructionKindEvalTemplateExpr
	InstructionKindAdd
	InstructionKindSub
	InstructionKindMul
	InstructionKindDiv
	InstructionKindMod
	InstructionKindEqual
	InstructionKindNotEqual
	InstructionKindGreaterThan
	InstructionKindGreaterEqual
	InstructionKindLessThan
	InstructionKindLessEqual
	InstructionKindAnd
	InstructionKindOr
	InstructionKindRefEqual
	InstructionKindRefNotEqual
	InstructionKindAnnotAccess
	InstructionKindNot
	InstructionKindNegate
	InstructionKindBitwiseAnd
	InstructionKindBitwiseOr
	InstructionKindBitwiseXor
	InstructionKindBitwiseLeftShift
	InstructionKindBitwiseRightShift
	InstructionKindBitwiseUnsignedRightShift
	InstructionKindBitwiseComplement
	InstructionKindPushScope
	InstructionKindPopScope
	InstructionKindWaitAll
	InstructionKindXMLFilter
)

func BB(number int) BIRBasicBlock {
	return BIRBasicBlock{
		Number: number,
		ID:     model.Name(fmt.Sprintf("bb%d", number)),
	}
}
