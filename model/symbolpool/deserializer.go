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
	"strings"

	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/values"
)

type symbolReader struct {
	r               *bytes.Reader
	cp              []string
	tp              *semtypes.TypePool
	env             *context.CompilerEnvironment
	externalRefKeys []serializedSymbolRefKey
	sigHandles      []context.FunctionSignatureHandle
}

func Unmarshal(env *context.CompilerEnvironment, data []byte) (model.ExportedSymbolSpace, error) {
	sr := &symbolReader{
		r:   bytes.NewReader(data),
		env: env,
	}
	return sr.deserialize()
}

func (sr *symbolReader) deserialize() (result model.ExportedSymbolSpace, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = model.ExportedSymbolSpace{}
			err = fmt.Errorf("symbol deserializer failed: %v", r)
		}
	}()

	magic := make([]byte, 4)
	_, err = sr.r.Read(magic)
	if err != nil {
		panic(fmt.Sprintf("reading magic: %v", err))
	}
	if string(magic) != symMagic {
		panic(fmt.Sprintf("invalid symbol magic: %x", magic))
	}

	var version int32
	read(sr.r, &version)
	if version != symVersion {
		panic(fmt.Sprintf("unsupported symbol version: %d", version))
	}

	var tpSize int64
	read(sr.r, &tpSize)
	tpBytes := make([]byte, tpSize)
	_, err = sr.r.Read(tpBytes)
	if err != nil {
		panic(fmt.Sprintf("reading type pool: %v", err))
	}
	sr.tp = semtypes.UnmarshalTypePool(tpBytes, sr.env.GetTypeEnv())

	sr.cp = deserializeConstantPool(sr.r)
	sr.externalRefKeys = sr.readExternalSymbolRefPool()

	mainSpace := sr.readSymbolSpace()
	annotationSpace := sr.readSymbolSpace()

	return model.NewExportedSymbolSpaces([]*model.SymbolSpace{mainSpace}, []*model.SymbolSpace{annotationSpace}), nil
}

func (sr *symbolReader) readResourceMethodSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	methodName := sr.readStringCP()
	pathType := sr.readType()
	typedSig, sigHandle := sr.readFunctionSignatureBody(space)
	rm := model.NewResourceMethodSymbol(name, methodName, isPublic, diagnostics.NewBuiltinLocation())
	rm.SetType(ty)
	rm.SetTypedSignature(typedSig)
	rm.SetPathListType(pathType)
	ref := addDeserializedSymbol(space, name, rm)
	if sigHandle >= 0 {
		sr.env.AssociateFunctionSignature(ref, sr.sigHandles[sigHandle])
	}
}

func addDeserializedSymbol(space *model.SymbolSpace, name string, sym model.Symbol) model.SymbolRef {
	if !sym.IsPublic() {
		if _, exists := space.GetSymbol(name); exists {
			return space.RefAt(space.AppendSymbol(sym))
		}
	}
	space.AddSymbol(name, sym)
	ref, _ := space.GetSymbol(name)
	return ref
}

// storeAnnotations records deserialized annotation values on the compiler
// environment, keyed by the symbol's ref (annotations no longer live on the
// symbol itself).
func (sr *symbolReader) storeAnnotations(ref model.SymbolRef, annotations values.AnnotationValues) {
	for key, value := range annotations {
		sr.env.SetSymbolAnnotationValue(ref, key, value)
	}
}

func (sr *symbolReader) readPackageIdentifier() *model.PackageID {
	org := sr.readStringCP()
	pkg := sr.readStringCP()
	version := sr.readStringCP()
	nameComps := model.CreateNameComps(model.Name(pkg))
	versionName := model.Name(version)
	if versionName == "" {
		versionName = model.DEFAULT_VERSION
	}
	return sr.env.NewPackageID(model.Name(org), nameComps, versionName)
}

func (sr *symbolReader) readSymbolSpace() *model.SymbolSpace {
	var count int64
	read(sr.r, &count)
	if count == symbolSpaceNilSentinel {
		return nil
	}

	pkgID := sr.readPackageIdentifier()
	space := sr.env.NewSymbolSpace(*pkgID)
	sr.readFunctionSignatureTable(space)
	opaque := model.OpaqueSymbols(space.Pkg)
	for i := int64(0); i < count; i++ {
		sr.readSymbol(space, opaque)
	}

	return space
}

func (sr *symbolReader) readSymbol(space *model.SymbolSpace, opaque []model.Symbol) {
	var tag uint8
	read(sr.r, &tag)

	switch tag {
	case symTagOpaque:
		var idx int32
		read(sr.r, &idx)
		sym := opaque[idx]
		// Set the space the (monomorphized) function is added to. No
		// monomorphization cache is installed here; nil closures mean no
		// caching (the symbol resolver installs one when compiling from source).
		if fn, ok := sym.(*model.OpaqueFunctionSymbol); ok {
			fn.SymbolSpace = space
		}
		space.AddSymbol(sym.Name(), sym)
	case symTagType:
		sr.readTypeSymbol(space)
	case symTagClass:
		sr.readClassSymbol(space, false)
	case symTagNetworkClass:
		sr.readClassSymbol(space, true)
	case symTagRecord:
		sr.readRecordSymbol(space)
	case symTagObjectType:
		sr.readObjectTypeSymbol(space)
	case symTagErrorType:
		sr.readErrorTypeSymbol(space)
	case symTagValue:
		sr.readValueSymbol(space)
	case symTagConstantValue:
		sr.readConstantValueSymbol(space)
	case symTagAnnotation:
		sr.readAnnotationSymbol(space)
	case symTagFunction:
		sr.readFunctionSymbol(space)
	case symTagDependentlyTypedFunction:
		sr.readDependentlyTypedFunctionSymbol(space)
	case symTagResourceMethod:
		sr.readResourceMethodSymbol(space)
	default:
		panic(fmt.Sprintf("unknown symbol tag: %d", tag))
	}
}

func (sr *symbolReader) readSymbolBase() (name string, isPublic bool, ty semtypes.SemType) {
	name = sr.readStringCP()
	read(sr.r, &isPublic)
	ty = sr.readType()
	return
}

func (sr *symbolReader) readTypeSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	sym := model.NewTypeSymbol(name, isPublic, diagnostics.NewBuiltinLocation())
	sym.SetType(ty)
	annotations := sr.readAnnotationValues()
	_ = sr.readInclusionMembers(space)
	ref := addDeserializedSymbol(space, name, &sym)
	sr.storeAnnotations(ref, annotations)
}

func (sr *symbolReader) readRecordSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	sym := model.NewRecordSymbol(name, isPublic, diagnostics.NewBuiltinLocation())
	sym.SetType(ty)
	annotations := sr.readAnnotationValues()
	for _, m := range sr.readInclusionMembers(space) {
		sym.AddMember(m)
	}
	ref := addDeserializedSymbol(space, name, &sym)
	sr.storeAnnotations(ref, annotations)
}

func (sr *symbolReader) readObjectTypeSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	sym := model.NewObjectTypeSymbol(name, isPublic, diagnostics.NewBuiltinLocation())
	sym.SetType(ty)
	annotations := sr.readAnnotationValues()
	for _, m := range sr.readInclusionMembers(space) {
		sym.AddMember(m)
	}
	ids := sr.readDistinctTypes(space)
	sym.SetDistinctTypeIDs(ids)
	sym.SetType(intersectDistinctAtoms(ty, ids, semtypes.ObjectDefinitionDistinct))
	ref := addDeserializedSymbol(space, name, &sym)
	sr.storeAnnotations(ref, annotations)
	sr.registerLangLibDistinctTypeSymbol(space, name, ref, ids)
}

func (sr *symbolReader) readErrorTypeSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	sym := model.NewErrorTypeSymbol(name, isPublic, diagnostics.NewBuiltinLocation())
	sym.SetType(ty)
	annotations := sr.readAnnotationValues()
	ids := sr.readDistinctTypes(space)
	sym.SetDistinctTypeIDs(ids)
	sym.SetType(intersectDistinctAtoms(ty, ids, semtypes.ErrorDistinct))
	ref := addDeserializedSymbol(space, name, &sym)
	sr.storeAnnotations(ref, annotations)
	sr.registerLangLibDistinctTypeSymbol(space, name, ref, ids)
}

func (sr *symbolReader) readDistinctTypes(space *model.SymbolSpace) []int {
	var count int64
	read(sr.r, &count)
	ids := make([]int, 0, count)
	for i := int64(0); i < count; i++ {
		ref := sr.readSymbolRef(space)
		ids = append(ids, sr.env.DistinctTypeID(ref))
	}
	return ids
}

func intersectDistinctAtoms(ty semtypes.SemType, ids []int, atom func(int) semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(ty) {
		return ty
	}
	for _, id := range ids {
		ty = semtypes.Intersect(ty, atom(id))
	}
	return ty
}

func (sr *symbolReader) registerLangLibDistinctTypeSymbol(space *model.SymbolSpace, name string, ref model.SymbolRef, ids []int) {
	if space.Pkg.Organization != "ballerina" || !strings.HasPrefix(space.Pkg.Package, "lang.") {
		return
	}
	for _, id := range ids {
		distinctRef, ok := sr.env.DistinctTypeSymbolRef(id)
		if ok && distinctRef == ref {
			if !sr.env.RegisterLangLibDistinctTypeSymbol(space.Pkg.Package, name, ref) {
				panic(fmt.Sprintf("conflicting lang library distinct type symbol: %s:%s", space.Pkg.Package, name))
			}
			return
		}
	}
}

func (sr *symbolReader) readInclusionMembers(space *model.SymbolSpace) []model.InclusionMember {
	var count int64
	read(sr.r, &count)
	members := make([]model.InclusionMember, 0, count)
	for i := int64(0); i < count; i++ {
		var tag uint8
		read(sr.r, &tag)
		switch tag {
		case inclusionMemberTagField:
			name := sr.readStringCP()
			ty := sr.readType()
			var isPublic bool
			read(sr.r, &isPublic)
			var flags uint8
			read(sr.r, &flags)
			var fdFlags model.FieldDescriptorFlag
			if flags&1 != 0 {
				fdFlags |= model.FieldDescriptorReadonly
			}
			if flags&2 != 0 {
				fdFlags |= model.FieldDescriptorOptional
			}
			if flags&4 != 0 {
				fdFlags |= model.FieldDescriptorHasDefault
			}
			fd := model.NewFieldDescriptor(name, fdFlags, isPublic)
			fd.SetMemberType(ty)
			fd.DefaultFnRef = sr.readSymbolRef(space)
			members = append(members, &fd)
		case inclusionMemberTagMethod:
			name := sr.readStringCP()
			ty := sr.readType()
			var kind uint8
			read(sr.r, &kind)
			var isPublic bool
			read(sr.r, &isPublic)
			methodRef := sr.readSymbolRef(space)
			md := model.NewMethodDescriptor(name, model.InclusionMemberKind(kind), isPublic, methodRef)
			md.SetMemberType(ty)
			members = append(members, &md)
		case inclusionMemberTagRestType:
			ty := sr.readType()
			rd := model.NewRestTypeDescriptor()
			rd.SetMemberType(ty)
			members = append(members, &rd)
		}
	}
	return members
}

func (sr *symbolReader) readSymbolRef(space *model.SymbolSpace) model.SymbolRef {
	var tag uint8
	read(sr.r, &tag)
	switch tag {
	case symbolRefTagEmpty:
		return model.SymbolRef{}
	case symbolRefTagLocal:
		var index int32
		read(sr.r, &index)
		return model.SymbolRef{
			Index:      int(index),
			SpaceIndex: space.SpaceIndex(),
		}
	case symbolRefTagExternal:
		var index int32
		read(sr.r, &index)
		key := sr.externalRefKeys[index]
		ref, ok := sr.env.FindSymbol(key.pkg, key.name)
		if !ok {
			panic(fmt.Sprintf("external symbol not found: %s/%s:%s", key.pkg.Organization, key.pkg.Package, key.name))
		}
		return ref
	default:
		panic(fmt.Sprintf("unknown symbol ref tag: %d", tag))
	}
}

func (sr *symbolReader) readExternalSymbolRefPool() []serializedSymbolRefKey {
	var count int64
	read(sr.r, &count)
	keys := make([]serializedSymbolRefKey, count)
	for i := range keys {
		pkgID := sr.readPackageIdentifier()
		keys[i] = serializedSymbolRefKey{
			pkg:  model.PackageIdentifierFromID(pkgID),
			name: sr.readStringCP(),
		}
	}
	return keys
}

func (sr *symbolReader) readClassSymbol(space *model.SymbolSpace, isNetwork bool) {
	name, isPublic, ty := sr.readSymbolBase()
	var sym model.ClassSymbol
	if isNetwork {
		sym = model.NewNetworkClassSymbol(name, isPublic, diagnostics.NewBuiltinLocation())
	} else {
		sym = model.NewClassSymbol(name, isPublic, diagnostics.NewBuiltinLocation())
	}
	sym.SetType(ty)
	annotations := sr.readAnnotationValues()
	methods := make(map[string]model.SymbolRef)
	for _, m := range sr.readInclusionMembers(space) {
		sym.AddMember(m)
		if md, ok := m.(*model.MethodDescriptor); ok {
			methods[md.MemberName()] = md.MethodRef
		}
	}
	ids := sr.readDistinctTypes(space)
	sym.SetDistinctTypeIDs(ids)
	sym.SetType(intersectDistinctAtoms(ty, ids, semtypes.ObjectDefinitionDistinct))
	sym.SetMethods(methods)
	if isNetwork {
		var rmCount int64
		read(sr.r, &rmCount)
		networkSym := sym.(*model.NetworkClassSymbol)
		for i := int64(0); i < rmCount; i++ {
			networkSym.AddResourceMethod(sr.readSymbolRef(space))
		}
	}
	ref := addDeserializedSymbol(space, name, sym)
	sr.storeAnnotations(ref, annotations)
	sr.registerLangLibDistinctTypeSymbol(space, name, ref, ids)
}

type valueSymbolFields struct {
	name           string
	isPublic       bool
	ty             semtypes.SemType
	isConst        bool
	isParameter    bool
	isFinal        bool
	isConfigurable bool
	isIsolated     bool
}

func (sr *symbolReader) readValueSymbolFields() valueSymbolFields {
	f := valueSymbolFields{}
	f.name, f.isPublic, f.ty = sr.readSymbolBase()
	read(sr.r, &f.isConst)
	read(sr.r, &f.isParameter)
	read(sr.r, &f.isFinal)
	read(sr.r, &f.isConfigurable)
	read(sr.r, &f.isIsolated)
	return f
}

func applyValueSymbolFields(sym *model.VariableSymbol, f valueSymbolFields) {
	sym.SetType(f.ty)
	if f.isFinal {
		sym.SetFinal()
	}
	if f.isConfigurable {
		sym.SetConfigurable()
	}
	if f.isIsolated {
		sym.SetIsolated()
	}
}

func (sr *symbolReader) readValueSymbol(space *model.SymbolSpace) {
	f := sr.readValueSymbolFields()
	sym := model.NewVariableSymbol(f.name, f.isPublic, f.isConst, f.isParameter, diagnostics.NewBuiltinLocation())
	applyValueSymbolFields(&sym, f)
	addDeserializedSymbol(space, f.name, &sym)
}

func (sr *symbolReader) readConstantValueSymbol(space *model.SymbolSpace) {
	f := sr.readValueSymbolFields()
	sym := model.NewConstantValueSymbol(f.name, f.isPublic, diagnostics.NewBuiltinLocation())
	applyValueSymbolFields(&sym.VariableSymbol, f)
	sym.SetConstantValue(sr.readAnnotationValue())
	addDeserializedSymbol(space, f.name, sym)
}

func (sr *symbolReader) readAnnotationSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	var isConst bool
	read(sr.r, &isConst)
	var count int64
	read(sr.r, &count)
	attachPoints := make([]string, 0, count)
	for i := int64(0); i < count; i++ {
		attachPoints = append(attachPoints, sr.readStringCP())
	}
	sym := model.NewAnnotationSymbol(name, isPublic, isConst, attachPoints, diagnostics.NewBuiltinLocation())
	sym.SetType(ty)
	addDeserializedSymbol(space, name, &sym)
}

func (sr *symbolReader) readFunctionSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()

	typedSig, sigHandle := sr.readFunctionSignatureBody(space)
	sym := model.NewFunctionSymbol(name, typedSig, isPublic, diagnostics.NewBuiltinLocation())
	sym.SetType(ty)
	ref := addDeserializedSymbol(space, name, sym)
	if sigHandle >= 0 {
		sr.env.AssociateFunctionSignature(ref, sr.sigHandles[sigHandle])
	}
}

func (sr *symbolReader) readFunctionSignatureBody(space *model.SymbolSpace) (model.TypedFunctionSignature, int64) {
	var paramCount int64
	read(sr.r, &paramCount)
	paramTypes := make([]semtypes.SemType, paramCount)
	for i := int64(0); i < paramCount; i++ {
		paramTypes[i] = sr.readType()
	}
	returnType := sr.readType()
	var hasRestParam bool
	read(sr.r, &hasRestParam)
	var restParamType semtypes.SemType
	if hasRestParam {
		restParamType = sr.readType()
	}
	var flags uint8
	read(sr.r, &flags)
	typedSig := model.TypedFunctionSignature{
		ParamTypes:    paramTypes,
		ReturnType:    returnType,
		RestParamType: restParamType,
		Flags:         model.FuncSymbolFlags(flags),
	}
	var sigHandle int64
	read(sr.r, &sigHandle)
	return typedSig, sigHandle
}

func (sr *symbolReader) readFunctionSignatureTable(space *model.SymbolSpace) {
	var count int64
	read(sr.r, &count)
	sr.sigHandles = make([]context.FunctionSignatureHandle, count)
	for i := int64(0); i < count; i++ {
		params, hasRest := sr.readUntypedFunctionSignatureParams(space)
		sr.sigHandles[i] = sr.env.AllocateFunctionSignature(params, hasRest)
	}
}

func (sr *symbolReader) readUntypedFunctionSignatureParams(space *model.SymbolSpace) ([]model.Param, bool) {
	var hasRest bool
	read(sr.r, &hasRest)
	var paramCount int64
	read(sr.r, &paramCount)
	params := make([]model.Param, paramCount)
	for i := int64(0); i < paramCount; i++ {
		params[i].Name = sr.readStringCP()
		var flag uint8
		read(sr.r, &flag)
		params[i].Flag = model.ParamFlag(flag)
		var hasDefault bool
		read(sr.r, &hasDefault)
		if hasDefault {
			var kind uint8
			read(sr.r, &kind)
			params[i].Default = &model.DefaultableParam{Kind: model.DefaultableParamKind(kind)}
			if params[i].Default.Kind != model.DefaultableParamKindInferredTypedesc {
				params[i].Default.Symbol = sr.readSymbolRef(space)
			}
		}
		var hasIncluded bool
		read(sr.r, &hasIncluded)
		if !hasIncluded {
			continue
		}
		metadata := &model.IncludedRecordMetadata{}
		read(sr.r, &metadata.IsOpen)
		var fieldCount int64
		read(sr.r, &fieldCount)
		for j := int64(0); j < fieldCount; j++ {
			name := sr.readStringCP()
			ty := sr.readType()
			if semtypes.IsNever(ty) {
				metadata.NeverFields = append(metadata.NeverFields, name)
			} else {
				metadata.RequiredFields = append(metadata.RequiredFields, name)
			}
		}
		params[i].IncludedRecord = metadata
	}
	return params, hasRest
}

func (sr *symbolReader) readDependentlyTypedFunctionSymbol(space *model.SymbolSpace) {
	name := sr.readStringCP()
	var isPublic bool
	read(sr.r, &isPublic)
	var paramCount int64
	read(sr.r, &paramCount)
	paramTypes := make([]semtypes.SemType, paramCount)
	for i := int64(0); i < paramCount; i++ {
		paramTypes[i] = sr.readType()
	}
	var flags uint8
	read(sr.r, &flags)

	sym := model.NewDependentlyTypedFunctionSymbol(name, model.FuncSymbolFlags(flags), isPublic, diagnostics.NewBuiltinLocation())
	sym.SetParamTypes(paramTypes)
	var sigHandle int64
	read(sr.r, &sigHandle)
	sym.SetReturnType(sr.readTypeOp())
	ref := addDeserializedSymbol(space, name, sym)
	if sigHandle >= 0 {
		sr.env.AssociateFunctionSignature(ref, sr.sigHandles[sigHandle])
	}
}

func (sr *symbolReader) readTypeOp() model.TypeOp {
	var tag uint8
	read(sr.r, &tag)
	switch tag {
	case typeOpTagIdentity:
		return &model.IdentityTypeOp{Type: sr.readType()}
	case typeOpTagRef:
		var idx int64
		read(sr.r, &idx)
		return &model.RefTypeOp{Index: int(idx)}
	case typeOpTagUnion:
		lhs := sr.readTypeOp()
		rhs := sr.readTypeOp()
		return &model.BinaryTypeOp{Kind: model.TypeOpUnion, Lhs: lhs, Rhs: rhs}
	case typeOpTagIntersect:
		lhs := sr.readTypeOp()
		rhs := sr.readTypeOp()
		return &model.BinaryTypeOp{Kind: model.TypeOpIntersection, Lhs: lhs, Rhs: rhs}
	default:
		panic(fmt.Sprintf("unknown TypeOp tag: %d", tag))
	}
}

func (sr *symbolReader) readDefaultableParams(paramCount int, space *model.SymbolSpace) model.DefaultableParamInfo {
	var count int64
	read(sr.r, &count)
	if count == 0 {
		return model.NewDefaultableParamInfo(paramCount)
	}
	info := model.NewDefaultableParamInfo(paramCount)
	for i := int64(0); i < count; i++ {
		var idx int64
		read(sr.r, &idx)
		var kind uint8
		read(sr.r, &kind)
		if model.DefaultableParamKind(kind) == model.DefaultableParamKindInferredTypedesc {
			info.SetInferredTypedesc(int(idx))
			continue
		}
		ref := sr.readSymbolRef(space)
		info.SetDefaultable(int(idx), ref)
	}
	return info
}

func (sr *symbolReader) readIncludedRecordParams(paramCount int) *model.IncludedRecordParamInfo {
	var count int64
	read(sr.r, &count)
	if count == 0 {
		return nil
	}
	info := model.NewIncludedRecordParamInfo(paramCount)
	for i := int64(0); i < count; i++ {
		var idx int64
		read(sr.r, &idx)
		info.Set(int(idx))
		var fieldCount int64
		read(sr.r, &fieldCount)
		names := make([]string, fieldCount)
		for j := int64(0); j < fieldCount; j++ {
			names[j] = sr.readStringCP()
		}
		info.SetFields(int(idx), names)
	}
	return info
}

func (sr *symbolReader) readStringCP() string {
	var idx int32
	read(sr.r, &idx)
	return sr.cp[idx]
}

func (sr *symbolReader) readType() semtypes.SemType {
	var idx int32
	read(sr.r, &idx)
	if idx == -1 {
		return semtypes.SemType{}
	}
	return sr.tp.Get(semtypes.TypePoolIndex(idx))
}
