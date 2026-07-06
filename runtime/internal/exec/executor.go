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

package exec

import (
	"fmt"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/model"
	"ballerina-lang-go/runtime/extern"
	runtimeframe "ballerina-lang-go/runtime/internal/frame"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const maxRecursionDepth = 5000

func executeFunction(ctx *extern.Context, birFunc *bir.BIRFunction, args []values.BalValue, parentFrame *Frame) values.BalValue {
	frame := createFunctionFrame(ctx, birFunc, args, parentFrame)
	bb := &birFunc.BasicBlocks[0]
	if len(birFunc.ErrorTable) > 0 {
		executeFunctionWithTrap(ctx, birFunc, bb, frame)
	} else {
		executeFunctionNoTrap(ctx, bb, frame)
	}
	result := frame.Local(0)
	popFrame(ctx)
	return result
}

func popFrame(ctx *extern.Context) {
	cs := getCallStack(ctx)
	frame := cs.top()
	cs.Pop()
	frame.Free()
}

func pushFrame(ctx *extern.Context, frame *Frame) {
	getCallStack(ctx).Push(frame)
}

func callStackDepth(ctx *extern.Context) int {
	return getCallStack(ctx).len()
}

func getCallStack(ctx *extern.Context) *callStack {
	return ctx.CallStack.(*callStack)
}

func createFunctionFrame(ctx *extern.Context, birFunc *bir.BIRFunction, args []values.BalValue, parentFrame *Frame) *Frame {
	frame := runtimeframe.New(len(birFunc.LocalVars), parentFrame)
	frame.SetFunctionKey(birFunc.FunctionLookupKey)
	initLocalsForFunction(ctx, birFunc, args, frame)
	pushFrame(ctx, frame)
	if callStackDepth(ctx) > maxRecursionDepth {
		panic(values.NewErrorWithMessage("stack overflow"))
	}
	return frame
}

func initLocalsForFunction(ctx *extern.Context, birFunc *bir.BIRFunction, args []values.BalValue, frame *Frame) {
	frame.SetLocal(0, nil)
	localVars := &birFunc.LocalVars
	argOffset := 0
	if birFunc.Flags.Has(model.FlagAttached) {
		frame.SetLocal(1, args[0])
		argOffset = 1
	}
	requiredCount := len(birFunc.RequiredParams)
	for i := range requiredCount {
		frame.SetLocal(i+1+argOffset, args[i+argOffset])
	}

	if birFunc.RestParams != nil {
		restArgs := args[requiredCount+argOffset:]
		restParamIdx := requiredCount + 1 + argOffset
		restParamType := (*localVars)[restParamIdx].GetType()
		atomic := semtypes.ToListAtomicType(ctx.TypeCtx, restParamType)
		if atomic == nil {
			panic("rest parameter type has no list atomic representation")
		}
		initial := make([]values.BalValue, len(restArgs))
		copy(initial, restArgs)
		list := values.NewList(restParamType, atomic, true, nil, len(restArgs), initial)
		frame.SetLocal(restParamIdx, list)
	} else {
		if len(args) > requiredCount+argOffset {
			panic(values.NewErrorWithMessage("too many arguments"))
		}
	}
}

func executeFunctionWithTrap(ctx *extern.Context, birFunc *bir.BIRFunction, bb *bir.BIRBasicBlock, frame *Frame) {
	currentFrame := frame
	for {
		curBBNumber := bb.Number
		nextBB, nextFrame, recovered := executeBasicBlockWithTrap(ctx, bb, frame, currentFrame)

		if recovered != nil {
			// Resolve the innermost error-table entry covering the current block and
			// continue execution at its target with the recovered error value.
			handler := findTrapErrorEntry(birFunc, curBBNumber)
			if handler == nil {
				panic(recovered)
			}
			unwindCallStackToFrame(ctx, frame)
			errVal := panicValueToErrorValue(recovered)
			currentFrame = setRecoveredError(ctx, handler.ErrorOp, nextFrame, errVal)
			bb = &birFunc.BasicBlocks[handler.Target]
			continue
		}

		bb = nextBB
		currentFrame = nextFrame
		if bb == nil {
			break
		}
	}
}

func executeFunctionNoTrap(ctx *extern.Context, bb *bir.BIRBasicBlock, frame *Frame) {
	currentFrame := frame
	for {
		var nextBB *bir.BIRBasicBlock
		nextBB, currentFrame = executeBasicBlock(ctx, bb, frame, currentFrame)
		bb = nextBB
		if bb == nil {
			break
		}
	}
}

func executeBasicBlockWithTrap(ctx *extern.Context, bb *bir.BIRBasicBlock, frame *Frame, currentFrame *Frame) (nextBB *bir.BIRBasicBlock, nextFrame *Frame, recovered any) {
	defer func() {
		if r := recover(); r != nil {
			nextFrame = currentFrame
			recovered = r
		}
	}()
	for _, inst := range bb.Instructions {
		getCallStack(ctx).SetCurrentLocation(inst.GetPos())
		currentFrame = execInstruction(ctx, inst, currentFrame)
	}
	getCallStack(ctx).SetCurrentLocation(bb.Terminator.GetPos())
	return execTerminator(ctx, bb.Terminator, currentFrame), currentFrame, nil
}

func executeBasicBlock(ctx *extern.Context, bb *bir.BIRBasicBlock, frame *Frame, currentFrame *Frame) (*bir.BIRBasicBlock, *Frame) {
	for _, inst := range bb.Instructions {
		getCallStack(ctx).SetCurrentLocation(inst.GetPos())
		currentFrame = execInstruction(ctx, inst, currentFrame)
	}
	getCallStack(ctx).SetCurrentLocation(bb.Terminator.GetPos())
	return execTerminator(ctx, bb.Terminator, currentFrame), currentFrame
}

func execInstruction(ctx *extern.Context, inst bir.BIRNonTerminator, frame *Frame) *Frame {
	switch v := inst.(type) {
	case *bir.PushScopeFrame:
		return runtimeframe.New(v.NumLocals, frame)
	case *bir.PopScopeFrame:
		parent := frame.Parent()
		frame.Free()
		return parent
	case *bir.ConstantLoad:
		execConstantLoad(ctx, v, frame)
	case *bir.Move:
		execMove(ctx, v, frame)
	case *bir.NewArray:
		execNewArray(ctx, v, frame)
	case *bir.NewMap:
		execNewMap(ctx, v, frame)
	case *bir.NewError:
		execNewError(ctx, v, frame)
	case *bir.NewObject:
		execNewObject(ctx, v, frame)
	case *bir.NewStream:
		execNewStream(ctx, v, frame)
	case *bir.StreamNext:
		execStreamNext(ctx, v, frame)
	case *bir.StreamClose:
		execStreamClose(ctx, v, frame)
	case *bir.FieldAccess:
		switch v.GetKind() {
		case bir.INSTRUCTION_KIND_ARRAY_STORE:
			execArrayStore(ctx, v, frame)
		case bir.INSTRUCTION_KIND_ARRAY_LOAD:
			execArrayLoad(ctx, v, frame)
		case bir.INSTRUCTION_KIND_ARRAY_FILLING_LOAD:
			execArrayFillingLoad(ctx, v, frame)
		case bir.INSTRUCTION_KIND_MAP_STORE:
			execMapStore(ctx, v, frame)
		case bir.INSTRUCTION_KIND_MAP_FILLING_LOAD:
			execMapFillingLoad(ctx, v, frame)
		case bir.INSTRUCTION_KIND_MAP_LOAD:
			execMapLoad(ctx, v, frame)
		case bir.INSTRUCTION_KIND_OBJECT_STORE:
			execObjectStore(ctx, v, frame)
		case bir.INSTRUCTION_KIND_OBJECT_LOAD:
			execObjectLoad(ctx, v, frame)
		default:
			fmt.Printf("UNKNOWN_FIELD_ACCESS_KIND(%d)\n", v.GetKind())
		}
	case *bir.BinaryOp:
		switch v.GetKind() {
		case bir.INSTRUCTION_KIND_ADD:
			execBinaryOpAdd(ctx, v, frame)
		case bir.INSTRUCTION_KIND_SUB:
			execBinaryOpSub(ctx, v, frame)
		case bir.INSTRUCTION_KIND_MUL:
			execBinaryOpMul(ctx, v, frame)
		case bir.INSTRUCTION_KIND_DIV:
			execBinaryOpDiv(ctx, v, frame)
		case bir.INSTRUCTION_KIND_MOD:
			execBinaryOpMod(ctx, v, frame)
		case bir.INSTRUCTION_KIND_EQUAL:
			execBinaryOpEqual(ctx, v, frame)
		case bir.INSTRUCTION_KIND_NOT_EQUAL:
			execBinaryOpNotEqual(ctx, v, frame)
		case bir.INSTRUCTION_KIND_GREATER_THAN:
			execBinaryOpGT(ctx, v, frame)
		case bir.INSTRUCTION_KIND_GREATER_EQUAL:
			execBinaryOpGTE(ctx, v, frame)
		case bir.INSTRUCTION_KIND_LESS_THAN:
			execBinaryOpLT(ctx, v, frame)
		case bir.INSTRUCTION_KIND_LESS_EQUAL:
			execBinaryOpLTE(ctx, v, frame)
		case bir.INSTRUCTION_KIND_AND:
			execBinaryOpAnd(ctx, v, frame)
		case bir.INSTRUCTION_KIND_OR:
			execBinaryOpOr(ctx, v, frame)
		case bir.INSTRUCTION_KIND_REF_EQUAL:
			execBinaryOpRefEqual(ctx, v, frame)
		case bir.INSTRUCTION_KIND_REF_NOT_EQUAL:
			execBinaryOpRefNotEqual(ctx, v, frame)
		case bir.INSTRUCTION_KIND_CLOSED_RANGE:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_CLOSED_RANGE")
		case bir.INSTRUCTION_KIND_HALF_OPEN_RANGE:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_HALF_OPEN_RANGE")
		case bir.INSTRUCTION_KIND_ANNOT_ACCESS:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_ANNOT_ACCESS")
		case bir.INSTRUCTION_KIND_BITWISE_AND:
			execBinaryOpBitwiseAnd(ctx, v, frame)
		case bir.INSTRUCTION_KIND_BITWISE_OR:
			execBinaryOpBitwiseOr(ctx, v, frame)
		case bir.INSTRUCTION_KIND_BITWISE_XOR:
			execBinaryOpBitwiseXor(ctx, v, frame)
		case bir.INSTRUCTION_KIND_BITWISE_LEFT_SHIFT:
			execBinaryOpBitwiseLeftShift(ctx, v, frame)
		case bir.INSTRUCTION_KIND_BITWISE_RIGHT_SHIFT:
			execBinaryOpBitwiseRightShift(ctx, v, frame)
		case bir.INSTRUCTION_KIND_BITWISE_UNSIGNED_RIGHT_SHIFT:
			execBinaryOpBitwiseUnsignedRightShift(ctx, v, frame)
		default:
			fmt.Printf("UNKNOWN_BINARY_INSTRUCTION_KIND(%d)\n", v.GetKind())
		}
	case *bir.UnaryOp:
		switch v.GetKind() {
		case bir.INSTRUCTION_KIND_NOT:
			execUnaryOpNot(ctx, v, frame)
		case bir.INSTRUCTION_KIND_NEGATE:
			execUnaryOpNegate(ctx, v, frame)
		case bir.INSTRUCTION_KIND_BITWISE_COMPLEMENT:
			execUnaryOpBitwiseComplement(ctx, v, frame)
		case bir.INSTRUCTION_KIND_TYPEOF:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_TYPEOF")
		default:
			fmt.Printf("UNKNOWN_UNARY_INSTRUCTION_KIND(%d)\n", v.GetKind())
		}
	case *bir.TypeCast:
		execTypeCast(ctx, v, frame)
	case *bir.TypeTest:
		execTypeTest(ctx, v, frame)
	case *bir.FPLoad:
		execFPLoad(ctx, v, frame)
	case *bir.NewXMLElement:
		execNewXMLElement(ctx, v, frame)
	case *bir.NewXMLPI:
		execNewXMLPI(ctx, v, frame)
	case *bir.NewXMLComment:
		execNewXMLComment(ctx, v, frame)
	case *bir.NewXMLText:
		execNewXMLText(ctx, v, frame)
	case *bir.NewXMLSequence:
		execNewXMLSequence(ctx, v, frame)
	case *bir.EvalTemplateExpr:
		execEvalTemplateExpr(ctx, v, frame)
	default:
		fmt.Printf("UNKNOWN_INSTRUCTION_TYPE(%T)\n", inst)
	}
	return frame
}

func execTerminator(ctx *extern.Context, term bir.BIRTerminator, frame *Frame) *bir.BIRBasicBlock {
	switch v := term.(type) {
	case *bir.Goto:
		return v.ThenBB
	case *bir.Branch:
		return execBranch(ctx, v, frame)
	case *bir.Panic:
		return execPanic(ctx, v, frame)
	case *bir.Call:
		switch v.GetKind() {
		case bir.INSTRUCTION_KIND_CALL:
			return execCall(ctx, v, frame)
		case bir.INSTRUCTION_KIND_ASYNC_CALL:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_ASYNC_CALL")
		case bir.INSTRUCTION_KIND_WAIT:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_WAIT")
		case bir.INSTRUCTION_KIND_FP_CALL:
			return execFpCall(ctx, v, frame)
		case bir.INSTRUCTION_KIND_WK_RECEIVE:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_WK_RECEIVE")
		case bir.INSTRUCTION_KIND_WK_SEND:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_WK_SEND")
		case bir.INSTRUCTION_KIND_FLUSH:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_FLUSH")
		case bir.INSTRUCTION_KIND_LOCK:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_LOCK")
		case bir.INSTRUCTION_KIND_FIELD_LOCK:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_FIELD_LOCK")
		case bir.INSTRUCTION_KIND_UNLOCK:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_UNLOCK")
		case bir.INSTRUCTION_KIND_WAIT_ALL:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_WAIT_ALL")
		case bir.INSTRUCTION_KIND_WK_ALT_RECEIVE:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_WK_ALT_RECEIVE")
		case bir.INSTRUCTION_KIND_WK_MULTIPLE_RECEIVE:
			fmt.Println("NOT IMPLEMENTED: INSTRUCTION_KIND_WK_MULTIPLE_RECEIVE")
		default:
			fmt.Printf("UNKNOWN_CALL_INSTRUCTION_KIND(%d)\n", v.GetKind())
		}
	case *bir.Return:
		return nil
	case *bir.LockStart:
		ctx.AcquireLock(v.LockKey)
		return v.ThenBB
	case *bir.LockEnd:
		ctx.ReleaseLock()
		return v.ThenBB
	case *bir.ResourceFunctionCall:
		return execResourceCall(ctx, v, frame)
	default:
		fmt.Printf("UNKNOWN_TERMINATOR_TYPE(%T)\n", term)
	}
	return nil
}

func panicValueToErrorValue(r any) values.BalValue {
	// `trap` expects runtime failures to be raised as `*values.Error`.
	// If this isn't the case, treat it as an unrecoverable interpreter issue.
	if err, ok := r.(*values.Error); ok {
		return err
	}
	panic(r)
}

func setRecoveredError(ctx *extern.Context, op *bir.BIROperand, currentFrame *Frame, errVal values.BalValue) *Frame {
	if gv, ok := op.VariableDcl.(*bir.BIRGlobalVariableDcl); ok {
		module := getModule(ctx, gv.PkgId)
		module.Globals[gv.GlobalVarLookupKey] = errVal
		return currentFrame
	}
	targetFrame := resolveFrame(currentFrame, op.Address)
	unwindScopeFramesToFrame(currentFrame, targetFrame)
	targetFrame.SetLocal(op.Address.FrameIndex, errVal)
	return targetFrame
}

func findTrapErrorEntry(birFunc *bir.BIRFunction, bbNumber int) *bir.BIRErrorEntry {
	var best *bir.BIRErrorEntry
	var bestSpan int
	found := false
	for i := range birFunc.ErrorTable {
		entry := &birFunc.ErrorTable[i]
		start := entry.Start
		end := entry.End
		if bbNumber < start || bbNumber > end {
			continue
		}
		span := end - start
		// Prefer the narrowest enclosing range, i.e. nearest (innermost) trap.
		if !found || span < bestSpan {
			best = entry
			bestSpan = span
			found = true
		}
	}
	return best
}

func unwindCallStackToFrame(ctx *extern.Context, frame *Frame) {
	for callStackDepth(ctx) > 0 && getCallStack(ctx).top() != frame {
		popFrame(ctx)
	}
}

func unwindScopeFramesToFrame(currentFrame *Frame, targetFrame *Frame) {
	for currentFrame != nil && currentFrame != targetFrame {
		parent := currentFrame.Parent()
		currentFrame.Free()
		currentFrame = parent
	}
}
