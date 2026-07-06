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

package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/context"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/desugar"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

type birReader struct {
	r   *bytes.Reader
	cp  []any
	tp  *semtypes.TypePool
	ctx *context.CompilerContext
}

func Unmarshal(ctx *context.CompilerContext, data []byte) (*bir.BIRPackage, error) {
	reader := &birReader{
		r:   bytes.NewReader(data),
		ctx: ctx,
	}
	return reader.readPackage()
}

func (br *birReader) readPackage() (pkg *bir.BIRPackage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("BIR deserializer failed: %v", r)
		}
	}()

	magic := make([]byte, 4)
	br.read(magic)

	if string(magic) != BIR_MAGIC {
		panic(fmt.Sprintf("invalid BIR magic: %x", magic))
	}

	var version int32
	br.read(&version)

	if version != BIR_VERSION {
		panic(fmt.Sprintf("unsupported BIR version: %d", version))
	}

	br.readTypePool()
	br.readConstantPool()

	var pkgIdx int32
	br.read(&pkgIdx)

	pkgID := br.getPackageFromCP(int(pkgIdx))
	globalVars := br.readGlobalVars(pkgID)
	classDefs := br.readClassDefs()

	var initFunction *bir.BIRFunction
	var hasInit bool
	br.read(&hasInit)
	if hasInit {
		initFunction = br.readFunction()
	}

	var mainFunction *bir.BIRFunction
	var hasMain bool
	br.read(&hasMain)
	if hasMain {
		mainFunction = br.readFunction()
	}

	functions := br.readFunctions()

	pkg = &bir.BIRPackage{
		PackageID:    pkgID,
		GlobalVars:   globalVars,
		ClassDefs:    classDefs,
		Functions:    functions,
		InitFunction: initFunction,
		MainFunction: mainFunction,
	}
	rebindLifecycleFunctions(pkg)
	return pkg, nil
}

// rebindLifecycleFunctions restores StartFunction/GracefulStopFunction/
// ImmediateStopFunction pointers on a deserialized BIR package. These are
// generated as regular functions named `$start`, `$gracefulStop` and
// `$immediateStop` and not serialized as dedicated slots.
func rebindLifecycleFunctions(pkg *bir.BIRPackage) {
	for i := range pkg.Functions {
		fn := &pkg.Functions[i]
		switch fn.Name.Value() {
		case desugar.StartFunctionName:
			pkg.StartFunction = fn
		case desugar.GracefulStopFunctionName:
			pkg.GracefulStopFunction = fn
		case desugar.ImmediateStopFunctionName:
			pkg.ImmediateStopFunction = fn
		}
	}
}

func (br *birReader) readTypePool() {
	var tpSize int64
	br.read(&tpSize)
	tpBytes := make([]byte, tpSize)
	_, err := br.r.Read(tpBytes)
	if err != nil {
		panic(fmt.Sprintf("reading type pool bytes: %v", err))
	}
	br.tp = semtypes.UnmarshalTypePool(tpBytes, br.ctx.GetTypeEnv())
}

func (br *birReader) readType() semtypes.SemType {
	var idx int32
	br.read(&idx)
	if idx == -1 {
		return semtypes.SemType{}
	}
	return br.tp.Get(semtypes.TypePoolIndex(idx))
}

func (br *birReader) readConstantPool() {
	var cpSize int64
	br.read(&cpSize)

	br.cp = make([]any, cpSize)

	for i := 0; i < int(cpSize); i++ {
		var tag int8
		br.read(&tag)
		br.readConstantPoolEntry(tag, i)
	}
}

func (br *birReader) readConstantPoolEntry(tag int8, i int) {
	switch tag {
	case 0:
		br.cp[i] = nil
	case 1:
		var length int64
		br.read(&length)

		if length < 0 {
			br.cp[i] = (*string)(nil)
		} else {
			strBytes := make([]byte, length)
			br.read(strBytes)

			br.cp[i] = string(strBytes)
		}
	case 2:
		var orgIdx int32
		br.read(&orgIdx)

		var pkgNameIdx int32
		br.read(&pkgNameIdx)

		var moduleNameIdx int32
		br.read(&moduleNameIdx)

		var versionIdx int32
		br.read(&versionIdx)

		org := model.Name(br.getStringFromCP(int(orgIdx)))
		pkgName := model.Name(br.getStringFromCP(int(pkgNameIdx)))
		_ = br.getStringFromCP(int(moduleNameIdx))
		version := model.Name(br.getStringFromCP(int(versionIdx)))
		nameComps := model.CreateNameComps(pkgName)
		br.cp[i] = br.ctx.NewPackageID(org, nameComps, version)
	case 3:
		panic("shape not implemented")
	default:
		panic(fmt.Sprintf("unknown CP tag: %d", tag))
	}
}

func (br *birReader) getFromCP(index int) any {
	if index < 0 || index >= len(br.cp) {
		return nil
	}
	return br.cp[index]
}

func (br *birReader) getStringFromCP(index int) string {
	v := br.getFromCP(index)
	return v.(string)
}

func (br *birReader) getPackageFromCP(index int) *model.PackageID {
	v := br.getFromCP(index)
	return v.(*model.PackageID)
}

func (br *birReader) readGlobalVars(pkgID *model.PackageID) map[string]bir.BIRGlobalVariableDcl {
	count := br.readLength()
	variables := make(map[string]bir.BIRGlobalVariableDcl, count)
	for i := 0; i < int(count); i++ {
		pos := br.readPosition()
		_ = br.readKind() // kind (ignored, concrete type determines it)
		name := br.readStringCPEntry()
		flags := br.readFlags()

		ty := br.readType()

		lookupKey := pkgID.OrgName.Value() + "/" + pkgID.PkgName.Value() + ":" + name.Value()
		gv := bir.BIRGlobalVariableDcl{
			Flags:              flags,
			GlobalVarLookupKey: lookupKey,
		}
		gv.Pos = pos
		gv.Name = name
		gv.Type = ty
		gv.PkgId = pkgID

		variables[lookupKey] = gv
	}
	return variables
}

func (br *birReader) readClassDefs() []bir.BIRClassDef {
	count := br.readLength()
	classDefs := make([]bir.BIRClassDef, count)
	for i := 0; i < int(count); i++ {
		br.readClassDef(&classDefs[i])
	}
	return classDefs
}

func (br *birReader) readClassDef(classDef *bir.BIRClassDef) {
	name := br.readStringCPEntry()
	classDef.Name = name
	lookupKey := br.readStringCPEntry()
	classDef.LookupKey = lookupKey.Value()

	fieldCount := br.readLength()
	fields := make([]bir.ObjectField, fieldCount)
	for i := 0; i < int(fieldCount); i++ {
		fieldName := br.readStringCPEntry()
		fieldType := br.readType()
		fields[i] = bir.ObjectField{
			Name: fieldName.Value(),
			Ty:   fieldType,
		}
	}
	classDef.Fields = fields

	methodCount := br.readLength()
	vTable := make(map[string]*bir.BIRFunction, methodCount)
	for i := 0; i < int(methodCount); i++ {
		methodName := br.readStringCPEntry()
		fn := br.readFunction()
		vTable[methodName.Value()] = fn
	}
	classDef.VTable = vTable

	rmCount := br.readLength()
	rTable := make(map[string][]bir.BIRResourceMethod, rmCount)
	for i := 0; i < int(rmCount); i++ {
		methodNameN := br.readStringCPEntry()
		methodName := methodNameN.Value()
		entryCount := br.readLength()
		entries := make([]bir.BIRResourceMethod, entryCount)
		for j := 0; j < int(entryCount); j++ {
			segCount := br.readLength()
			segs := make([]bir.ResourcePathSegmentDef, segCount)
			for k := 0; k < int(segCount); k++ {
				segs[k] = bir.ResourcePathSegmentDef{Ty: br.readType()}
			}
			restTy := br.readType()
			if semtypes.IsZero(restTy) {
				restTy = semtypes.NEVER
			}
			fn := br.readFunction()
			entries[j] = bir.BIRResourceMethod{
				PathSegments:  segs,
				RestSegmentTy: restTy,
				Fn:            fn,
			}
		}
		rTable[methodName] = entries
	}
	classDef.RTable = rTable
}

func (br *birReader) readFunctions() []bir.BIRFunction {
	count := br.readLength()
	functions := make([]bir.BIRFunction, count)
	for i := 0; i < int(count); i++ {
		fn := br.readFunction()
		functions[i] = *fn
	}
	return functions
}

func (br *birReader) readFunction() *bir.BIRFunction {
	pos := br.readPosition()
	name := br.readStringCPEntry()
	originalName := br.readStringCPEntry()
	flag := br.readFlags()
	functionLookupKey := br.readStringCPEntry()
	requiredParamsCount := br.readLength()

	requiredParams := make([]bir.BIRParameter, requiredParamsCount)
	for j := 0; j < int(requiredParamsCount); j++ {
		paramName := br.readStringCPEntry()
		paramFlags := br.readFlags()

		requiredParams[j] = bir.BIRParameter{
			Name:  paramName,
			Flags: paramFlags,
		}
	}

	var hasRestParam bool
	br.read(&hasRestParam)

	_ = br.readLength() // Unused?

	argsCount := br.readLength()

	varMap := make(map[string]bir.BIRVariableDcl)
	bbMap := make(map[string]*bir.BIRBasicBlock)

	var hasReturnVar bool
	br.read(&hasReturnVar)

	var returnVar *bir.BIRLocalVariableDcl
	if hasReturnVar {
		_ = br.readKind()
		returnVarType := br.readType()
		returnVarName := br.readStringCPEntry()

		returnVar = &bir.BIRLocalVariableDcl{}
		returnVar.Name = returnVarName
		returnVar.Type = returnVarType
		varMap[returnVarName.Value()] = returnVar
	}

	localVarCount := br.readLength()
	localVars := make([]bir.BIRLocalVariableDcl, localVarCount)
	for j := 0; j < int(localVarCount); j++ {
		localVar := br.readLocalVar(varMap)
		localVars[j] = *localVar
	}

	basicBlockCount := br.readLength()
	basicBlocks := make([]bir.BIRBasicBlock, basicBlockCount)
	for j := 0; j < int(basicBlockCount); j++ {
		block := br.readBasicBlock(varMap)
		block.Number = j
		basicBlocks[j] = *block
		bbMap[block.Id.Value()] = &basicBlocks[j]
	}

	for j := range basicBlocks {
		bb := &basicBlocks[j]
		if bb.Terminator != nil {
			switch t := bb.Terminator.(type) {
			case *bir.Goto:
				if target, ok := bbMap[t.ThenBB.Id.Value()]; ok {
					t.ThenBB = target
				}
			case *bir.Branch:
				if target, ok := bbMap[t.TrueBB.Id.Value()]; ok {
					t.TrueBB = target
				}
				if target, ok := bbMap[t.FalseBB.Id.Value()]; ok {
					t.FalseBB = target
				}
			case *bir.Call:
				if target, ok := bbMap[t.ThenBB.Id.Value()]; ok {
					t.ThenBB = target
				}
			case *bir.Panic:
				// Panic has no ThenBB
			case *bir.LockStart:
				if target, ok := bbMap[t.ThenBB.Id.Value()]; ok {
					t.ThenBB = target
				}
			case *bir.LockEnd:
				if target, ok := bbMap[t.ThenBB.Id.Value()]; ok {
					t.ThenBB = target
				}
			case *bir.ResourceFunctionCall:
				if target, ok := bbMap[t.ThenBB.Id.Value()]; ok {
					t.ThenBB = target
				}
			}
		}
	}

	errorTableCount := br.readLength()
	errorTable := make([]bir.BIRErrorEntry, errorTableCount)
	for j := 0; j < int(errorTableCount); j++ {
		startId := br.readStringCPEntry()
		endId := br.readStringCPEntry()
		targetId := br.readStringCPEntry()
		errorOp := br.readOperand(varMap)
		errorTable[j] = bir.BIRErrorEntry{
			Start:   bbMap[startId.Value()].Number,
			End:     bbMap[endId.Value()].Number,
			Target:  bbMap[targetId.Value()].Number,
			ErrorOp: errorOp,
		}
	}

	var restParams *bir.BIRParameter
	if hasRestParam {
		paramStart := 1
		if len(localVars) > 1 && localVars[1].GetName() == "self" {
			paramStart = 2
		}
		restIdx := paramStart + len(requiredParams)
		restParams = &bir.BIRParameter{Name: localVars[restIdx].GetName()}
	}

	return &bir.BIRFunction{
		BIRNodeBase: bir.BIRNodeBase{
			Pos: pos,
		},
		Name:              name,
		OriginalName:      originalName,
		Flags:             flag,
		FunctionLookupKey: string(functionLookupKey),
		RequiredParams:    requiredParams,
		RestParams:        restParams,
		ArgsCount:         int(argsCount),
		ReturnVariable:    returnVar,
		LocalVars:         localVars,
		BasicBlocks:       basicBlocks,
		ErrorTable:        errorTable,
	}
}

func (br *birReader) readLocalVar(varMap map[string]bir.BIRVariableDcl) *bir.BIRLocalVariableDcl {
	_ = br.readKind()
	ty := br.readType()
	name := br.readStringCPEntry()

	localVar := &bir.BIRLocalVariableDcl{}
	localVar.Name = name
	localVar.Type = ty

	varMap[name.Value()] = localVar
	return localVar
}

func (br *birReader) readBasicBlock(varMap map[string]bir.BIRVariableDcl) *bir.BIRBasicBlock {
	id := br.readStringCPEntry()
	instructionCount := br.readLength()

	instructions := make([]bir.BIRInstruction, instructionCount)
	for k := 0; k < int(instructionCount); k++ {
		ins := br.readInstruction(varMap)
		instructions[k] = ins
	}

	term := br.readTerminator(varMap)

	return &bir.BIRBasicBlock{
		Id:           id,
		Instructions: instructions,
		Terminator:   term,
	}
}

func (br *birReader) readInstruction(varMap map[string]bir.BIRVariableDcl) bir.BIRInstruction {
	instructionKind := br.readInstructionKind()
	pos := br.readPosition()

	switch instructionKind {
	case bir.INSTRUCTION_KIND_MOVE:
		rhsOp := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		return &bir.Move{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			RhsOp: rhsOp,
		}
	case bir.INSTRUCTION_KIND_ADD, bir.INSTRUCTION_KIND_SUB, bir.INSTRUCTION_KIND_MUL,
		bir.INSTRUCTION_KIND_DIV, bir.INSTRUCTION_KIND_MOD, bir.INSTRUCTION_KIND_EQUAL,
		bir.INSTRUCTION_KIND_NOT_EQUAL, bir.INSTRUCTION_KIND_GREATER_THAN,
		bir.INSTRUCTION_KIND_GREATER_EQUAL, bir.INSTRUCTION_KIND_LESS_THAN,
		bir.INSTRUCTION_KIND_LESS_EQUAL, bir.INSTRUCTION_KIND_AND, bir.INSTRUCTION_KIND_OR,
		bir.INSTRUCTION_KIND_REF_EQUAL, bir.INSTRUCTION_KIND_REF_NOT_EQUAL,
		bir.INSTRUCTION_KIND_CLOSED_RANGE, bir.INSTRUCTION_KIND_HALF_OPEN_RANGE,
		bir.INSTRUCTION_KIND_ANNOT_ACCESS, bir.INSTRUCTION_KIND_BITWISE_AND,
		bir.INSTRUCTION_KIND_BITWISE_OR, bir.INSTRUCTION_KIND_BITWISE_XOR,
		bir.INSTRUCTION_KIND_BITWISE_LEFT_SHIFT, bir.INSTRUCTION_KIND_BITWISE_RIGHT_SHIFT,
		bir.INSTRUCTION_KIND_BITWISE_UNSIGNED_RIGHT_SHIFT:
		rhsOp1 := br.readOperand(varMap)
		rhsOp2 := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		return &bir.BinaryOp{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Kind:   instructionKind,
			RhsOp1: *rhsOp1,
			RhsOp2: *rhsOp2,
		}
	case bir.INSTRUCTION_KIND_TYPEOF, bir.INSTRUCTION_KIND_NOT, bir.INSTRUCTION_KIND_NEGATE,
		bir.INSTRUCTION_KIND_BITWISE_COMPLEMENT:
		rhsOp := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		return &bir.UnaryOp{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Kind:  instructionKind,
			RhsOp: rhsOp,
		}
	case bir.INSTRUCTION_KIND_CONST_LOAD:
		// Const load type placeholder (not used — type inferred from value)
		var constLoadTypeIdx int32
		br.read(&constLoadTypeIdx)

		lhsOp := br.readOperand(varMap)

		var isWrapped bool
		br.read(&isWrapped)

		var tagByte int8
		br.read(&tagByte)

		tag := typeTag(tagByte)
		value := br.readConstValueByTag(tag)

		if isWrapped {
			value = bir.ConstValue{
				Value: value,
			}
		}

		return &bir.ConstantLoad{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Value: value,
		}
	case bir.INSTRUCTION_KIND_MAP_STORE, bir.INSTRUCTION_KIND_MAP_LOAD,
		bir.INSTRUCTION_KIND_ARRAY_STORE, bir.INSTRUCTION_KIND_ARRAY_LOAD,
		bir.INSTRUCTION_KIND_ARRAY_FILLING_LOAD,
		bir.INSTRUCTION_KIND_MAP_FILLING_LOAD,
		bir.INSTRUCTION_KIND_OBJECT_STORE, bir.INSTRUCTION_KIND_OBJECT_LOAD:
		lhsOp := br.readOperand(varMap)
		keyOp := br.readOperand(varMap)
		rhsOp := br.readOperand(varMap)
		var filler values.FillerFactory
		if instructionKind == bir.INSTRUCTION_KIND_MAP_FILLING_LOAD && lhsOp != nil && lhsOp.VariableDcl != nil {
			// After filling, the loaded value is guaranteed non-nil, so strip NIL
			// from the operand type before looking up the filler factory.
			tyCx := semtypes.TypeCheckContext(br.ctx.GetTypeEnv())
			valueType := semtypes.Diff(lhsOp.VariableDcl.GetType(), semtypes.NIL)
			filler, _ = values.FillerFactoryFor(tyCx, valueType)
		}
		return &bir.FieldAccess{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Kind:   instructionKind,
			KeyOp:  keyOp,
			RhsOp:  rhsOp,
			Filler: filler,
		}
	case bir.INSTRUCTION_KIND_NEW_ARRAY:
		ty := br.readType()
		lhsOp := br.readOperand(varMap)
		sizeOp := br.readOperand(varMap)
		var isReadonly bool
		br.read(&isReadonly)
		valuesCount := br.readLength()
		arrayValues := make([]*bir.BIROperand, valuesCount)
		for k := 0; k < int(valuesCount); k++ {
			arrayValues[k] = br.readOperand(varMap)
		}
		return &bir.NewArray{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Type:       ty,
			SizeOp:     sizeOp,
			Values:     arrayValues,
			Filler:     br.restFillerFactoryForListType(ty),
			IsReadonly: isReadonly,
		}
	case bir.INSTRUCTION_KIND_TYPE_CAST:
		lhsOp := br.readOperand(varMap)
		rhsOp := br.readOperand(varMap)
		ty := br.readType()

		return &bir.TypeCast{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			RhsOp: rhsOp,
			Type:  ty,
		}
	case bir.INSTRUCTION_KIND_TYPE_TEST:
		rhsOp := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		ty := br.readType()
		var isNegation bool
		br.read(&isNegation)
		return &bir.TypeTest{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			RhsOp:      rhsOp,
			Type:       ty,
			IsNegation: isNegation,
		}
	case bir.INSTRUCTION_KIND_NEW_STRUCTURE:
		ty := br.readType()
		lhsOp := br.readOperand(varMap)
		var isReadonly bool
		br.read(&isReadonly)
		valuesCount := br.readLength()
		values := make([]bir.MappingConstructorEntry, valuesCount)
		for k := 0; k < int(valuesCount); k++ {
			var isKeyValuePair bool
			br.read(&isKeyValuePair)
			if !isKeyValuePair {
				panic("spread entries in mapping constructors are not supported")
			}
			keyOp := br.readOperand(varMap)
			valueOp := br.readOperand(varMap)
			values[k] = bir.NewMappingConstructorKeyValueEntry(keyOp, valueOp)
		}
		defaultsCount := br.readLength()
		defaults := make([]bir.MappingConstructorDefaultEntry, defaultsCount)
		for k := 0; k < int(defaultsCount); k++ {
			defaults[k] = bir.MappingConstructorDefaultEntry{
				FieldName:         string(br.readStringCPEntry()),
				FunctionLookupKey: string(br.readStringCPEntry()),
			}
		}
		return &bir.NewMap{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Type:       ty,
			Values:     values,
			Defaults:   defaults,
			IsReadonly: isReadonly,
		}
	case bir.INSTRUCTION_KIND_NEW_ERROR:
		ty := br.readType()
		lhsOp := br.readOperand(varMap)
		typeName := br.readStringCPEntry()
		messageOp := br.readOperand(varMap)
		var hasCauseOp bool
		br.read(&hasCauseOp)
		var causeOp *bir.BIROperand
		if hasCauseOp {
			causeOp = br.readOperand(varMap)
		}
		var hasDetailOp bool
		br.read(&hasDetailOp)
		var detailOp *bir.BIROperand
		if hasDetailOp {
			detailOp = br.readOperand(varMap)
		}
		return &bir.NewError{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			Type:      ty,
			TypeName:  string(typeName),
			MessageOp: messageOp,
			CauseOp:   causeOp,
			DetailOp:  detailOp,
		}
	case bir.INSTRUCTION_KIND_NEW_INSTANCE:
		classDefRef := br.readStringCPEntry()
		lhsOp := br.readOperand(varMap)
		return &bir.NewObject{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				LhsOp:       lhsOp,
			},
			ClassDefRef: classDefRef.Value(),
		}
	case bir.INSTRUCTION_KIND_NEW_STREAM:
		streamTy := br.readType()
		lhsOp := br.readOperand(varMap)
		implOp := br.readOperand(varMap)
		return bir.NewStreamConstructor(streamTy, lhsOp, implOp, pos)
	case bir.INSTRUCTION_KIND_STREAM_NEXT:
		lhsOp := br.readOperand(varMap)
		streamOp := br.readOperand(varMap)
		return bir.NewStreamNext(lhsOp, streamOp, pos)
	case bir.INSTRUCTION_KIND_STREAM_CLOSE:
		lhsOp := br.readOperand(varMap)
		streamOp := br.readOperand(varMap)
		return bir.NewStreamClose(lhsOp, streamOp, pos)
	case bir.INSTRUCTION_KIND_FP_LOAD:
		functionLookupKey := br.readStringCPEntry()
		ty := br.readType()
		lhsOp := br.readOperand(varMap)
		var isClosure bool
		br.read(&isClosure)
		fpLoad := bir.NewFPLoad(string(functionLookupKey), ty, lhsOp, pos)
		fpLoad.IsClosure = isClosure
		return fpLoad
	case bir.INSTRUCTION_KIND_PUSH_SCOPE:
		var numLocals int32
		br.read(&numLocals)
		return &bir.PushScopeFrame{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
			},
			NumLocals: int(numLocals),
		}
	case bir.INSTRUCTION_KIND_POP_SCOPE:
		return &bir.PopScopeFrame{
			BIRInstructionBase: bir.BIRInstructionBase{
				BIRNodeBase: bir.BIRNodeBase{Pos: pos},
			},
		}
	case bir.INSTRUCTION_KIND_NEW_XML_ELEMENT:
		nameOp := br.readOperand(varMap)
		var hasChildren bool
		br.read(&hasChildren)
		var childrenOp *bir.BIROperand
		if hasChildren {
			childrenOp = br.readOperand(varMap)
		}
		var attrsOp *bir.BIROperand
		var hasAttrs bool
		br.read(&hasAttrs)
		if hasAttrs {
			attrsOp = br.readOperand(varMap)
		}
		var namespacesOp *bir.BIROperand
		var hasNamespaces bool
		br.read(&hasNamespaces)
		if hasNamespaces {
			namespacesOp = br.readOperand(varMap)
		}
		lhsOp := br.readOperand(varMap)
		return bir.NewXMLElementInstr(lhsOp, nameOp, childrenOp, attrsOp, namespacesOp, pos)
	case bir.INSTRUCTION_KIND_NEW_XML_PI:
		targetOp := br.readOperand(varMap)
		dataOp := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		return bir.NewXMLPIInstr(lhsOp, targetOp, dataOp, pos)
	case bir.INSTRUCTION_KIND_NEW_XML_COMMENT:
		bodyOp := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		return bir.NewXMLCommentInstr(lhsOp, bodyOp, pos)
	case bir.INSTRUCTION_KIND_NEW_XML_TEXT:
		bodyOp := br.readOperand(varMap)
		lhsOp := br.readOperand(varMap)
		return bir.NewXMLTextInstr(lhsOp, bodyOp, pos)
	case bir.INSTRUCTION_KIND_NEW_XML_SEQUENCE:
		count := br.readLength()
		children := make([]*bir.BIROperand, count)
		for k := 0; k < int(count); k++ {
			children[k] = br.readOperand(varMap)
		}
		lhsOp := br.readOperand(varMap)
		return bir.NewXMLSequenceInstr(lhsOp, children, pos)
	case bir.INSTRUCTION_KIND_EVAL_TEMPLATE_EXPR:
		var kind uint8
		br.read(&kind)
		strCount := br.readLength()
		strs := make([]string, strCount)
		for k := 0; k < int(strCount); k++ {
			strs[k] = string(br.readStringCPEntry())
		}
		var totalLen int32
		br.read(&totalLen)
		insCount := br.readLength()
		insertions := make([]*bir.BIROperand, insCount)
		for k := 0; k < int(insCount); k++ {
			insertions[k] = br.readOperand(varMap)
		}
		lhsOp := br.readOperand(varMap)
		return bir.NewEvalTemplateExpr(bir.TemplateKind(kind), strs, insertions, lhsOp, pos)
	default:
		panic(fmt.Sprintf("unsupported instruction kind: %d", instructionKind))
	}
}

func (br *birReader) readTerminator(varMap map[string]bir.BIRVariableDcl) bir.BIRTerminator {
	var terminatorKind uint8
	br.read(&terminatorKind)

	if terminatorKind == 0 {
		return nil
	}

	termInstructionKind := bir.InstructionKind(terminatorKind)
	pos := br.readPosition()

	switch termInstructionKind {
	case bir.INSTRUCTION_KIND_RETURN:
		return &bir.Return{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				},
			},
		}

	case bir.INSTRUCTION_KIND_GOTO:
		id := br.readStringCPEntry()
		return &bir.Goto{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				},
				ThenBB: &bir.BIRBasicBlock{
					Id: id,
				},
			},
		}
	case bir.INSTRUCTION_KIND_BRANCH:
		op := br.readOperand(varMap)
		trueBBId := br.readStringCPEntry()
		falseBBId := br.readStringCPEntry()

		return &bir.Branch{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				},
			},
			Op: op,
			TrueBB: &bir.BIRBasicBlock{
				Id: trueBBId,
			},
			FalseBB: &bir.BIRBasicBlock{
				Id: falseBBId,
			},
		}
	case bir.INSTRUCTION_KIND_CALL, bir.INSTRUCTION_KIND_FP_CALL:
		var isMethodCall bool
		br.read(&isMethodCall)

		pkg := br.readPackageCPEntry()
		name := br.readStringCPEntry()
		functionLookupKey := br.readStringCPEntry()
		argsCount := br.readLength()

		args := make([]bir.BIROperand, argsCount)
		for k := 0; k < int(argsCount); k++ {
			arg := br.readOperand(varMap)
			args[k] = *arg
		}

		var lshOpExists bool
		br.read(&lshOpExists)

		var lhsOp *bir.BIROperand
		if lshOpExists {
			lhsOp = br.readOperand(varMap)
		}

		thenBBId := br.readStringCPEntry()

		var fpOperand *bir.BIROperand
		if termInstructionKind == bir.INSTRUCTION_KIND_FP_CALL {
			fpOperand = br.readOperand(varMap)
		}

		return &bir.Call{
			Kind:              termInstructionKind,
			IsMethodCall:      isMethodCall,
			CalleePkg:         pkg,
			Name:              name,
			FunctionLookupKey: string(functionLookupKey),
			Args:              args,
			FpOperand:         fpOperand,
			BIRTerminatorBase: bir.BIRTerminatorBase{
				ThenBB: &bir.BIRBasicBlock{
					Id: thenBBId,
				},
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
					LhsOp:       lhsOp,
				},
			},
		}
	case bir.INSTRUCTION_KIND_PANIC:
		errorOp := br.readOperand(varMap)
		return &bir.Panic{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				},
			},
			ErrorOp: errorOp,
		}
	case bir.INSTRUCTION_KIND_LOCK:
		key := br.readStringCPEntry()
		thenBBId := br.readStringCPEntry()
		return &bir.LockStart{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				},
				ThenBB: &bir.BIRBasicBlock{Id: thenBBId},
			},
			LockKey: string(key),
		}
	case bir.INSTRUCTION_KIND_RESOURCE_CALL:
		receiver := br.readOperand(varMap)
		methodNameN := br.readStringCPEntry()
		methodName := methodNameN.Value()
		segCount := br.readLength()
		pathSegments := make([]bir.BIROperand, segCount)
		for k := 0; k < int(segCount); k++ {
			pathSegments[k] = *br.readOperand(varMap)
		}
		argCount := br.readLength()
		args := make([]bir.BIROperand, argCount)
		for k := 0; k < int(argCount); k++ {
			args[k] = *br.readOperand(varMap)
		}
		var lhsExists bool
		br.read(&lhsExists)
		var lhsOp *bir.BIROperand
		if lhsExists {
			lhsOp = br.readOperand(varMap)
		}
		thenBBId := br.readStringCPEntry()
		return &bir.ResourceFunctionCall{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
					LhsOp:       lhsOp,
				},
				ThenBB: &bir.BIRBasicBlock{Id: thenBBId},
			},
			Receiver:     *receiver,
			MethodName:   methodName,
			PathSegments: pathSegments,
			Args:         args,
		}
	case bir.INSTRUCTION_KIND_UNLOCK:
		key := br.readStringCPEntry()
		thenBBId := br.readStringCPEntry()
		return &bir.LockEnd{
			BIRTerminatorBase: bir.BIRTerminatorBase{
				BIRInstructionBase: bir.BIRInstructionBase{
					BIRNodeBase: bir.BIRNodeBase{Pos: pos},
				},
				ThenBB: &bir.BIRBasicBlock{Id: thenBBId},
			},
			LockKey: string(key),
		}
	default:
		panic(fmt.Sprintf("unsupported terminator kind: %d", termInstructionKind))
	}
}

func (br *birReader) readOperand(varMap map[string]bir.BIRVariableDcl) *bir.BIROperand {
	var ignoreVariable bool
	br.read(&ignoreVariable)

	if ignoreVariable {
		ty := br.readType()
		ignored := &bir.BIRLocalVariableDcl{}
		ignored.Type = ty
		return &bir.BIROperand{
			VariableDcl: ignored,
		}
	}

	kind := br.readKind()
	_ = br.readScope() // scope (ignored)
	name := br.readStringCPEntry()

	if kind == bir.VAR_KIND_GLOBAL {
		lookupKey := br.readStringCPEntry()
		pkgId := br.readPackageCPEntry()
		gv := &bir.BIRGlobalVariableDcl{
			GlobalVarLookupKey: string(lookupKey),
		}
		gv.Name = name
		gv.PkgId = pkgId
		return &bir.BIROperand{VariableDcl: gv}
	}

	varDcl, ok := varMap[name.Value()]
	if !ok {
		varDcl = &bir.BIRLocalVariableDcl{}
		varDcl.(*bir.BIRLocalVariableDcl).SetName(name)
		varMap[name.Value()] = varDcl
	}

	var mode uint8
	var frameIndex, baseIndex int32
	br.read(&mode)
	br.read(&frameIndex)
	br.read(&baseIndex)

	return &bir.BIROperand{
		VariableDcl: varDcl,
		Address: bir.Address{
			Mode:       bir.AddressingMode(mode),
			FrameIndex: int(frameIndex),
			BaseIndex:  int(baseIndex),
		},
	}
}

func (br *birReader) readConstValue() any {
	var tagByte int8
	br.read(&tagByte)

	tag := typeTag(tagByte)
	return br.readConstValueByTag(tag)
}

func (br *birReader) readConstValueByTag(tag typeTag) any {
	switch tag {
	case typeTagInt,
		typeTagSigned32,
		typeTagSigned16,
		typeTagSigned8,
		typeTagUnsigned32,
		typeTagUnsigned16,
		typeTagUnsigned8:
		var val int64
		br.read(&val)
		return val
	case typeTagByte:
		var val byte
		br.read(&val)
		return val
	case typeTagFloat:
		var val float64
		br.read(&val)
		return val
	case typeTagBoolean:
		var val bool
		br.read(&val)
		return val
	case typeTagString, typeTagCharString:
		var idx int32
		br.read(&idx)
		return br.getStringFromCP(int(idx))
	case typeTagDecimal:
		var idx int32
		br.read(&idx)
		str := br.getStringFromCP(int(idx))
		r, err := decimal.FromString(str)
		if err != nil {
			panic(fmt.Sprintf("invalid decimal value %q: %v", str, err))
		}
		return r
	case typeTagNil:
		var idx int32
		br.read(&idx)
		return nil
	default:
		var idx int32
		br.read(&idx)

		if idx == -1 {
			return nil
		}
		return br.getFromCP(int(idx))
	}
}

func (br *birReader) restFillerFactoryForListType(ty semtypes.SemType) values.FillerFactory {
	if semtypes.IsZero(ty) {
		return nil
	}
	tyCx := semtypes.TypeCheckContext(br.ctx.GetTypeEnv())
	lat := semtypes.ToListAtomicType(tyCx, ty)
	if lat == nil {
		return nil
	}
	factory, _ := values.FillerFactoryFor(tyCx, lat.Rest())
	return factory
}

func (br *birReader) read(v any) {
	err := binary.Read(br.r, binary.BigEndian, v)
	if err != nil {
		panic(fmt.Sprintf("binary read failed: %v", err))
	}
}

func (br *birReader) readKind() bir.VarKind {
	var val uint8
	br.read(&val)
	return bir.VarKind(val)
}

func (br *birReader) readFlags() model.Flag {
	var val int64
	br.read(&val)
	return model.Flag(val)
}

func (br *birReader) readStringCPEntry() model.Name {
	var idx int32
	br.read(&idx)
	return model.Name(br.getStringFromCP(int(idx)))
}

func (br *birReader) readLength() int64 {
	var val int64
	br.read(&val)
	return val
}

func (br *birReader) readInstructionKind() bir.InstructionKind {
	var val uint8
	br.read(&val)
	return bir.InstructionKind(val)
}

func (br *birReader) readScope() bir.VarScope {
	var val uint8
	br.read(&val)
	return bir.VarScope(val)
}

func (br *birReader) readPackageCPEntry() *model.PackageID {
	var idx int32
	br.read(&idx)
	if idx == -1 {
		return nil
	}
	return br.getPackageFromCP(int(idx))
}

func (br *birReader) readPosition() bir.Location {
	var sourceFileIdx int32
	br.read(&sourceFileIdx)

	sourceFileName := br.getStringFromCP(int(sourceFileIdx))

	var sLine int32
	br.read(&sLine)
	var sCol int32
	br.read(&sCol)
	var eLine int32
	br.read(&eLine)
	var eCol int32
	br.read(&eCol)

	return bir.NewLocation(sourceFileName, int(sLine), int(eLine), int(sCol), int(eCol))
}
