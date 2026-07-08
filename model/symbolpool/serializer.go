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

package symbolpool

import (
	"bytes"
	"fmt"
	"sort"

	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	symMagic   = "\x53\x59\x4d\x42"
	symVersion = 8
)

const (
	symTagType uint8 = iota
	symTagClass
	symTagValue
	symTagFunction
	symTagDependentlyTypedFunction
	symTagRecord
	symTagObjectType
	symTagAnnotation
	symTagNetworkClass
	symTagResourceMethod
	symTagConstantValue
	symTagOpaque
	symTagErrorType
)

const (
	typeOpTagIdentity uint8 = iota
	typeOpTagRef
	typeOpTagUnion
	typeOpTagIntersect
)

const (
	inclusionMemberTagField uint8 = iota
	inclusionMemberTagMethod
	inclusionMemberTagRestType
)

const (
	symbolRefTagEmpty uint8 = iota
	symbolRefTagLocal
	symbolRefTagExternal
)

type serializedSymbolRefKey struct {
	pkg  model.PackageIdentifier
	name string
}

type symbolWriter struct {
	cp              *constantPool
	tp              *semtypes.TypePool
	compilerEnv     *context.CompilerEnvironment
	refMap          map[model.SymbolRef]int
	sigMap          map[context.FunctionSignatureHandle]int
	sigHandles      []context.FunctionSignatureHandle
	externalRefMap  map[serializedSymbolRefKey]int
	externalRefKeys []serializedSymbolRefKey
}

func Marshal(exported model.ExportedSymbolSpace, env *context.CompilerEnvironment) ([]byte, error) {
	sw := &symbolWriter{
		cp:          newConstantPool(),
		tp:          semtypes.NewTypePool(),
		compilerEnv: env,
	}
	return sw.serialize(exported)
}

// symbolAnnotations returns the annotation values stored for the symbol at the
// given ref. Annotations live on the compiler environment (keyed by ref) rather
// than on the symbol, so the serializer fetches them here to keep them written
// alongside their symbol.
func (sw *symbolWriter) symbolAnnotations(ref model.SymbolRef) values.AnnotationValues {
	return sw.compilerEnv.SymbolAnnotationValues(ref)
}

func (sw *symbolWriter) serialize(exported model.ExportedSymbolSpace) ([]byte, error) {
	body := &bytes.Buffer{}
	if err := sw.writeSymbolSpaces(body, exported.MainSpaces); err != nil {
		return nil, err
	}
	if err := sw.writeSymbolSpaces(body, exported.AnnotationSpaces); err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if _, err := buf.Write([]byte(symMagic)); err != nil {
		return nil, fmt.Errorf("writing magic: %v", err)
	}
	if err := write(buf, int32(symVersion)); err != nil {
		return nil, err
	}

	tpBytes := semtypes.MarshalTypePool(sw.tp, sw.compilerEnv.GetTypeEnv())
	if err := write(buf, int64(len(tpBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(tpBytes); err != nil {
		return nil, fmt.Errorf("writing type pool: %v", err)
	}

	cpBytes, err := sw.cp.serialize()
	if err != nil {
		return nil, fmt.Errorf("writing constant pool: %v", err)
	}
	if _, err := buf.Write(cpBytes); err != nil {
		return nil, fmt.Errorf("writing constant pool: %v", err)
	}

	if err := sw.writeExternalSymbolRefPool(buf); err != nil {
		return nil, err
	}

	if _, err := buf.Write(body.Bytes()); err != nil {
		return nil, fmt.Errorf("writing body: %v", err)
	}

	return buf.Bytes(), nil
}

func (sw *symbolWriter) writePackageIdentifier(buf *bytes.Buffer, pkg model.PackageIdentifier) error {
	if err := sw.writeStringCP(buf, pkg.Organization); err != nil {
		return err
	}
	if err := sw.writeStringCP(buf, pkg.Package); err != nil {
		return err
	}
	return sw.writeStringCP(buf, pkg.Version)
}

// symbolSpaceNilSentinel marks a nil space, distinguishing it from a non-nil
// but empty space (which still carries a package identifier).
const symbolSpaceNilSentinel = int64(-1)

func (sw *symbolWriter) writeSymbolSpaces(buf *bytes.Buffer, spaces []*model.SymbolSpace) error {
	spaces = compactSymbolSpaces(spaces)
	if len(spaces) == 0 {
		return write(buf, symbolSpaceNilSentinel)
	}

	snapshots := make([][]model.SymbolRef, len(spaces))
	totalLen := 0
	for i, space := range spaces {
		for ref := range space.Symbols() {
			snapshots[i] = append(snapshots[i], ref)
		}
		totalLen += len(snapshots[i])
	}
	if err := write(buf, int64(totalLen)); err != nil {
		return err
	}
	if err := sw.writePackageIdentifier(buf, spaces[0].Pkg); err != nil {
		return err
	}

	sw.refMap = make(map[model.SymbolRef]int, totalLen)
	sw.sigMap = make(map[context.FunctionSignatureHandle]int)
	sw.sigHandles = nil
	nextIndex := 0
	for _, refs := range snapshots {
		for _, ref := range refs {
			sw.refMap[ref] = nextIndex
			sw.collectFunctionSignature(ref)
			nextIndex++
		}
	}
	if err := sw.writeFunctionSignatureTable(buf); err != nil {
		return err
	}

	for _, refs := range snapshots {
		for _, ref := range refs {
			sym := sw.compilerEnv.GetSymbol(ref)
			if err := sw.writeSymbol(buf, ref, sym); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sw *symbolWriter) collectFunctionSignature(ref model.SymbolRef) {
	handle, ok := sw.compilerEnv.FunctionSignatureHandle(ref)
	if !ok {
		return
	}
	if _, exists := sw.sigMap[handle]; exists {
		return
	}
	sw.sigMap[handle] = len(sw.sigHandles)
	sw.sigHandles = append(sw.sigHandles, handle)
}

func (sw *symbolWriter) writeFunctionSignatureTable(buf *bytes.Buffer) error {
	if err := write(buf, int64(len(sw.sigHandles))); err != nil {
		return err
	}
	for _, handle := range sw.sigHandles {
		if err := sw.writeUntypedFunctionSignature(buf, sw.compilerEnv.GetFunctionSignatureByHandle(handle)); err != nil {
			return err
		}
	}
	return nil
}

func compactSymbolSpaces(spaces []*model.SymbolSpace) []*model.SymbolSpace {
	result := make([]*model.SymbolSpace, 0, len(spaces))
	for _, space := range spaces {
		if space != nil {
			result = append(result, space)
		}
	}
	return result
}

func (sw *symbolWriter) writeSymbol(buf *bytes.Buffer, ref model.SymbolRef, sym model.Symbol) error {
	if op, ok := sym.(model.OpaqueSymbol); ok {
		if err := write(buf, symTagOpaque); err != nil {
			return err
		}
		return write(buf, int32(op.OpaqueID()))
	}
	switch s := sym.(type) {
	case *model.NetworkClassSymbol:
		return sw.writeClassSymbol(buf, symTagNetworkClass, s, sw.symbolAnnotations(ref))
	case model.ClassSymbol:
		return sw.writeClassSymbol(buf, symTagClass, s, sw.symbolAnnotations(ref))
	case *model.RecordSymbol:
		return sw.writeRecordSymbol(buf, s, sw.symbolAnnotations(ref))
	case *model.ObjectTypeSymbol:
		return sw.writeObjectTypeSymbol(buf, s, sw.symbolAnnotations(ref))
	case *model.ErrorTypeSymbol:
		return sw.writeErrorTypeSymbol(buf, s, sw.symbolAnnotations(ref))
	case *model.TypeSymbol:
		return sw.writeTypeSymbol(buf, s, sw.symbolAnnotations(ref))
	case *model.ConstantValueSymbol:
		return sw.writeConstantValueSymbol(buf, s)
	case *model.VariableSymbol:
		return sw.writeValueSymbol(buf, s)
	case *model.AnnotationSymbol:
		return sw.writeAnnotationSymbol(buf, s)
	case model.DependentlyTypedFunctionSymbol:
		return sw.writeDependentlyTypedFunctionSymbol(buf, ref, s)
	case *model.ResourceMethodSymbol:
		return sw.writeResourceMethodSymbol(buf, ref, s)
	case model.FunctionSymbol:
		return sw.writeFunctionSymbol(buf, ref, s)
	default:
		return fmt.Errorf("unsupported symbol type: %T", sym)
	}
}

func (sw *symbolWriter) writeSymbolBase(buf *bytes.Buffer, sym model.Symbol) error {
	return sw.writeSymbolBaseWithType(buf, sym, sym.Type(), sw.writeType)
}

func (sw *symbolWriter) writeObjectSymbolBase(buf *bytes.Buffer, sym model.Symbol) error {
	return sw.writeSymbolBaseWithType(buf, sym, sym.Type(), sw.writeObjectDefinitionType)
}

func (sw *symbolWriter) writeErrorSymbolBase(buf *bytes.Buffer, sym model.Symbol) error {
	return sw.writeSymbolBaseWithType(buf, sym, sym.Type(), sw.writeErrorDefinitionType)
}

func (sw *symbolWriter) writeSymbolBaseWithType(buf *bytes.Buffer, sym model.Symbol, ty semtypes.SemType, writeType func(*bytes.Buffer, semtypes.SemType) error) error {
	if err := sw.writeStringCP(buf, sym.Name()); err != nil {
		return err
	}
	if err := write(buf, sym.IsPublic()); err != nil {
		return err
	}
	return writeType(buf, ty)
}

func (sw *symbolWriter) writeTypeSymbol(buf *bytes.Buffer, sym *model.TypeSymbol, annotations values.AnnotationValues) error {
	if err := write(buf, symTagType); err != nil {
		return err
	}
	if err := sw.writeSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := sw.writeAnnotationValues(buf, annotations); err != nil {
		return err
	}
	return sw.writeInclusionMembers(buf, nil)
}

func (sw *symbolWriter) writeRecordSymbol(buf *bytes.Buffer, sym *model.RecordSymbol, annotations values.AnnotationValues) error {
	if err := write(buf, symTagRecord); err != nil {
		return err
	}
	if err := sw.writeSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := sw.writeAnnotationValues(buf, annotations); err != nil {
		return err
	}
	return sw.writeInclusionMembers(buf, sym.Members())
}

func (sw *symbolWriter) writeObjectTypeSymbol(buf *bytes.Buffer, sym *model.ObjectTypeSymbol, annotations values.AnnotationValues) error {
	if err := write(buf, symTagObjectType); err != nil {
		return err
	}
	if err := sw.writeObjectSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := sw.writeAnnotationValues(buf, annotations); err != nil {
		return err
	}
	if err := sw.writeInclusionMembers(buf, sym.Members()); err != nil {
		return err
	}
	return sw.writeDistinctTypeIDs(buf, sym.DistinctTypeIDs())
}

func (sw *symbolWriter) writeErrorTypeSymbol(buf *bytes.Buffer, sym *model.ErrorTypeSymbol, annotations values.AnnotationValues) error {
	if err := write(buf, symTagErrorType); err != nil {
		return err
	}
	if err := sw.writeErrorSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := sw.writeAnnotationValues(buf, annotations); err != nil {
		return err
	}
	return sw.writeDistinctTypeIDs(buf, sym.DistinctTypeIDs())
}

func (sw *symbolWriter) writeDistinctTypeIDs(buf *bytes.Buffer, ids []int) error {
	if err := write(buf, int64(len(ids))); err != nil {
		return err
	}
	for _, id := range ids {
		ref, ok := sw.compilerEnv.DistinctTypeSymbolRef(id)
		if !ok {
			return fmt.Errorf("missing symbol ref for distinct type id %d", id)
		}
		if err := sw.writeSymbolRef(buf, ref); err != nil {
			return err
		}
	}
	return nil
}

func (sw *symbolWriter) writeInclusionMembers(buf *bytes.Buffer, members []model.InclusionMember) error {
	if err := write(buf, int64(len(members))); err != nil {
		return err
	}
	for _, m := range members {
		switch member := m.(type) {
		case *model.FieldDescriptor:
			if err := write(buf, inclusionMemberTagField); err != nil {
				return err
			}
			if err := sw.writeStringCP(buf, member.MemberName()); err != nil {
				return err
			}
			if err := sw.writeType(buf, member.MemberType()); err != nil {
				return err
			}
			if err := write(buf, member.IsPublic()); err != nil {
				return err
			}
			var flags uint8
			if member.IsReadonly() {
				flags |= 1
			}
			if member.IsOptional() {
				flags |= 2
			}
			if member.HasDefault() {
				flags |= 4
			}
			if err := write(buf, flags); err != nil {
				return err
			}
			if err := sw.writeSymbolRef(buf, member.DefaultFnRef); err != nil {
				return err
			}
		case *model.MethodDescriptor:
			if err := write(buf, inclusionMemberTagMethod); err != nil {
				return err
			}
			if err := sw.writeStringCP(buf, member.MemberName()); err != nil {
				return err
			}
			if err := sw.writeType(buf, member.MemberType()); err != nil {
				return err
			}
			if err := write(buf, uint8(member.MemberKind())); err != nil {
				return err
			}
			if err := write(buf, member.IsPublic()); err != nil {
				return err
			}
			if err := sw.writeSymbolRef(buf, member.MethodRef); err != nil {
				return err
			}
		case *model.RestTypeDescriptor:
			if err := write(buf, inclusionMemberTagRestType); err != nil {
				return err
			}
			if err := sw.writeType(buf, member.MemberType()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown inclusion member type: %T", m)
		}
	}
	return nil
}

func (sw *symbolWriter) writeSymbolRef(buf *bytes.Buffer, ref model.SymbolRef) error {
	if ref.IsEmpty() {
		return write(buf, symbolRefTagEmpty)
	}
	if idx, ok := sw.refMap[ref]; ok {
		if err := write(buf, symbolRefTagLocal); err != nil {
			return err
		}
		return write(buf, int32(idx))
	}
	idx := sw.externalSymbolRefIndex(ref)
	if err := write(buf, symbolRefTagExternal); err != nil {
		return err
	}
	return write(buf, int32(idx))
}

func (sw *symbolWriter) externalSymbolRefIndex(ref model.SymbolRef) int {
	key := serializedSymbolRefKey{
		pkg:  sw.compilerEnv.SymbolPackage(ref),
		name: sw.compilerEnv.SymbolName(ref),
	}
	if sw.externalRefMap == nil {
		sw.externalRefMap = make(map[serializedSymbolRefKey]int)
	}
	if idx, ok := sw.externalRefMap[key]; ok {
		return idx
	}
	idx := len(sw.externalRefKeys)
	sw.externalRefMap[key] = idx
	sw.externalRefKeys = append(sw.externalRefKeys, key)
	sw.cp.addString(key.pkg.Organization)
	sw.cp.addString(key.pkg.Package)
	sw.cp.addString(key.pkg.Version)
	sw.cp.addString(key.name)
	return idx
}

func (sw *symbolWriter) writeExternalSymbolRefPool(buf *bytes.Buffer) error {
	if err := write(buf, int64(len(sw.externalRefKeys))); err != nil {
		return err
	}
	for _, key := range sw.externalRefKeys {
		if err := sw.writePackageIdentifier(buf, key.pkg); err != nil {
			return err
		}
		if err := sw.writeStringCP(buf, key.name); err != nil {
			return err
		}
	}
	return nil
}

func (sw *symbolWriter) writeClassSymbol(buf *bytes.Buffer, tag uint8, sym model.ClassSymbol, annotations values.AnnotationValues) error {
	if err := write(buf, tag); err != nil {
		return err
	}
	if err := sw.writeObjectSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := sw.writeAnnotationValues(buf, annotations); err != nil {
		return err
	}
	if err := sw.writeInclusionMembers(buf, sym.Members()); err != nil {
		return err
	}
	if err := sw.writeDistinctTypeIDs(buf, sym.DistinctTypeIDs()); err != nil {
		return err
	}
	if tag == symTagNetworkClass {
		refs := sym.(*model.NetworkClassSymbol).ResourceMethods()
		if err := write(buf, int64(len(refs))); err != nil {
			return err
		}
		for _, ref := range refs {
			if err := sw.writeSymbolRef(buf, ref); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sw *symbolWriter) writeValueSymbol(buf *bytes.Buffer, sym *model.VariableSymbol) error {
	if err := write(buf, symTagValue); err != nil {
		return err
	}
	return sw.writeValueSymbolBody(buf, sym)
}

func (sw *symbolWriter) writeValueSymbolBody(buf *bytes.Buffer, sym *model.VariableSymbol) error {
	if err := sw.writeSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := write(buf, sym.Kind() == model.SymbolKindConstant); err != nil {
		return err
	}
	if err := write(buf, sym.Kind() == model.SymbolKindParemeter); err != nil {
		return err
	}
	if err := write(buf, sym.IsFinal()); err != nil {
		return err
	}
	if err := write(buf, sym.IsConfigurable()); err != nil {
		return err
	}
	return write(buf, sym.IsIsolated())
}

func (sw *symbolWriter) writeConstantValueSymbol(buf *bytes.Buffer, sym *model.ConstantValueSymbol) error {
	if err := write(buf, symTagConstantValue); err != nil {
		return err
	}
	if err := sw.writeValueSymbolBody(buf, &sym.VariableSymbol); err != nil {
		return err
	}
	return sw.writeAnnotationValue(buf, sym.ConstantValue())
}

func (sw *symbolWriter) writeAnnotationSymbol(buf *bytes.Buffer, sym *model.AnnotationSymbol) error {
	if err := write(buf, symTagAnnotation); err != nil {
		return err
	}
	if err := sw.writeSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := write(buf, sym.IsConst()); err != nil {
		return err
	}
	keys := sym.AttachPointKeys()
	sort.Strings(keys)
	if err := write(buf, int64(len(keys))); err != nil {
		return err
	}
	for _, key := range keys {
		if err := sw.writeStringCP(buf, key); err != nil {
			return err
		}
	}
	return nil
}

func (sw *symbolWriter) writeFunctionSymbol(buf *bytes.Buffer, ref model.SymbolRef, sym model.FunctionSymbol) error {
	if err := write(buf, symTagFunction); err != nil {
		return err
	}
	if err := sw.writeSymbolBase(buf, sym); err != nil {
		return err
	}
	return sw.writeFunctionSignatureBody(buf, ref, sym.TypedSignature())
}

func (sw *symbolWriter) writeFunctionSignatureBody(buf *bytes.Buffer, ref model.SymbolRef, sig model.TypedFunctionSignature) error {
	if err := write(buf, int64(len(sig.ParamTypes))); err != nil {
		return err
	}
	for _, pt := range sig.ParamTypes {
		if err := sw.writeType(buf, pt); err != nil {
			return err
		}
	}
	if err := sw.writeType(buf, sig.ReturnType); err != nil {
		return err
	}
	if err := write(buf, !semtypes.IsZero(sig.RestParamType)); err != nil {
		return err
	}
	if !semtypes.IsZero(sig.RestParamType) {
		if err := sw.writeType(buf, sig.RestParamType); err != nil {
			return err
		}
	}
	if err := write(buf, uint8(sig.Flags)); err != nil {
		return err
	}
	return sw.writeFunctionSignatureIndex(buf, ref)
}

func (sw *symbolWriter) writeResourceMethodSymbol(buf *bytes.Buffer, ref model.SymbolRef, sym *model.ResourceMethodSymbol) error {
	if err := write(buf, symTagResourceMethod); err != nil {
		return err
	}
	if err := sw.writeSymbolBase(buf, sym); err != nil {
		return err
	}
	if err := sw.writeStringCP(buf, sym.MethodName()); err != nil {
		return err
	}
	if err := sw.writeType(buf, sym.PathListType()); err != nil {
		return err
	}
	return sw.writeFunctionSignatureBody(buf, ref, sym.TypedSignature())
}

func (sw *symbolWriter) writeDependentlyTypedFunctionSymbol(buf *bytes.Buffer, ref model.SymbolRef, sym model.DependentlyTypedFunctionSymbol) error {
	if err := write(buf, symTagDependentlyTypedFunction); err != nil {
		return err
	}
	if err := sw.writeStringCP(buf, sym.Name()); err != nil {
		return err
	}
	if err := write(buf, sym.IsPublic()); err != nil {
		return err
	}
	paramTypes := sym.ParamTypes()
	if err := write(buf, int64(len(paramTypes))); err != nil {
		return err
	}
	for _, pt := range paramTypes {
		if err := sw.writeType(buf, pt); err != nil {
			return err
		}
	}
	if err := write(buf, uint8(sym.FuncFlags())); err != nil {
		return err
	}
	if err := sw.writeFunctionSignatureIndex(buf, ref); err != nil {
		return err
	}
	return sw.writeTypeOp(buf, sym.ReturnType())
}

func (sw *symbolWriter) writeTypeOp(buf *bytes.Buffer, op model.TypeOp) error {
	switch o := op.(type) {
	case *model.IdentityTypeOp:
		if err := write(buf, typeOpTagIdentity); err != nil {
			return err
		}
		return sw.writeType(buf, o.Type)
	case *model.RefTypeOp:
		if err := write(buf, typeOpTagRef); err != nil {
			return err
		}
		return write(buf, int64(o.Index))
	case *model.BinaryTypeOp:
		var tag uint8
		if o.Kind == model.TypeOpUnion {
			tag = typeOpTagUnion
		} else {
			tag = typeOpTagIntersect
		}
		if err := write(buf, tag); err != nil {
			return err
		}
		if err := sw.writeTypeOp(buf, o.Lhs); err != nil {
			return err
		}
		return sw.writeTypeOp(buf, o.Rhs)
	default:
		return fmt.Errorf("unsupported TypeOp: %T", op)
	}
}

func (sw *symbolWriter) writeFunctionSignatureIndex(buf *bytes.Buffer, ref model.SymbolRef) error {
	handle, ok := sw.compilerEnv.FunctionSignatureHandle(ref)
	index := int64(-1)
	if ok {
		index = int64(sw.sigMap[handle])
	}
	return write(buf, index)
}

func (sw *symbolWriter) writeUntypedFunctionSignature(buf *bytes.Buffer, sig model.UntypedFunctionSignature) error {
	if err := write(buf, sig.HasRest); err != nil {
		return err
	}
	if err := write(buf, int64(len(sig.ParamNames))); err != nil {
		return err
	}
	for i, name := range sig.ParamNames {
		if err := sw.writeStringCP(buf, name); err != nil {
			return err
		}
		var flag model.ParamFlag
		if i < len(sig.ParamFlags) {
			flag = sig.ParamFlags[i]
		}
		if err := write(buf, uint8(flag)); err != nil {
			return err
		}
		var dp *model.DefaultableParam
		if i < len(sig.Default) {
			dp = sig.Default[i]
		}
		if err := write(buf, dp != nil); err != nil {
			return err
		}
		if dp != nil {
			if err := write(buf, uint8(dp.Kind)); err != nil {
				return err
			}
			if dp.Kind != model.DefaultableParamKindInferredTypedesc {
				if err := sw.writeSymbolRef(buf, dp.Symbol); err != nil {
					return err
				}
			}
		}
		metadata := sig.IncludedRecordMetadata[i]
		if err := write(buf, metadata != nil); err != nil {
			return err
		}
		if metadata == nil {
			continue
		}
		if err := write(buf, metadata.IsOpen); err != nil {
			return err
		}
		nFields := len(metadata.RequiredFields) + len(metadata.NeverFields)
		if err := write(buf, int64(nFields)); err != nil {
			return err
		}
		for _, name := range metadata.RequiredFields {
			if err := sw.writeStringCP(buf, name); err != nil {
				return err
			}
			if err := sw.writeType(buf, semtypes.VAL); err != nil {
				return err
			}
		}
		for _, name := range metadata.NeverFields {
			if err := sw.writeStringCP(buf, name); err != nil {
				return err
			}
			if err := sw.writeType(buf, semtypes.NEVER); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sw *symbolWriter) writeDefaultableParams(buf *bytes.Buffer, info *model.DefaultableParamInfo, paramCount int) error {
	var defaults []int
	for i := 0; i < paramCount; i++ {
		if _, ok := info.Get(i); ok {
			defaults = append(defaults, i)
		}
	}
	if err := write(buf, int64(len(defaults))); err != nil {
		return err
	}
	for _, idx := range defaults {
		if err := write(buf, int64(idx)); err != nil {
			return err
		}
		param, _ := info.Get(idx)
		if err := write(buf, uint8(param.Kind)); err != nil {
			return err
		}
		if param.Kind == model.DefaultableParamKindInferredTypedesc {
			continue
		}
		if err := sw.writeSymbolRef(buf, param.Symbol); err != nil {
			return err
		}
	}
	return nil
}

func (sw *symbolWriter) writeIncludedRecordParams(buf *bytes.Buffer, info *model.IncludedRecordParamInfo, paramCount int) error {
	var included []int
	if info != nil {
		for i := 0; i < paramCount; i++ {
			if info.IsIncluded(i) {
				included = append(included, i)
			}
		}
	}
	if err := write(buf, int64(len(included))); err != nil {
		return err
	}
	for _, idx := range included {
		if err := write(buf, int64(idx)); err != nil {
			return err
		}
		fields := info.Fields(idx)
		if err := write(buf, int64(len(fields))); err != nil {
			return err
		}
		for _, name := range fields {
			if err := sw.writeStringCP(buf, name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sw *symbolWriter) writeStringCP(buf *bytes.Buffer, s string) error {
	return write(buf, sw.cp.addString(s))
}

func (sw *symbolWriter) writeType(buf *bytes.Buffer, ty semtypes.SemType) error {
	if semtypes.IsZero(ty) {
		return write(buf, int32(-1))
	}
	return write(buf, int32(sw.tp.Put(ty)))
}

func (sw *symbolWriter) writeObjectDefinitionType(buf *bytes.Buffer, ty semtypes.SemType) error {
	if semtypes.IsZero(ty) {
		return write(buf, int32(-1))
	}
	return write(buf, int32(sw.tp.PutObjectDefinition(ty)))
}

func (sw *symbolWriter) writeErrorDefinitionType(buf *bytes.Buffer, ty semtypes.SemType) error {
	if semtypes.IsZero(ty) {
		return write(buf, int32(-1))
	}
	return write(buf, int32(sw.tp.PutErrorDefinition(ty)))
}
