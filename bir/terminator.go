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
	"ballerina-lang-go/common"
	"ballerina-lang-go/model"
	"ballerina-lang-go/runtime/extern"
)

type BIRTerminator = BIRInstruction

type (
	BIRTerminatorBase struct {
		BIRInstructionBase
		ThenBB *BIRBasicBlock
	}

	Goto struct {
		BIRTerminatorBase
	}

	Call struct {
		BIRTerminatorBase
		Kind              InstructionKind
		IsMethodCall      bool
		Args              []BIROperand
		Name              model.Name
		CalleePkg         *model.PackageID
		CalleeFlags       common.Set[model.Flag]
		FunctionLookupKey string
		CachedBIRFunc     *BIRFunction
		// CachedMethodLookupKey is used only for method calls. It ensures CachedBIRFunc
		// matches the receiver object's resolved method lookup key for this call site.
		CachedMethodLookupKey string
		CachedNativeFunc      extern.NativeFunc
		FpOperand             *BIROperand // For FP_CALL: the operand holding the function value
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
		Receiver     BIROperand
		MethodName   string
		PathSegments []BIROperand
		Args         []BIROperand
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
)

func (g *Goto) GetKind() InstructionKind {
	return INSTRUCTION_KIND_GOTO
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
	return c.Kind
}

func (c *Call) GetLhsOperand() *BIROperand {
	return c.LhsOp
}

func NewCall(kind InstructionKind, args []BIROperand, name model.Name, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *Call {
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
		Kind: kind,
		Args: args,
		Name: name,
	}
}

func (r *Return) GetKind() InstructionKind {
	return INSTRUCTION_KIND_RETURN
}

func (b *Branch) GetKind() InstructionKind {
	return INSTRUCTION_KIND_BRANCH
}

func (p *Panic) GetKind() InstructionKind {
	return INSTRUCTION_KIND_PANIC
}

func (l *LockStart) GetKind() InstructionKind {
	return INSTRUCTION_KIND_LOCK
}

func (l *LockEnd) GetKind() InstructionKind {
	return INSTRUCTION_KIND_UNLOCK
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
	return INSTRUCTION_KIND_RESOURCE_CALL
}

func (r *ResourceFunctionCall) GetLhsOperand() *BIROperand {
	return r.LhsOp
}

func NewResourceFunctionCall(receiver BIROperand, methodName string, pathSegments, args []BIROperand, thenBB *BIRBasicBlock, lhsOp *BIROperand, pos Location) *ResourceFunctionCall {
	return &ResourceFunctionCall{
		BIRTerminatorBase: BIRTerminatorBase{
			BIRInstructionBase: BIRInstructionBase{
				BIRNodeBase: BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			ThenBB: thenBB,
		},
		Receiver:     receiver,
		MethodName:   methodName,
		PathSegments: pathSegments,
		Args:         args,
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
