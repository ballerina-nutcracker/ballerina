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

package opaque

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// functionDefinitions is the immutable table of opaque function definitions,
// keyed by organization and package, with the package-scoped opaque id as the
// slice position. A nil slot is a position held by an opaque type symbol.
var functionDefinitions = buildFunctionDefinitions()

func buildFunctionDefinitions() map[packageKey][]*FunctionDefinition {
	mapParams := func() []model.Param {
		return []model.Param{{Name: "m"}, {Name: "k"}}
	}
	// map:get and map:remove monomorphize identically, so they share one closure.
	// Their cache partitions stay distinct because owner is the caller's table
	// slot; both key on the container type alone, so a shared partition would
	// hand one function the other's symbol.
	mapMember := func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
		semanticError SemanticError, _ bool, args []ast.BLangExpression,
		_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
		if len(args) == 0 {
			semanticError("missing container argument", pos)
			return model.SymbolRef{}, false
		}
		containerTy, ok := resolve(args[0], semtypes.SemType{})
		if !ok {
			return model.SymbolRef{}, false
		}
		if ref, found := ctx.lookupMono(owner, containerTy); found {
			return ref, true
		}
		cx := ctx.typeContext()
		if !semtypes.IsSubtype(cx, containerTy, semtypes.Mapping) {
			semanticError("expect first argument to be a subtype of map<any|error>", pos)
			return model.SymbolRef{}, false
		}
		memberType := semtypes.MappingMemberTypeInnerValProj(cx, containerTy, semtypes.String)
		ref, ok := materialize(model.TypedFunctionSignature{
			ParamTypes:    []semtypes.SemType{containerTy, semtypes.String},
			RestParamType: semtypes.Never,
			ReturnType:    memberType,
			Flags:         model.FuncSymbolFlagIsolated,
		})
		if !ok {
			return model.SymbolRef{}, false
		}
		ctx.storeMono(owner, ref, containerTy)
		return ref, true
	}

	return map[packageKey][]*FunctionDefinition{
		{org: "ballerina", pkg: "lang.array"}: {
			model.OpaqueFnArrayPush: {
				name: "push",
				params: []model.Param{
					{Name: "arr"},
					{Name: "vals", Flag: model.ParamFlagRestParam},
				},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					if len(args) == 0 {
						semanticError("missing container argument", pos)
						return model.SymbolRef{}, false
					}
					containerTy, ok := resolve(args[0], semtypes.SemType{})
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					cx := ctx.typeContext()
					if !semtypes.IsSubtype(cx, containerTy, semtypes.List) {
						semanticError("expect first argument to be a subtype of (any|error)[]", pos)
						return model.SymbolRef{}, false
					}
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy},
						RestParamType: semtypes.ListProj(cx, containerTy, semtypes.Int),
						ReturnType:    semtypes.Nil,
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
			model.OpaqueFnArrayMap: {
				name: "map",
				params: []model.Param{
					{Name: "arr"},
					{Name: "func", Flag: model.ParamFlagIsolated},
				},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, isolated bool, args []ast.BLangExpression,
					expected semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					if len(args) == 0 {
						semanticError("missing container argument", pos)
						return model.SymbolRef{}, false
					}
					containerExpr := args[0]
					containerTy, ok := resolve(containerExpr, semtypes.SemType{})
					if !ok {
						return model.SymbolRef{}, false
					}
					cx := ctx.typeContext()
					env := ctx.typeEnv()
					if !semtypes.IsSubtype(cx, containerTy, semtypes.List) {
						semanticError("expect first argument to be a list subtype", containerExpr.GetPosition())
						return model.SymbolRef{}, false
					}
					memberTy := semtypes.ListProj(cx, containerTy, semtypes.Int)

					if len(args) < 2 {
						semanticError("missing callback argument", pos)
						return model.SymbolRef{}, false
					}
					callbackExpr := args[1]
					callbackReturnTy := semtypes.Val
					if !semtypes.IsZero(expected) && !semtypes.IsNever(expected) &&
						semtypes.IsSubtype(cx, expected, semtypes.List) {
						callbackReturnTy = semtypes.ListProj(cx, expected, semtypes.Int)
					}
					callbackFlags := model.FuncSymbolFlags(0)
					if isolated {
						callbackFlags = model.FuncSymbolFlagIsolated
					}
					callbackTopTy := FunctionSemType(env, model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{memberTy},
						ReturnType:    callbackReturnTy,
						RestParamType: semtypes.Never,
						Flags:         callbackFlags,
					})
					callbackTy, ok := resolve(callbackExpr, callbackTopTy)
					if !ok {
						return model.SymbolRef{}, false
					}
					callbackArgsDef := semtypes.NewListDefinition()
					callbackArgsTy := callbackArgsDef.Define(env, []semtypes.SemType{memberTy},
						semtypes.ListMutability(semtypes.CellMutabilityNone))
					var resultMemberTy semtypes.SemType
					if semtypes.IsNever(memberTy) {
						resultMemberTy = semtypes.FunctionReturnType(cx, callbackTy,
							semtypes.FunctionParamListType(cx, callbackTy))
					} else {
						resultMemberTy = semtypes.FunctionReturnType(cx, callbackTy, callbackArgsTy)
					}
					if semtypes.IsZero(resultMemberTy) {
						semanticError("callback is not callable with the array member type",
							callbackExpr.GetPosition())
						return model.SymbolRef{}, false
					}
					callbackParamTy := FunctionSemType(env, model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{memberTy},
						ReturnType:    resultMemberTy,
						RestParamType: semtypes.Never,
						Flags:         callbackFlags,
					})
					if ref, found := ctx.lookupMono(owner, containerTy, resultMemberTy,
						callbackParamTy); found {
						return ref, true
					}
					resultDef := semtypes.NewListDefinition()
					resultTy := resultDef.Define(env, nil, semtypes.ListRest(resultMemberTy))
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy, callbackParamTy},
						ReturnType:    resultTy,
						RestParamType: semtypes.Never,
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy, resultMemberTy, callbackParamTy)
					return ref, true
				},
			},
			model.OpaqueFnArrayIndexOf: {
				name: "indexOf",
				// Declared in lang.array's source, with `startIndex`'s default.
				sourceDeclared: true,
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					if len(args) == 0 {
						semanticError("missing container argument", pos)
						return model.SymbolRef{}, false
					}
					containerTy, ok := resolve(args[0], semtypes.SemType{})
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					cx := ctx.typeContext()
					anydataArrDef := semtypes.NewListDefinition()
					anydataArrTy := anydataArrDef.Define(ctx.typeEnv(), nil,
						semtypes.ListRest(semtypes.CreateAnydata(cx)))
					if !semtypes.IsSubtype(cx, containerTy, anydataArrTy) {
						semanticError("expect first argument to be a subtype of anydata[]", pos)
						return model.SymbolRef{}, false
					}
					valType := semtypes.ListProj(cx, containerTy, semtypes.Int)
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy, valType, semtypes.Int},
						RestParamType: semtypes.Never,
						ReturnType:    semtypes.Union(semtypes.Int, semtypes.Nil),
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
			model.OpaqueFnArrayRemove: {
				name:   "remove",
				params: []model.Param{{Name: "arr"}, {Name: "index"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					containerTy, ok := resolveMutableList(ctx, resolve, semanticError, args, pos)
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy, semtypes.Int},
						RestParamType: semtypes.Never,
						ReturnType:    semtypes.ListProj(ctx.typeContext(), containerTy, semtypes.Int),
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
			model.OpaqueFnArrayRemoveAll: {
				name:   "removeAll",
				params: []model.Param{{Name: "arr"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					containerTy, ok := resolveMutableList(ctx, resolve, semanticError, args, pos)
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy},
						RestParamType: semtypes.Never,
						ReturnType:    semtypes.Nil,
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
		},
		{org: "ballerina", pkg: "lang.map"}: {
			model.OpaqueFnMapRemove: {name: "remove", params: mapParams(), monomorphize: mapMember},
			model.OpaqueFnMapGet:    {name: "get", params: mapParams(), monomorphize: mapMember},
		},
		{org: "ballerina", pkg: "lang.xml"}: {
			// ids 0-3 are opaque type symbols and stay nil.
			model.OpaqueFnXMLIterator: {
				name:   "iterator",
				params: []model.Param{{Name: "x"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					if len(args) == 0 {
						semanticError("missing container argument", pos)
						return model.SymbolRef{}, false
					}
					containerTy, ok := resolve(args[0], semtypes.SemType{})
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					cx := ctx.typeContext()
					if !semtypes.IsSubtype(cx, containerTy, semtypes.XML) {
						semanticError("expect first argument to be a subtype of xml", pos)
						return model.SymbolRef{}, false
					}
					itemTy := semtypes.XMLItemType(containerTy)
					env := ctx.typeEnv()
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy},
						RestParamType: semtypes.Never,
						ReturnType: ctx.xmlIteratorType(itemTy, func() semtypes.SemType {
							recordDef := semtypes.NewMappingDefinition()
							recordTy := recordDef.Define(env,
								[]semtypes.Field{semtypes.FieldFrom("value", itemTy, false, false)},
								semtypes.Never)
							nextReturnTy := semtypes.Union(recordTy, semtypes.Nil)
							ld := semtypes.NewListDefinition()
							emptyParams := ld.Define(env, nil, semtypes.ListMutability(semtypes.CellMutabilityNone))
							fd := semtypes.NewFunctionDefinition()
							nextFnTy := fd.Define(env, emptyParams, nextReturnTy,
								semtypes.FunctionQualifiersFrom(env, true, false))
							iterOd := semtypes.NewObjectDefinition()
							return iterOd.Define(env, semtypes.ObjectQualifiersDefault, []semtypes.Member{{
								Name:       "next",
								ValueType:  nextFnTy,
								Kind:       semtypes.MemberKindMethod,
								Visibility: semtypes.VisibilityPublic,
								Immutable:  true,
							}})
						}),
						Flags: model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
		},
	}
}

// resolveMutableList resolves the container argument of an array function that
// mutates its container in place, rejecting a container that is statically
// readonly.
func resolveMutableList(ctx *Context, resolve Resolve, semanticError SemanticError,
	args []ast.BLangExpression, pos diagnostics.Location) (semtypes.SemType, bool) {
	if len(args) == 0 {
		semanticError("missing container argument", pos)
		return semtypes.SemType{}, false
	}
	containerTy, ok := resolve(args[0], semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, false
	}
	cx := ctx.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.List) {
		semanticError("expect first argument to be a subtype of (any|error)[]", pos)
		return semtypes.SemType{}, false
	}
	if semtypes.IsSubtype(cx, containerTy, semtypes.ValReadonly) {
		semanticError("cannot update 'readonly' value of type '"+semtypes.ToString(cx, containerTy)+"'", pos)
		return semtypes.SemType{}, false
	}
	return containerTy, true
}
