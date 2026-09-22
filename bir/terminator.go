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
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type BIRTerminator = BIRInstruction

type CallKind uint8

const (
	CallKindFunction CallKind = iota
	CallKindFunctionPointer
	CallKindMethod
	CallKindResource
)

type (
	BIRTerminatorBase struct {
		BIRInstructionBase
		ThenBB *BIRBasicBlock
	}

	Goto struct {
		BIRTerminatorBase
	}

	CallSite struct {
		Kind              CallKind
		Args              []BIROperand
		Name              model.Name
		CalleePkg         *model.PackageID
		FunctionLookupKey string
		FpOperand         *BIROperand
		Receiver          *BIROperand
		MethodName        string
		PathSegments      []BIROperand
	}

	Call struct {
		BIRTerminatorBase
		CallSite
		CachedBIRFunc *BIRFunction
		// CachedMethodLookupKey is used only for method calls. It ensures CachedBIRFunc
		// matches the receiver object's resolved method lookup key for this call site.
		CachedMethodLookupKey string
		CachedNativeFunc      extern.NativeFunc
	}

	Return struct {
		BIRTerminatorBase
	}

	Branch struct {
		BIRTerminatorBase
		Op      *BIROperand
		TrueBB  *BIRBasicBlock
		FalseBB *BIRBasicBlock
	}

	Panic struct {
		BIRTerminatorBase
		ErrorOp *BIROperand
	}

	LockStart struct {
		BIRTerminatorBase
		LockKey string
	}

	LockEnd struct {
		BIRTerminatorBase
		LockKey string
	}

	ResourceFunctionCall struct {
		BIRTerminatorBase
		CallSite
	}

	StartAction struct {
		BIRTerminatorBase
		Call       CallSite
		IsIsolated bool
	}

	SingleWaitAction struct {
		BIRTerminatorBase
		Future BIROperand
	}

	AlternateWaitAction struct {
		BIRTerminatorBase
		Futures []BIROperand
	}

	MultipleWaitAction struct {
		BIRTerminatorBase
		Futures    []BIROperand
		FieldNames []string
		Type       semtypes.SemType
	}
)

var (
	_ BIRTerminator        = &Goto{}
	_ BIRAssignInstruction = &Call{}
	_ BIRTerminator        = &Return{}
	_ BIRTerminator        = &Branch{}
	_ BIRTerminator        = &Panic{}
	_ BIRTerminator        = &LockStart{}
	_ BIRTerminator        = &LockEnd{}
	_ BIRAssignInstruction = &ResourceFunctionCall{}
	_ BIRAssignInstruction = &StartAction{}
	_ BIRAssignInstruction = &SingleWaitAction{}
	_ BIRAssignInstruction = &AlternateWaitAction{}
	_ BIRAssignInstruction = &MultipleWaitAction{}
)

func (g *Goto) GetKind() InstructionKind {
	return InstructionKindGoto
}

func NewReturn(pos Location) *Return {
	return &Return{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{
					Pos: pos,
				},
			},
		},
	}
}

func NewGoto(thenBB *BIRBasicBlock, pos Location) *Goto {
	return &Goto{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{
					Pos: pos,
				},
			},
			ThenBB: thenBB,
		},
	}
}

func (c *Call) GetKind() InstructionKind {
	if c.Kind == CallKindFunctionPointer {
		return InstructionKindFPCall
	}
	return InstructionKindCall
}

func (c *Call) GetLhsOperand() *BIROperand {
	return c.LhsOp
}

func NewCall(call CallSite, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *Call {
	return &Call{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{
					Pos: pos,
				},
				LhsOp: lhsOp,
			},
			ThenBB: thenBB,
		},
		CallSite: call,
	}
}

func (r *Return) GetKind() InstructionKind {
	return InstructionKindReturn
}

func (b *Branch) GetKind() InstructionKind {
	return InstructionKindBranch
}

func (p *Panic) GetKind() InstructionKind {
	return InstructionKindPanic
}

func (l *LockStart) GetKind() InstructionKind {
	return InstructionKindLock
}

func (l *LockEnd) GetKind() InstructionKind {
	return InstructionKindUnlock
}

func NewLockStart(key string, thenBB *BIRBasicBlock, pos Location) *LockStart {
	return &LockStart{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{Pos: pos},
			},
			ThenBB: thenBB,
		},
		LockKey: key,
	}
}

func NewLockEnd(key string, thenBB *BIRBasicBlock, pos Location) *LockEnd {
	return &LockEnd{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{Pos: pos},
			},
			ThenBB: thenBB,
		},
		LockKey: key,
	}
}

func NewPanic(errorOp *BIROperand, pos Location) *Panic {
	return &Panic{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{
					Pos: pos,
				},
			},
		},
		ErrorOp: errorOp,
	}
}

func (r *ResourceFunctionCall) GetKind() InstructionKind {
	return InstructionKindResourceCall
}

func (r *ResourceFunctionCall) GetLhsOperand() *BIROperand {
	return r.LhsOp
}

func NewResourceFunctionCall(call CallSite, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *ResourceFunctionCall {
	return &ResourceFunctionCall{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			ThenBB: thenBB,
		},
		CallSite: call,
	}
}

func (s *StartAction) GetKind() InstructionKind   { return InstructionKindAsyncCall }
func (s *StartAction) GetLhsOperand() *BIROperand { return s.LhsOp }

func NewStartAction(call CallSite, isolated bool, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *StartAction {
	return &StartAction{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{BIRNodeBase: BIRNodeBase{Pos: pos}, LhsOp: lhsOp},
			ThenBB:             thenBB,
		},
		Call:       call,
		IsIsolated: isolated,
	}
}

func (s *SingleWaitAction) GetKind() InstructionKind   { return InstructionKindWait }
func (s *SingleWaitAction) GetLhsOperand() *BIROperand { return s.LhsOp }

func NewSingleWaitAction(future BIROperand, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *SingleWaitAction {
	return &SingleWaitAction{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{BIRNodeBase: BIRNodeBase{Pos: pos}, LhsOp: lhsOp},
			ThenBB:             thenBB,
		},
		Future: future,
	}
}

func (a *AlternateWaitAction) GetKind() InstructionKind   { return InstructionKindAlternateWait }
func (a *AlternateWaitAction) GetLhsOperand() *BIROperand { return a.LhsOp }

func NewAlternateWaitAction(futures []BIROperand, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *AlternateWaitAction {
	return &AlternateWaitAction{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{BIRNodeBase: BIRNodeBase{Pos: pos}, LhsOp: lhsOp},
			ThenBB:             thenBB,
		},
		Futures: futures,
	}
}

func (m *MultipleWaitAction) GetKind() InstructionKind   { return InstructionKindWaitAll }
func (m *MultipleWaitAction) GetLhsOperand() *BIROperand { return m.LhsOp }

func NewMultipleWaitAction(futures []BIROperand, fieldNames []string, ty semtypes.SemType, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *MultipleWaitAction {
	return &MultipleWaitAction{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{BIRNodeBase: BIRNodeBase{Pos: pos}, LhsOp: lhsOp},
			ThenBB:             thenBB,
		},
		Futures:    futures,
		FieldNames: fieldNames,
		Type:       ty,
	}
}

func NewBranch(op *BIROperand, trueBB, falseBB *BIRBasicBlock, pos Location) *Branch {
	return &Branch{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{
					Pos: pos,
				},
			},
			ThenBB: trueBB,
		},
		Op:      op,
		TrueBB:  trueBB,
		FalseBB: falseBB,
	}
}
