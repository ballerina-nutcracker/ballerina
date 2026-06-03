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
	"fmt"
	"math"
	"sort"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	BIR_MAGIC   = "\xba\x10\xc0\xde"
	BIR_VERSION = 82
)

type birWriter struct {
	cp  *ConstantPool
	tp  *semtypes.TypePool
	env semtypes.Env
}

func Marshal(tyEnv semtypes.Env, pkg *bir.BIRPackage) ([]byte, error) {
	writer := &birWriter{
		cp:  NewConstantPool(),
		tp:  semtypes.NewTypePool(),
		env: tyEnv,
	}
	return writer.serialize(pkg)
}

func (bw *birWriter) serialize(pkg *bir.BIRPackage) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("BIR serializer failed due to %s", r)
		}
	}()

	birbuf := &bytes.Buffer{}
	bw.writePackageCPEntry(birbuf, pkg.PackageID)
	bw.writeGlobalVars(birbuf, pkg)
	bw.writeClassDefs(birbuf, pkg)
	bw.writeFunctions(birbuf, pkg)

	buf := &bytes.Buffer{}
	_, err = buf.Write([]byte(BIR_MAGIC))
	if err != nil {
		panic(fmt.Sprintf("writing BIR magic: %v", err))
	}

	write(buf, int32(BIR_VERSION))

	tpBytes := semtypes.MarshalTypePool(bw.tp, bw.env)
	write(buf, int64(len(tpBytes)))
	_, err = buf.Write(tpBytes)
	if err != nil {
		panic(fmt.Sprintf("writing type pool bytes: %v", err))
	}

	cpBytes, err := bw.cp.Serialize()
	if err != nil {
		panic(fmt.Sprintf("serializing constant pool: %v", err))
	}

	_, err = buf.Write(cpBytes)
	if err != nil {
		panic(fmt.Sprintf("writing constant pool bytes: %v", err))
	}

	_, err = buf.Write(birbuf.Bytes())
	if err != nil {
		panic(fmt.Sprintf("writing BIR buffer bytes: %v", err))
	}

	return buf.Bytes(), nil
}

func (bw *birWriter) writeGlobalVars(buf *bytes.Buffer, pkg *bir.BIRPackage) {
	bw.writeLength(buf, len(pkg.GlobalVars))
	for _, gv := range pkg.GlobalVars {
		bw.writePosition(buf, gv.Pos)
		bw.writeKind(buf, bir.VAR_KIND_GLOBAL)
		name := gv.GetName()
		bw.writeStringCPEntry(buf, name.Value())
		bw.writeFlags(buf, gv.Flags)
		bw.writeType(buf, gv.GetType())
	}
}

func (bw *birWriter) writeClassDefs(buf *bytes.Buffer, pkg *bir.BIRPackage) {
	bw.writeLength(buf, len(pkg.ClassDefs))
	for _, classDef := range pkg.ClassDefs {
		bw.writeClassDef(buf, &classDef)
	}
}

func (bw *birWriter) writeClassDef(buf *bytes.Buffer, classDef *bir.BIRClassDef) {
	bw.writeStringCPEntry(buf, classDef.Name.Value())
	bw.writeStringCPEntry(buf, classDef.LookupKey)
	bw.writeLength(buf, len(classDef.Fields))
	for _, field := range classDef.Fields {
		bw.writeStringCPEntry(buf, field.Name)
		bw.writeType(buf, field.Ty)
	}
	var methodNames []string
	for name := range classDef.VTable {
		methodNames = append(methodNames, name)
	}
	sort.Strings(methodNames)
	bw.writeLength(buf, len(methodNames))
	for _, name := range methodNames {
		bw.writeStringCPEntry(buf, name)
		bw.writeFunction(buf, classDef.VTable[name])
	}
	var rmNames []string
	for name := range classDef.RTable {
		rmNames = append(rmNames, name)
	}
	sort.Strings(rmNames)
	bw.writeLength(buf, len(rmNames))
	for _, name := range rmNames {
		entries := classDef.RTable[name]
		bw.writeStringCPEntry(buf, name)
		bw.writeLength(buf, len(entries))
		for i := range entries {
			entry := &entries[i]
			bw.writeLength(buf, len(entry.PathSegments))
			for _, seg := range entry.PathSegments {
				bw.writeType(buf, seg.Ty)
			}
			bw.writeType(buf, entry.RestSegmentTy)
			bw.writeFunction(buf, entry.Fn)
		}
	}
}

func (bw *birWriter) writeFunctions(buf *bytes.Buffer, pkg *bir.BIRPackage) {
	write(buf, pkg.InitFunction != nil)
	if pkg.InitFunction != nil {
		bw.writeFunction(buf, pkg.InitFunction)
	}
	write(buf, pkg.MainFunction != nil)
	if pkg.MainFunction != nil {
		bw.writeFunction(buf, pkg.MainFunction)
	}
	bw.writeLength(buf, len(pkg.Functions))
	for _, fn := range pkg.Functions {
		bw.writeFunction(buf, &fn)
	}
}

func (bw *birWriter) writeFunction(buf *bytes.Buffer, fn *bir.BIRFunction) {
	bw.writePosition(buf, fn.Pos)
	bw.writeStringCPEntry(buf, fn.Name.Value())
	bw.writeStringCPEntry(buf, fn.OriginalName.Value())
	bw.writeFlags(buf, fn.Flags)
	bw.writeStringCPEntry(buf, fn.FunctionLookupKey)

	bw.writeLength(buf, len(fn.RequiredParams))
	for _, requiredParam := range fn.RequiredParams {
		bw.writeStringCPEntry(buf, requiredParam.Name.Value())
		bw.writeFlags(buf, requiredParam.Flags)
	}
	write(buf, fn.RestParams != nil)

	birbuf := &bytes.Buffer{}
	bw.writeLength(birbuf, fn.ArgsCount)

	write(birbuf, fn.ReturnVariable != nil)
	if fn.ReturnVariable != nil {
		bw.writeKind(birbuf, bir.VAR_KIND_RETURN)
		bw.writeType(birbuf, fn.ReturnVariable.GetType())
		retName := fn.ReturnVariable.GetName()
		bw.writeStringCPEntry(birbuf, retName.Value())
	}

	bw.writeLength(birbuf, len(fn.LocalVars))
	for _, localVar := range fn.LocalVars {
		bw.writeLocalVar(birbuf, &localVar)
	}

	bw.writeLength(birbuf, len(fn.BasicBlocks))
	for _, bb := range fn.BasicBlocks {
		bw.writeBasicBlock(birbuf, &bb)
	}

	bw.writeLength(birbuf, len(fn.ErrorTable))
	for _, entry := range fn.ErrorTable {
		bw.writeStringCPEntry(birbuf, fmt.Sprintf("bb%d", entry.Start))
		bw.writeStringCPEntry(birbuf, fmt.Sprintf("bb%d", entry.End))
		bw.writeStringCPEntry(birbuf, fmt.Sprintf("bb%d", entry.Target))
		bw.writeOperand(birbuf, entry.ErrorOp)
	}

	bw.writeBufferLength(buf, birbuf)
	_, err := buf.Write(birbuf.Bytes())
	if err != nil {
		panic(fmt.Sprintf("writing function body bytes: %v", err))
	}
}

func (bw *birWriter) writeLocalVar(buf *bytes.Buffer, localVar *bir.BIRLocalVariableDcl) {
	bw.writeKind(buf, bir.VAR_KIND_LOCAL)
	bw.writeType(buf, localVar.GetType())
	name := localVar.GetName()
	bw.writeStringCPEntry(buf, name.Value())
}

func (bw *birWriter) writeBasicBlock(buf *bytes.Buffer, bb *bir.BIRBasicBlock) {
	bw.writeStringCPEntry(buf, bb.Id.Value())
	bw.writeLength(buf, len(bb.Instructions))

	for _, instr := range bb.Instructions {
		bw.writeInstructionKind(buf, instr.GetKind())
		bw.writePosition(buf, instr.GetPos())
		bw.writeInstruction(buf, instr)
	}

	if bb.Terminator == nil {
		write(buf, uint8(0))
		return
	}
	bw.writeInstructionKind(buf, bb.Terminator.GetKind())
	bw.writePosition(buf, bb.Terminator.GetPos())
	bw.writeTerminator(buf, bb.Terminator)
}

func (bw *birWriter) writeInstruction(buf *bytes.Buffer, instr bir.BIRInstruction) {
	switch instr := instr.(type) {
	case *bir.Move:
		bw.writeOperand(buf, instr.RhsOp)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.BinaryOp:
		bw.writeOperand(buf, &instr.RhsOp1)
		bw.writeOperand(buf, &instr.RhsOp2)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.UnaryOp:
		bw.writeOperand(buf, instr.RhsOp)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.ConstantLoad:
		write(buf, int32(-1))
		bw.writeOperand(buf, instr.LhsOp)

		isWrapped := false
		tagValue := instr.Value
		if cv, isConstValue := instr.Value.(bir.ConstValue); isConstValue {
			isWrapped = true
			tagValue = cv.Value
		}

		write(buf, isWrapped)
		bw.writeConstValue(buf, tagValue)
	case *bir.FieldAccess:
		// TODO: MAP_LOAD and ARRAY_LOAD
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeOperand(buf, instr.KeyOp)
		bw.writeOperand(buf, instr.RhsOp)
	case *bir.NewArray:
		bw.writeType(buf, instr.Type)
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeOperand(buf, instr.SizeOp)
		write(buf, instr.IsReadonly)
		bw.writeLength(buf, len(instr.Values))
		for _, v := range instr.Values {
			bw.writeOperand(buf, v)
		}
	case *bir.TypeCast:
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeOperand(buf, instr.RhsOp)
		bw.writeType(buf, instr.Type)
		// TODO: Write checkTypes
	case *bir.TypeTest:
		bw.writeOperand(buf, instr.RhsOp)
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeType(buf, instr.Type)
		write(buf, instr.IsNegation)
	case *bir.NewMap:
		bw.writeType(buf, instr.Type)
		bw.writeOperand(buf, instr.LhsOp)
		write(buf, instr.IsReadonly)
		bw.writeLength(buf, len(instr.Values))
		for _, entry := range instr.Values {
			write(buf, entry.IsKeyValuePair())
			if entry.IsKeyValuePair() {
				kvEntry := entry.(*bir.MappingConstructorKeyValueEntry)
				bw.writeOperand(buf, kvEntry.KeyOp())
				bw.writeOperand(buf, kvEntry.ValueOp())
			}
		}
		bw.writeLength(buf, len(instr.Defaults))
		for _, def := range instr.Defaults {
			bw.writeStringCPEntry(buf, def.FieldName)
			bw.writeStringCPEntry(buf, def.FunctionLookupKey)
		}
	case *bir.NewError:
		bw.writeType(buf, instr.Type)
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeStringCPEntry(buf, instr.TypeName)
		bw.writeOperand(buf, instr.MessageOp)
		write(buf, instr.CauseOp != nil)
		if instr.CauseOp != nil {
			bw.writeOperand(buf, instr.CauseOp)
		}
		write(buf, instr.DetailOp != nil)
		if instr.DetailOp != nil {
			bw.writeOperand(buf, instr.DetailOp)
		}
	case *bir.NewObject:
		bw.writeStringCPEntry(buf, instr.ClassDefRef)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.NewStream:
		bw.writeType(buf, instr.StreamType)
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeOperand(buf, instr.ImplOp)
	case *bir.StreamNext:
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeOperand(buf, instr.StreamOp)
	case *bir.StreamClose:
		bw.writeOperand(buf, instr.LhsOp)
		bw.writeOperand(buf, instr.StreamOp)
	case *bir.FPLoad:
		bw.writeStringCPEntry(buf, instr.FunctionLookupKey)
		bw.writeType(buf, instr.Type)
		bw.writeOperand(buf, instr.LhsOp)
		write(buf, instr.IsClosure)
	case *bir.PushScopeFrame:
		write(buf, int32(instr.NumLocals))
	case *bir.PopScopeFrame:
		// no fields to write
	case *bir.NewXMLElement:
		bw.writeOperand(buf, instr.NameOp)
		write(buf, instr.ChildrenOp != nil)
		if instr.ChildrenOp != nil {
			bw.writeOperand(buf, instr.ChildrenOp)
		}
		write(buf, instr.AttrsOp != nil)
		if instr.AttrsOp != nil {
			bw.writeOperand(buf, instr.AttrsOp)
		}
		write(buf, instr.NamespacesOp != nil)
		if instr.NamespacesOp != nil {
			bw.writeOperand(buf, instr.NamespacesOp)
		}
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.NewXMLPI:
		bw.writeOperand(buf, instr.TargetOp)
		bw.writeOperand(buf, instr.DataOp)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.NewXMLComment:
		bw.writeOperand(buf, instr.BodyOp)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.NewXMLText:
		bw.writeOperand(buf, instr.BodyOp)
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.NewXMLSequence:
		bw.writeLength(buf, len(instr.Children))
		for _, child := range instr.Children {
			bw.writeOperand(buf, child)
		}
		bw.writeOperand(buf, instr.LhsOp)
	case *bir.EvalTemplateExpr:
		write(buf, uint8(instr.Kind))
		bw.writeLength(buf, len(instr.Strings))
		for _, s := range instr.Strings {
			bw.writeStringCPEntry(buf, s)
		}
		write(buf, int32(instr.LiteralsTotalLen))
		bw.writeLength(buf, len(instr.Insertions))
		for _, op := range instr.Insertions {
			bw.writeOperand(buf, op)
		}
		bw.writeOperand(buf, instr.LhsOp)
	default:
		panic(fmt.Sprintf("unsupported instruction type: %T", instr))
	}
}

func (bw *birWriter) writeTerminator(buf *bytes.Buffer, term bir.BIRTerminator) {
	switch term := term.(type) {
	case *bir.Goto:
		bw.writeStringCPEntry(buf, term.ThenBB.Id.Value())
	case *bir.Branch:
		bw.writeOperand(buf, term.Op)
		bw.writeStringCPEntry(buf, term.TrueBB.Id.Value())
		bw.writeStringCPEntry(buf, term.FalseBB.Id.Value())
	case *bir.Call:
		write(buf, term.IsMethodCall)
		bw.writePackageCPEntry(buf, term.CalleePkg)
		bw.writeStringCPEntry(buf, term.Name.Value())
		bw.writeStringCPEntry(buf, term.FunctionLookupKey)

		bw.writeLength(buf, len(term.Args))
		for _, arg := range term.Args {
			bw.writeOperand(buf, &arg)
		}

		if term.LhsOp != nil {
			write(buf, uint8(1))
			bw.writeOperand(buf, term.LhsOp)
		} else {
			write(buf, uint8(0))
		}

		bw.writeStringCPEntry(buf, term.ThenBB.Id.Value())

		if term.Kind == bir.INSTRUCTION_KIND_FP_CALL {
			bw.writeOperand(buf, term.FpOperand)
		}
	case *bir.Return:
	case *bir.Panic:
		bw.writeOperand(buf, term.ErrorOp)
	case *bir.LockStart:
		bw.writeStringCPEntry(buf, term.LockKey)
		bw.writeStringCPEntry(buf, term.ThenBB.Id.Value())
	case *bir.LockEnd:
		bw.writeStringCPEntry(buf, term.LockKey)
		bw.writeStringCPEntry(buf, term.ThenBB.Id.Value())
	case *bir.ResourceFunctionCall:
		bw.writeOperand(buf, &term.Receiver)
		bw.writeStringCPEntry(buf, term.MethodName)
		bw.writeLength(buf, len(term.PathSegments))
		for i := range term.PathSegments {
			bw.writeOperand(buf, &term.PathSegments[i])
		}
		bw.writeLength(buf, len(term.Args))
		for i := range term.Args {
			bw.writeOperand(buf, &term.Args[i])
		}
		if term.LhsOp != nil {
			write(buf, uint8(1))
			bw.writeOperand(buf, term.LhsOp)
		} else {
			write(buf, uint8(0))
		}
		bw.writeStringCPEntry(buf, term.ThenBB.Id.Value())
	default:
		panic(fmt.Sprintf("unsupported terminator type: %T", term))
	}
}

func (bw *birWriter) writeOperand(buf *bytes.Buffer, op *bir.BIROperand) {
	if op == nil || op.VariableDcl == nil {
		write(buf, false)
		write(buf, uint8(bir.VAR_KIND_TEMP))
		bw.writeScope(buf, bir.VAR_SCOPE_FUNCTION)
		bw.writeStringCPEntry(buf, "")
		return
	}

	write(buf, false)
	// Determine kind and scope from concrete type
	var kind bir.VarKind
	var scope bir.VarScope
	if _, ok := op.VariableDcl.(*bir.BIRGlobalVariableDcl); ok {
		kind = bir.VAR_KIND_GLOBAL
		scope = bir.VAR_SCOPE_GLOBAL
	} else {
		kind = bir.VAR_KIND_LOCAL
		scope = bir.VAR_SCOPE_FUNCTION
	}
	bw.writeKind(buf, kind)
	bw.writeScope(buf, scope)
	name := op.VariableDcl.GetName()
	bw.writeStringCPEntry(buf, name.Value())
	if gv, ok := op.VariableDcl.(*bir.BIRGlobalVariableDcl); ok {
		bw.writeStringCPEntry(buf, gv.GlobalVarLookupKey)
		bw.writePackageCPEntry(buf, gv.PkgId)
	} else {
		write(buf, uint8(op.Address.Mode))
		write(buf, int32(op.Address.FrameIndex))
		write(buf, int32(op.Address.BaseIndex))
	}
}

func (bw *birWriter) writeConstValue(buf *bytes.Buffer, value any) {
	tag, err := bw.inferTag(value)
	if err != nil {
		panic(fmt.Sprintf("inferring constant load tag: %v", err))
	}
	write(buf, int8(tag))
	bw.writeConstValueByTag(buf, tag, value)
}

func (bw *birWriter) writeConstValueByTag(buf *bytes.Buffer, tag typeTag, value any) {
	if cv, isConstValue := value.(bir.ConstValue); isConstValue {
		bw.writeConstValueByTag(buf, tag, cv.Value)
		return
	}

	switch tag {
	case typeTagInt,
		typeTagSigned32,
		typeTagSigned16,
		typeTagSigned8,
		typeTagUnsigned32,
		typeTagUnsigned16,
		typeTagUnsigned8:
		var val int64
		switch v := value.(type) {
		case int64:
			val = v
		case int:
			val = int64(v)
		case int32:
			val = int64(v)
		case int16:
			val = int64(v)
		case int8:
			val = int64(v)
		default:
			panic(fmt.Sprintf("expected integer for tag %v, got %T", tag, value))
		}
		write(buf, val)
	case typeTagByte:
		var val byte
		switch v := value.(type) {
		case byte:
			val = v
		case int:
			val = byte(v)
		case int32:
			val = byte(v)
		default:
			panic(fmt.Sprintf("expected byte for tag %v, got %T", tag, value))
		}
		write(buf, val)
	case typeTagFloat:
		var val float64
		switch v := value.(type) {
		case float64:
			val = v
		case float32:
			val = float64(v)
		default:
			panic(fmt.Sprintf("expected float for tag %v, got %T", tag, value))
		}
		write(buf, val)
	case typeTagString, typeTagCharString, typeTagDecimal:
		var val string
		switch v := value.(type) {
		case string:
			val = v
		case *string:
			if v != nil {
				val = *v
			} else {
				val = ""
			}
		case *decimal.Decimal:
			val = v.String()
		default:
			panic(fmt.Sprintf("expected string for tag %v, got %T", tag, value))
		}
		cpIdx := bw.cp.AddStringCPEntry(val)
		write(buf, cpIdx)
	case typeTagBoolean:
		var val bool
		switch v := value.(type) {
		case bool:
			val = v
		default:
			panic(fmt.Sprintf("expected boolean for tag %v, got %T", tag, value))
		}
		write(buf, val)
	case typeTagNil:
		write(buf, int32(-1))
	case typeTagMap:
		m, ok := value.(*values.Map)
		if !ok {
			panic(fmt.Sprintf("expected map for tag %v, got %T", tag, value))
		}
		bw.writeType(buf, m.Type)
		write(buf, m.IsReadonly())
		keys := m.Keys()
		write(buf, int64(len(keys)))
		for _, key := range keys {
			bw.writeStringCPEntry(buf, key)
			value, _ := m.Get(key)
			bw.writeConstValue(buf, value)
		}
	case typeTagList:
		list, ok := value.(*values.List)
		if !ok {
			panic(fmt.Sprintf("expected list for tag %v, got %T", tag, value))
		}
		bw.writeType(buf, list.Type)
		write(buf, list.IsReadonly())
		write(buf, int64(list.Len()))
		for i := 0; i < list.Len(); i++ {
			bw.writeConstValue(buf, list.Get(i))
		}
	case typeTagTypedesc:
		td, ok := value.(*values.TypeDesc)
		if !ok {
			panic(fmt.Sprintf("expected typedesc for tag %v, got %T", tag, value))
		}
		bw.writeType(buf, td.Type)
		keys := make([]string, 0, len(td.Annotations))
		for key := range td.Annotations {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		write(buf, int64(len(keys)))
		for _, key := range keys {
			bw.writeStringCPEntry(buf, key)
			bw.writeConstValue(buf, td.Annotations[key])
		}
	case typeTagRuntimeRef:
		ref, ok := value.(*values.RuntimeAnnotationValueRef)
		if !ok {
			panic(fmt.Sprintf("expected runtime annotation reference for tag %v, got %T", tag, value))
		}
		bw.writeStringCPEntry(buf, ref.Organization)
		bw.writeStringCPEntry(buf, ref.Module)
		bw.writeStringCPEntry(buf, ref.GlobalName)
	default:
		panic(fmt.Sprintf("unsupported tag for constant value: %v", tag))
	}
}

// FIXME: Remove this after implementing types
func (bw *birWriter) inferTag(value any) (typeTag, error) {
	switch v := value.(type) {
	case bir.ConstValue:
		return bw.inferTag(v.Value)
	case int, int64, int32, int16, int8:
		return typeTagInt, nil
	case float64, float32:
		return typeTagFloat, nil
	case string, *string:
		return typeTagString, nil
	case bool:
		return typeTagBoolean, nil
	case byte:
		return typeTagByte, nil
	case *decimal.Decimal:
		return typeTagDecimal, nil
	case *values.Map:
		return typeTagMap, nil
	case *values.List:
		return typeTagList, nil
	case *values.TypeDesc:
		return typeTagTypedesc, nil
	case *values.RuntimeAnnotationValueRef:
		return typeTagRuntimeRef, nil
	case nil:
		return typeTagNil, nil
	default:
		return 0, fmt.Errorf("cannot infer tag for value %v (%T)", value, value)
	}
}

func (bw *birWriter) writeKind(buf *bytes.Buffer, kind bir.VarKind) {
	write(buf, uint8(kind))
}

func (bw *birWriter) writeFlags(buf *bytes.Buffer, flags model.Flag) {
	write(buf, int64(flags))
}

func (bw *birWriter) writeStringCPEntry(buf *bytes.Buffer, str string) {
	write(buf, bw.cp.AddStringCPEntry(str))
}

func (bw *birWriter) writeLength(buf *bytes.Buffer, length int) {
	write(buf, int64(length))
}

func (bw *birWriter) writeInstructionKind(buf *bytes.Buffer, kind bir.InstructionKind) {
	write(buf, uint8(kind))
}

func (bw *birWriter) writeScope(buf *bytes.Buffer, scope bir.VarScope) {
	write(buf, uint8(scope))
}

func (bw *birWriter) writePackageCPEntry(buf *bytes.Buffer, pkgID *model.PackageID) {
	pkgIdx := int32(-1)
	if pkgID != nil {
		pkgIdx = bw.cp.AddPackageCPEntry(pkgID)
	}
	write(buf, pkgIdx)
}

func (bw *birWriter) writeBufferLength(buf *bytes.Buffer, birbuf *bytes.Buffer) {
	write(buf, int64(birbuf.Len()))
}

func (bw *birWriter) writeType(buf *bytes.Buffer, ty semtypes.SemType) {
	if semtypes.IsZero(ty) {
		write(buf, int32(-1))
		return
	}
	write(buf, int32(bw.tp.Put(ty)))
}

func (bw *birWriter) writePosition(buf *bytes.Buffer, pos bir.Location) {
	var sLine int32 = math.MaxInt32
	var eLine int32 = math.MaxInt32
	var sCol int32 = math.MaxInt32
	var eCol int32 = math.MaxInt32
	sourceFileName := ""

	if !bir.IsLocationEmpty(pos) {
		sLine = int32(pos.StartLine())
		eLine = int32(pos.EndLine())
		sCol = int32(pos.StartColumn())
		eCol = int32(pos.EndColumn())
		if pos.FilePath() != "" {
			sourceFileName = pos.FilePath()
		}
	}

	bw.writeStringCPEntry(buf, sourceFileName)
	write(buf, sLine)
	write(buf, sCol)
	write(buf, eLine)
	write(buf, eCol)
}
