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

	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
)

type symbolReader struct {
	r               *bytes.Reader
	cp              []string
	tp              *semtypes.TypePool
	env             *context.CompilerEnvironment
	externalRefKeys []serializedSymbolRefKey
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
	typedSig, untypedSig := sr.readFunctionSignatureBody(space)
	rm := model.NewResourceMethodSymbol(name, methodName, isPublic)
	rm.SetType(ty)
	rm.SetTypedSignature(typedSig)
	rm.SetPathListType(pathType)
	ref := addDeserializedSymbol(space, name, rm)
	sr.env.SetFunctionSignature(ref, untypedSig)
}

func addDeserializedSymbol(space *model.SymbolSpace, name string, sym model.Symbol) model.SymbolRef {
	if !sym.IsPublic() {
		if _, exists := space.GetSymbol(name); exists {
			idx := space.AppendSymbol(sym)
			return space.RefAt(idx)
		}
	}
	space.AddSymbol(name, sym)
	ref, _ := space.GetSymbol(name)
	return ref
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
	case symTagValue:
		sr.readValueSymbol(space)
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
	sym := model.NewTypeSymbol(name, isPublic)
	sym.SetType(ty)
	_ = sr.readInclusionMembers(space)
	addDeserializedSymbol(space, name, &sym)
}

func (sr *symbolReader) readRecordSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	sym := model.NewRecordSymbol(name, isPublic)
	sym.SetType(ty)
	for _, m := range sr.readInclusionMembers(space) {
		sym.AddMember(m)
	}
	addDeserializedSymbol(space, name, &sym)
}

func (sr *symbolReader) readObjectTypeSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	sym := model.NewObjectTypeSymbol(name, isPublic)
	sym.SetType(ty)
	for _, m := range sr.readInclusionMembers(space) {
		sym.AddMember(m)
	}
	ids := sr.readDistinctTypes(space)
	sym.SetDistinctTypeIDs(ids)
	sym.SetType(addObjectDistinctAtoms(ty, ids))
	addDeserializedSymbol(space, name, &sym)
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

func addObjectDistinctAtoms(ty semtypes.SemType, ids []int) semtypes.SemType {
	if semtypes.IsZero(ty) {
		return ty
	}
	for _, id := range ids {
		ty = semtypes.Intersect(ty, semtypes.ObjectDefinitionDistinct(id))
	}
	return ty
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
		sym = model.NewNetworkClassSymbol(name, isPublic)
	} else {
		sym = model.NewClassSymbol(name, isPublic)
	}
	sym.SetType(ty)
	methods := make(map[string]model.SymbolRef)
	for _, m := range sr.readInclusionMembers(space) {
		sym.AddMember(m)
		if md, ok := m.(*model.MethodDescriptor); ok {
			methods[md.MemberName()] = md.MethodRef
		}
	}
	ids := sr.readDistinctTypes(space)
	sym.SetDistinctTypeIDs(ids)
	sym.SetType(addObjectDistinctAtoms(ty, ids))
	sym.SetMethods(methods)
	if isNetwork {
		var rmCount int64
		read(sr.r, &rmCount)
		networkSym := sym.(*model.NetworkClassSymbol)
		for i := int64(0); i < rmCount; i++ {
			networkSym.AddResourceMethod(sr.readSymbolRef(space))
		}
	}
	addDeserializedSymbol(space, name, sym)
}

func (sr *symbolReader) readValueSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()
	var isConst, isParameter, isFinal, isConfigurable, isIsolated bool
	read(sr.r, &isConst)
	read(sr.r, &isParameter)
	read(sr.r, &isFinal)
	read(sr.r, &isConfigurable)
	read(sr.r, &isIsolated)
	sym := model.NewValueSymbol(name, isPublic, isConst, isParameter)
	sym.SetType(ty)
	if isFinal {
		sym.SetFinal()
	}
	if isConfigurable {
		sym.SetConfigurable()
	}
	if isIsolated {
		sym.SetIsolated()
	}
	addDeserializedSymbol(space, name, &sym)
}

func (sr *symbolReader) readFunctionSymbol(space *model.SymbolSpace) {
	name, isPublic, ty := sr.readSymbolBase()

	typedSig, untypedSig := sr.readFunctionSignatureBody(space)
	sym := model.NewFunctionSymbol(name, typedSig, isPublic)
	sym.SetType(ty)
	ref := addDeserializedSymbol(space, name, sym)
	sr.env.SetFunctionSignature(ref, untypedSig)
}

func (sr *symbolReader) readFunctionSignatureBody(space *model.SymbolSpace) (model.TypedFunctionSignature, model.UntypedFunctionSignature) {
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
	return typedSig, sr.readUntypedFunctionSignature(space)
}

func (sr *symbolReader) readUntypedFunctionSignature(space *model.SymbolSpace) model.UntypedFunctionSignature {
	var hasRest bool
	read(sr.r, &hasRest)
	var paramCount int64
	read(sr.r, &paramCount)
	paramNames := make([]string, paramCount)
	flags := make([]model.ParamFlag, paramCount)
	defaults := make([]*model.DefaultableParam, paramCount)
	included := make([]*model.IncludedRecordMetadata, paramCount)
	for i := int64(0); i < paramCount; i++ {
		paramNames[i] = sr.readStringCP()
		var flag uint8
		read(sr.r, &flag)
		flags[i] = model.ParamFlag(flag)
		var hasDefault bool
		read(sr.r, &hasDefault)
		if hasDefault {
			var kind uint8
			read(sr.r, &kind)
			defaults[i] = &model.DefaultableParam{Kind: model.DefaultableParamKind(kind)}
			if defaults[i].Kind != model.DefaultableParamKindInferredTypedesc {
				defaults[i].Symbol = sr.readSymbolRef(space)
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
		included[i] = metadata
	}
	return model.NewUntypedFunctionSignature(paramNames, hasRest, flags, defaults, included)
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

	sym := model.NewDependentlyTypedFunctionSymbol(name, model.FuncSymbolFlags(flags), isPublic)
	sym.SetParamTypes(paramTypes)
	untypedSig := sr.readUntypedFunctionSignature(space)
	sym.SetReturnType(sr.readTypeOp())
	ref := addDeserializedSymbol(space, name, sym)
	sr.env.SetFunctionSignature(ref, untypedSig)
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
