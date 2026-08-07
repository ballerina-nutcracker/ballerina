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
	"ballerina/bir"
	"ballerina/model"
	"ballerina/runtime/extern"
	"ballerina/values"
)

// ClassTemplate holds the per-class data that is identical for every instance
// of a class and never mutated after construction: the method-key map and the
// resource table. It is computed once when a class is registered and shared by
// reference across all instances, so object creation only allocates the
// per-instance field map. FieldCount pre-sizes that map.
type ClassTemplate struct {
	MethodKeys map[string]string
	RTable     map[string][]values.ResourceEntry
	FieldCount int
}

type Registry struct {
	birFunctions    map[string]*bir.BIRFunction
	classTemplates  map[string]*ClassTemplate
	nativeFunctions map[string]*ExternFunction
	runtimeBuiltins map[string]extern.NativeFunc
	modules         map[string]*BIRModule
}

func NewRegistry(builtins map[string]extern.NativeFunc) *Registry {
	return &Registry{
		birFunctions:    make(map[string]*bir.BIRFunction),
		classTemplates:  make(map[string]*ClassTemplate),
		nativeFunctions: make(map[string]*ExternFunction),
		runtimeBuiltins: builtins,
		modules:         make(map[string]*BIRModule),
	}
}

// buildClassTemplate converts a class definition's method and resource tables
// into the shared, read-only form used by every instance of the class.
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
	return &ClassTemplate{
		MethodKeys: methodKeys,
		RTable:     rtable,
		FieldCount: len(def.Fields),
	}
}

func moduleKey(pkgId *model.PackageID) string {
	return pkgId.OrgName.Value() + "/" + pkgId.PkgName.Value()
}

func (r *Registry) RegisterModule(id *model.PackageID, m *BIRModule) *BIRModule {
	if m.Pkg != nil {
		for i := range m.Pkg.Functions {
			fn := &m.Pkg.Functions[i]
			r.birFunctions[fn.FunctionLookupKey] = fn
		}
		for i := range m.Pkg.ClassDefs {
			classDef := &m.Pkg.ClassDefs[i]
			r.classTemplates[classDef.LookupKey] = buildClassTemplate(classDef)
			for _, fn := range classDef.VTable {
				if fn.Flags.Has(model.FlagNative) {
					continue
				}
				r.birFunctions[fn.FunctionLookupKey] = fn
			}
			for _, entries := range classDef.RTable {
				for i := range entries {
					fn := entries[i].Fn
					if fn.Flags.Has(model.FlagNative) {
						continue
					}
					r.birFunctions[fn.FunctionLookupKey] = fn
				}
			}
		}
	}
	if id != nil && !id.IsUnnamed() {
		r.modules[moduleKey(id)] = m
	}
	return m
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
}

func (r *Registry) GetBIRFunction(funcName string) *bir.BIRFunction {
	return r.birFunctions[funcName]
}

func (r *Registry) GetNativeFunction(funcName string) *ExternFunction {
	return r.nativeFunctions[funcName]
}

func (r *Registry) GetRuntimeBuiltin(lookupKey string) extern.NativeFunc {
	return r.runtimeBuiltins[lookupKey]
}
