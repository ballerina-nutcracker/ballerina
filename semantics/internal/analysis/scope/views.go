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

package scope

import (
	"slices"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// FunctionInfo is a snapshot of the innermost closure.
type FunctionInfo struct {
	class    ClassInfo
	RetTy    semtypes.SemType
	symbol   model.SymbolRef
	self     model.SymbolRef
	ID       ScopeID
	kind     functionKind
	isolated bool
}

func (f FunctionInfo) IsIsolated() bool          { return f.isolated }
func (f FunctionInfo) IsModuleInit() bool        { return f.kind == fnModuleInit }
func (f FunctionInfo) IsLambda() bool            { return f.kind == fnLambda }
func (f FunctionInfo) IsExpressionClosure() bool { return f.kind == fnExpressionClosure }

// isSelf reports whether sym is the closure's self.
func (f FunctionInfo) isSelf(sym model.SymbolRef) bool { return !f.self.IsEmpty() && sym == f.self }

// isClassInit reports whether the closure is the init method of its class.
func (f FunctionInfo) isClassInit() bool { return !f.symbol.IsEmpty() && f.symbol == f.class.initFn }

// LockInfo is a snapshot of the innermost lock of the current closure.
type LockInfo struct {
	Position   diagnostics.Location
	Restricted model.SymbolRef
}

// fieldInfo is the data of a class or service field the lock and outside-lock
// rules need.
type fieldInfo struct {
	name      string
	sym       model.SymbolRef
	lockKey   string
	immutable bool
}

// ClassInfo is the data of an enclosing class or service, built once at class
// entry and never written afterwards.
type ClassInfo struct {
	fields []fieldInfo
	// fieldDescriptors describe every field the body has, both declared in
	// it and included with `*T`.
	fieldDescriptors []*model.FieldDescriptor
	initFn           model.SymbolRef
	isolated         bool
}

// NewClassInfo reads the fields and init function of a class (name non-empty)
// or service (name empty); fieldDescriptors describe its declared and
// included fields.
func NewClassInfo(cx Context, name string, isolated bool, fields []*ast.BLangVariable, fieldDescriptors []*model.FieldDescriptor,
	initFn *ast.BLangFunction) ClassInfo {
	ctx := cx.CompilerCtx()
	info := ClassInfo{isolated: isolated, fields: make([]fieldInfo, 0, len(fields)), fieldDescriptors: fieldDescriptors}
	for _, field := range fields {
		fieldName := field.Name.GetValue()
		sym := field.Symbol()
		info.fields = append(info.fields, fieldInfo{
			name:      fieldName,
			sym:       sym,
			lockKey:   fieldLockKey(ctx.SymbolPackage(sym), name, sym, fieldName),
			immutable: IsImmutableField(cx, field),
		})
	}
	slices.SortFunc(info.fields, func(a, b fieldInfo) int { return strings.Compare(a.name, b.name) })
	if initFn != nil {
		info.initFn = initFn.Symbol()
	}
	return info
}

// IsImmutableField reports whether field is final and of a readonly type.
func IsImmutableField(cx Context, field *ast.BLangVariable) bool {
	return field.IsFinal() && semtypes.IsSubtype(cx.TyCtx(), cx.CompilerCtx().SymbolType(field.Symbol()), semtypes.ValReadonly)
}

func (c ClassInfo) IsIsolated() bool { return c.isolated }

func (c ClassInfo) field(name string) (fieldInfo, bool) {
	i, ok := slices.BinarySearchFunc(c.fields, name, func(f fieldInfo, n string) int { return strings.Compare(f.name, n) })
	if !ok {
		return fieldInfo{}, false
	}
	return c.fields[i], true
}

func (c ClassInfo) fieldBySymbol(sym model.SymbolRef) (fieldInfo, bool) {
	for _, f := range c.fields {
		if f.sym == sym {
			return f, true
		}
	}
	return fieldInfo{}, false
}

func moduleVarLockKey(pkg model.PackageIdentifier, name string) string {
	return pkg.Organization + "/" + pkg.Package + ":" + name
}

// fieldLockKey is a per-field lock key. A class uses its module-level name; a
// service has no name, so the field's symbol, unique within the package,
// disambiguates it.
func fieldLockKey(pkg model.PackageIdentifier, className string, fieldRef model.SymbolRef, fieldName string) string {
	if className != "" {
		return pkg.Organization + "/" + pkg.Package + ":" + className + "." + fieldName
	}
	return pkg.Organization + "/" + pkg.Package + ":$service." + strconv.Itoa(fieldRef.SpaceIndex) + "." +
		strconv.Itoa(fieldRef.Index) + "." + fieldName
}

type varDeclFlags uint8

const (
	varFinal varDeclFlags = 1 << iota
	varConfigurable
	varIsolated
	varNoInitializer
)

// VarDecl is the declaration metadata of a module-level variable, constant,
// imported value or local.
type VarDecl struct {
	ty    semtypes.SemType
	flags varDeclFlags
}

func newVarDecl(ty semtypes.SemType, final, configurable, isolated, noInitializer bool) VarDecl {
	var flags varDeclFlags
	if final {
		flags |= varFinal
	}
	if configurable {
		flags |= varConfigurable
	}
	if isolated {
		flags |= varIsolated
	}
	if noInitializer {
		flags |= varNoInitializer
	}
	return VarDecl{ty: ty, flags: flags}
}

func (v VarDecl) Type() semtypes.SemType { return v.ty }
func (v VarDecl) IsFinal() bool          { return v.flags&varFinal != 0 }
func (v VarDecl) isConfigurable() bool   { return v.flags&varConfigurable != 0 }
func (v VarDecl) IsIsolated() bool       { return v.flags&varIsolated != 0 }
func (v VarDecl) HasNoInitializer() bool { return v.flags&varNoInitializer != 0 }

type lvalueKind uint8

const (
	lvMember lvalueKind = iota
	lvPlain
	lvWildcard
)

// LValue is an assignment target, built by the target validators one step per
// target node.
type LValue struct {
	root model.SymbolRef
	kind lvalueKind
}

// TargetVar is a plain variable target.
func TargetVar(ref *ast.BLangVarRef) LValue {
	return LValue{root: ref.Symbol(), kind: lvPlain}
}

// TargetWildcard is the `_` target.
func TargetWildcard(_ *ast.BLangWildCardBindingPattern) LValue {
	return LValue{kind: lvWildcard}
}

// TargetField is a field of inner; for `self.f` the root is the field's
// symbol.
func TargetField(cx Context, inner LValue, access *ast.BLangFieldBaseAccess) LValue {
	fn, ok := EnclosingFunction(cx)
	if ok && inner.kind == lvPlain && fn.isSelf(inner.root) {
		if field, found := fn.class.field(access.Field.GetValue()); found {
			return LValue{root: field.sym}
		}
	}
	return LValue{root: inner.root}
}

// FinalFieldReassigned returns the name of the final field of the enclosing
// class that access writes as `self.f` outside its one-time initialization: a
// plain `=` directly in the class init to a field without an initializer.
func FinalFieldReassigned(cx Context, access *ast.BLangFieldBaseAccess, compound bool) (string, bool) {
	fn, ok := EnclosingFunction(cx)
	receiver, isRef := access.Expr.(*ast.BLangVarRef)
	if !ok || !isRef || !fn.isSelf(receiver.Symbol()) {
		return "", false
	}
	name := access.Field.GetValue()
	for _, field := range fn.class.fieldDescriptors {
		if field.MemberName() == name && field.IsFinal() {
			return name, compound || field.HasDefault() || !fn.isClassInit()
		}
	}
	return "", false
}

// TargetIndex is a member of inner.
func TargetIndex(inner LValue, _ *ast.BLangIndexBasedAccess) LValue {
	return LValue{root: inner.root}
}
