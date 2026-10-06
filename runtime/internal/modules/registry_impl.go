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

package modules

import (
	"sync"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// ClassTemplate holds the per-class data that is identical for every instance
// of a class and never mutated after construction: the method-key map and the
// resource table. It is computed once when a class is registered and shared by
// reference across all instances, so object creation only allocates the
// per-instance field map. FieldCount pre-sizes that map.
type ClassTemplate struct {
	MethodKeys  map[string]string
	RTable      map[string][]values.ResourceEntry
	Annotations values.AnnotationValues
	FieldCount  int
}

type Registry struct {
	birFunctions        map[string]*bir.BIRFunction
	functionDescriptors map[string]*bir.BIRFunction
	classTemplates      map[string]*ClassTemplate
	nativeFunctions     map[string]*ExternFunction
	runtimeBuiltins     map[string]extern.NativeFunc
	modules             map[string]*BIRModule
	recordTypes         recordTypeTable
}

// recordTypeTable holds the record types of the loaded modules. It is guarded
// by a mutex because a module's init can start strands that construct records
// while a later module is still being registered.
type recordTypeTable struct {
	mu           sync.RWMutex
	withDefaults []*recordType
	// byAtom maps a mapping atom to its record type (one without defaults if
	// there is none). Atoms are not shared across separately deserialized
	// modules, so a miss falls back to a structural search of withDefaults.
	byAtom map[*semtypes.MappingAtomicType]*recordType
}

type recordType struct {
	ty semtypes.SemType
	// fieldDefaults maps a record field name to the lookup key of the
	// function computing its default value.
	fieldDefaults map[string]string
}

func NewRegistry(builtins map[string]extern.NativeFunc) *Registry {
	return &Registry{
		birFunctions:        make(map[string]*bir.BIRFunction),
		functionDescriptors: make(map[string]*bir.BIRFunction),
		classTemplates:      make(map[string]*ClassTemplate),
		nativeFunctions:     make(map[string]*ExternFunction),
		runtimeBuiltins:     builtins,
		modules:             make(map[string]*BIRModule),
		recordTypes:         recordTypeTable{byAtom: make(map[*semtypes.MappingAtomicType]*recordType)},
	}
}

// buildClassTemplate converts a class definition's methods, resources, and
// annotations into the shared, read-only form used by every instance.
func buildClassTemplate(def *bir.BIRClassDef) *ClassTemplate {
	methodKeys := make(map[string]string, len(def.VTable))
	for methodName, method := range def.VTable {
		methodKeys[methodName] = method.FunctionLookupKey
	}
	rtable := make(map[string][]values.ResourceEntry, len(def.RTable))
	for methodName, entries := range def.RTable {
		copied := make([]values.ResourceEntry, len(entries))
		for i, entry := range entries {
			segs := make([]values.ResourcePathSegmentDef, len(entry.PathSegments))
			for j, seg := range entry.PathSegments {
				segs[j] = values.ResourcePathSegmentDef{Ty: seg.Ty}
			}
			copied[i] = values.ResourceEntry{
				PathSegments:      segs,
				RestSegmentTy:     entry.RestSegmentTy,
				FunctionLookupKey: entry.Fn.FunctionLookupKey,
			}
		}
		rtable[methodName] = copied
	}
	annotations := def.Annotations
	if annotations == nil {
		annotations = values.NewAnnotationValues()
	}
	return &ClassTemplate{
		MethodKeys:  methodKeys,
		RTable:      rtable,
		Annotations: annotations,
		FieldCount:  len(def.Fields),
	}
}

func moduleKey(pkgId *model.PackageID) string {
	return pkgId.OrgName.Value() + "/" + pkgId.PkgName.Value()
}

func (r *Registry) RegisterModule(id *model.PackageID, m *BIRModule) *BIRModule {
	if m.Pkg != nil {
		for i := range m.Pkg.Functions {
			fn := &m.Pkg.Functions[i]
			r.registerFunctionDescriptor(fn)
		}
		for i := range m.Pkg.ClassDefs {
			classDef := &m.Pkg.ClassDefs[i]
			r.classTemplates[classDef.LookupKey] = buildClassTemplate(classDef)
			for _, fn := range classDef.VTable {
				r.registerFunctionDescriptor(fn)
			}
			for _, entries := range classDef.RTable {
				for i := range entries {
					r.registerFunctionDescriptor(entries[i].Fn)
				}
			}
		}
	}
	if id != nil && !id.IsUnnamed() {
		r.modules[moduleKey(id)] = m
	}
	return m
}

// RegisterRecordTypes records the record types a module declares.
func (r *Registry) RegisterRecordTypes(tc semtypes.Context, recordTypes []bir.BIRRecordType) {
	table := &r.recordTypes
	table.mu.Lock()
	defer table.mu.Unlock()
	for _, def := range recordTypes {
		atom := semtypes.ToMappingAtomicType(tc, def.Type)
		if atom == nil {
			continue
		}
		recordTy := newRecordType(def)
		if len(recordTy.fieldDefaults) > 0 {
			table.withDefaults = append(table.withDefaults, recordTy)
		}
		table.byAtom[atom] = recordTy
	}
}

func newRecordType(def bir.BIRRecordType) *recordType {
	fieldDefaults := make(map[string]string, len(def.FieldDefaults))
	for _, field := range def.FieldDefaults {
		fieldDefaults[field.FieldName] = field.FunctionLookupKey
	}
	return &recordType{ty: def.Type, fieldDefaults: fieldDefaults}
}

// RecordFieldDefault returns the lookup key of the default function of field in
// the record type recordTy, whose mapping atom is atom.
func (r *Registry) RecordFieldDefault(tc semtypes.Context, recordTy semtypes.SemType, atom *semtypes.MappingAtomicType, field string) (string, bool) {
	key, ok := r.recordTypes.lookup(tc, recordTy, atom).fieldDefaults[field]
	return key, ok
}

func (t *recordTypeTable) lookup(tc semtypes.Context, ty semtypes.SemType, atom *semtypes.MappingAtomicType) *recordType {
	t.mu.RLock()
	recordTy, ok := t.byAtom[atom]
	t.mu.RUnlock()
	if ok {
		return recordTy
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if recordTy, ok := t.byAtom[atom]; ok {
		return recordTy
	}
	recordTy = t.findStructurally(tc, ty)
	t.byAtom[atom] = recordTy
	return recordTy
}

func (t *recordTypeTable) findStructurally(tc semtypes.Context, ty semtypes.SemType) *recordType {
	for _, recordTy := range t.withDefaults {
		if semtypes.IsSameType(tc, recordTy.ty, ty) {
			return recordTy
		}
	}
	return &recordType{ty: ty, fieldDefaults: make(map[string]string)}
}

func (r *Registry) registerFunctionDescriptor(fn *bir.BIRFunction) {
	r.functionDescriptors[fn.FunctionLookupKey] = fn
	if !fn.Flags.Has(model.FlagNative) {
		r.birFunctions[fn.FunctionLookupKey] = fn
	}
}

func (r *Registry) GetModule(pkgId *model.PackageID) *BIRModule {
	return r.modules[moduleKey(pkgId)]
}

func (r *Registry) GetModuleByName(orgName, moduleName string) *BIRModule {
	return r.modules[orgName+"/"+moduleName]
}

func (r *Registry) RegisterExternFunction(orgName string, moduleName string, funcName string, impl extern.NativeFunc) {
	externFn := &ExternFunction{
		Name: funcName,
		Impl: impl,
	}
	moduleKey := orgName + "/" + moduleName
	qualifiedName := moduleKey + ":" + funcName
	r.nativeFunctions[qualifiedName] = externFn
	r.nativeFunctions[funcName] = externFn
}

// GetClassTemplate returns the shared, precomputed method/resource tables for a
// class. The returned maps are read-only and shared across every instance.
func (r *Registry) GetClassTemplate(lookupKey string) *ClassTemplate {
	return r.classTemplates[lookupKey]
}

// RegisterExternClassDef registers a synthetic BIRClassDef so that execNewObject
// can build method-key maps for Go-declared classes. VTable entries are intentionally
// NOT added to birFunctions so that exec falls through to nativeFunctions for dispatch.
func (r *Registry) RegisterExternClassDef(def *bir.BIRClassDef) {
	r.classTemplates[def.LookupKey] = buildClassTemplate(def)
	for _, fn := range def.VTable {
		r.functionDescriptors[fn.FunctionLookupKey] = fn
	}
	for _, entries := range def.RTable {
		for i := range entries {
			fn := entries[i].Fn
			r.functionDescriptors[fn.FunctionLookupKey] = fn
		}
	}
}

func (r *Registry) GetBIRFunction(funcName string) *bir.BIRFunction {
	return r.birFunctions[funcName]
}

func (r *Registry) GetFunctionDescriptor(funcName string) *bir.BIRFunction {
	return r.functionDescriptors[funcName]
}

func (r *Registry) GetNativeFunction(funcName string) *ExternFunction {
	return r.nativeFunctions[funcName]
}

func (r *Registry) GetRuntimeBuiltin(lookupKey string) extern.NativeFunc {
	return r.runtimeBuiltins[lookupKey]
}
