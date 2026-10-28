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

	// array:pop and array:shift share one monomorphizer. A shift also moves the
	// remaining members down, so it rejects a tuple whose members don't fit there.
	arrayRemoveEnd := func(name string, fromHead bool) monomorphizer {
		return func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
			semanticError SemanticError, _ bool, args []ast.BLangExpression,
			_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
			containerTy, ok := resolveMutableList(ctx, resolve, semanticError, args, pos)
			if !ok {
				return model.SymbolRef{}, false
			}
			if ref, found := ctx.lookupMono(owner, containerTy); found {
				return ref, true
			}
			cx := ctx.typeContext()
			if atomic := semtypes.ToListAtomicType(ctx.typeEnv(), containerTy); atomic != nil {
				if semtypes.IsNever(atomic.Rest()) {
					semanticError("cannot "+name+" a fixed-length list", pos)
					return model.SymbolRef{}, false
				}
				if fromHead && !canShiftMembersDown(cx, atomic) {
					semanticError("cannot "+name+" a tuple without violating its inherent type", pos)
					return model.SymbolRef{}, false
				}
			}
			ref, ok := materialize(model.TypedFunctionSignature{
				ParamTypes:    []semtypes.SemType{containerTy},
				RestParamType: semtypes.Never,
				ReturnType:    semtypes.ListProj(cx, containerTy, semtypes.Int),
				Flags:         model.FuncSymbolFlagIsolated,
			})
			if !ok {
				return model.SymbolRef{}, false
			}
			ctx.storeMono(owner, ref, containerTy)
			return ref, true
		}
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
					if atomic := semtypes.ToListAtomicType(ctx.typeEnv(), containerTy); atomic != nil && semtypes.IsNever(atomic.Rest()) {
						semanticError("cannot push to a fixed-length list", pos)
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
			model.OpaqueFnArrayToStream: {
				name:   "toStream",
				params: []model.Param{{Name: "arr"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					if len(args) == 0 {
						semanticError("missing container argument", pos)
						return model.SymbolRef{}, false
					}
					containerExpr := args[0]
					containerTy, ok := resolve(containerExpr, semtypes.SemType{})
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					cx := ctx.typeContext()
					if !semtypes.IsSubtype(cx, containerTy, semtypes.List) {
						semanticError("expect first argument to be a list subtype", containerExpr.GetPosition())
						return model.SymbolRef{}, false
					}
					memberTy := semtypes.ListProj(cx, containerTy, semtypes.Int)
					streamDef := semtypes.NewStreamDefinition()
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy},
						RestParamType: semtypes.Never,
						ReturnType:    streamDef.Define(ctx.typeEnv(), memberTy, semtypes.Nil),
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
			model.OpaqueFnArrayPop: {
				name:         "pop",
				params:       []model.Param{{Name: "arr"}},
				monomorphize: arrayRemoveEnd("pop", false),
			},
			model.OpaqueFnArrayShift: {
				name:         "shift",
				params:       []model.Param{{Name: "arr"}},
				monomorphize: arrayRemoveEnd("shift", true),
			},
			model.OpaqueFnArrayUnshift: {
				name: "unshift",
				params: []model.Param{
					{Name: "arr"},
					{Name: "vals", Flag: model.ParamFlagRestParam},
				},
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
					cx := ctx.typeContext()
					if atomic := semtypes.ToListAtomicType(ctx.typeEnv(), containerTy); atomic != nil && semtypes.IsNever(atomic.Rest()) {
						semanticError("cannot unshift a fixed-length list", pos)
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
			model.OpaqueFnArrayReduce: {
				name: "reduce",
				params: []model.Param{
					{Name: "arr"},
					{Name: "func", Flag: model.ParamFlagIsolated},
					{Name: "initial"},
				},
				monomorphize: monomorphizeArrayReduce,
			},
			model.OpaqueFnArraySort: {
				name: "sort",
				// Declared in lang.array's source, with `direction` and `key` defaults.
				sourceDeclared: true,
				monomorphize:   monomorphizeArraySort,
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
			model.OpaqueFnXMLGet: {
				name:   "get",
				params: []model.Param{{Name: "x"}, {Name: "i"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					containerTy, itemTy, ok := resolveXMLContainer(ctx, resolve, semanticError, args, pos)
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy, semtypes.Int},
						RestParamType: semtypes.Never,
						ReturnType:    itemTy,
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
			model.OpaqueFnXMLSlice: {
				name:   "slice",
				params: []model.Param{{Name: "x"}, {Name: "startIndex"}, {Name: "endIndex"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					containerTy, itemTy, ok := resolveXMLContainer(ctx, resolve, semanticError, args, pos)
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy, semtypes.Int, semtypes.Int},
						RestParamType: semtypes.Never,
						ReturnType:    semtypes.XMLSequence(itemTy),
						Flags:         model.FuncSymbolFlagIsolated,
					})
					if !ok {
						return model.SymbolRef{}, false
					}
					ctx.storeMono(owner, ref, containerTy)
					return ref, true
				},
			},
			model.OpaqueFnXMLMap: {
				name:   "map",
				params: xmlCallbackParams(),
				monomorphize: xmlCallbackMonomorphizer(semtypes.XML, func(callbackReturnTy semtypes.SemType) semtypes.SemType {
					return semtypes.XMLSequence(semtypes.XMLItemType(callbackReturnTy))
				}),
			},
			model.OpaqueFnXMLForEach: {
				name:   "forEach",
				params: xmlCallbackParams(),
				monomorphize: xmlCallbackMonomorphizer(semtypes.Nil, func(semtypes.SemType) semtypes.SemType {
					return semtypes.Nil
				}),
			},
			model.OpaqueFnXMLFilter: {
				name:   "filter",
				params: xmlCallbackParams(),
				monomorphize: xmlCallbackMonomorphizer(semtypes.Boolean, func(semtypes.SemType) semtypes.SemType {
					return semtypes.XML
				}),
			},
			// $stepIndex backs an indexed XML step. It is not visible to Ballerina
			// source, so the compiler is its only caller.
			model.OpaqueFnXMLStepIndex: {
				name:   "$stepIndex",
				params: []model.Param{{Name: "x"}, {Name: "i"}},
				monomorphize: func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
					semanticError SemanticError, _ bool, args []ast.BLangExpression,
					_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
					containerTy, itemTy, ok := resolveXMLContainer(ctx, resolve, semanticError, args, pos)
					if !ok {
						return model.SymbolRef{}, false
					}
					if ref, found := ctx.lookupMono(owner, containerTy); found {
						return ref, true
					}
					ref, ok := materialize(model.TypedFunctionSignature{
						ParamTypes:    []semtypes.SemType{containerTy, semtypes.Int},
						RestParamType: semtypes.Never,
						ReturnType:    semtypes.XMLSequence(itemTy),
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
	}
}

func xmlCallbackParams() []model.Param {
	return []model.Param{{Name: "x"}, {Name: "func", Flag: model.ParamFlagIsolated}}
}

// resolveXMLContainer resolves the container argument of an XML function and
// returns the container type together with the type of the items it holds.
func resolveXMLContainer(ctx *Context, resolve Resolve, semanticError SemanticError,
	args []ast.BLangExpression, pos diagnostics.Location) (semtypes.SemType, semtypes.SemType, bool) {
	if len(args) == 0 {
		semanticError("missing XML argument", pos)
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	containerExpr := args[0]
	containerTy, ok := resolve(containerExpr, semtypes.XML)
	if !ok {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	if !semtypes.IsSubtype(ctx.typeContext(), containerTy, semtypes.XML) {
		semanticError("expect first argument to be a subtype of xml", containerExpr.GetPosition())
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	return containerTy, semtypes.XMLItemType(containerTy), true
}

// xmlCallbackMonomorphizer builds the monomorphizer of an XML function that
// applies a callback to each item. returnConstraint is the widest return type the
// callback may have and result gives the function's return type from the
// callback's own return type.
func xmlCallbackMonomorphizer(returnConstraint semtypes.SemType,
	result func(callbackReturnTy semtypes.SemType) semtypes.SemType) monomorphizer {
	return func(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
		semanticError SemanticError, isolated bool, args []ast.BLangExpression,
		_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
		containerTy, itemTy, ok := resolveXMLContainer(ctx, resolve, semanticError, args, pos)
		if !ok {
			return model.SymbolRef{}, false
		}
		if len(args) < 2 {
			semanticError("missing XML callback argument", pos)
			return model.SymbolRef{}, false
		}
		callbackExpr := args[1]
		cx := ctx.typeContext()
		env := ctx.typeEnv()
		callbackFlags := model.FuncSymbolFlags(0)
		if isolated {
			callbackFlags = model.FuncSymbolFlagIsolated
		}
		callbackTopTy := FunctionSemType(env, model.TypedFunctionSignature{
			ParamTypes:    []semtypes.SemType{itemTy},
			RestParamType: semtypes.Never,
			ReturnType:    returnConstraint,
			Flags:         callbackFlags,
		})
		callbackTy, ok := resolve(callbackExpr, callbackTopTy)
		if !ok {
			return model.SymbolRef{}, false
		}
		callbackArgsDef := semtypes.NewListDefinition()
		callbackArgsTy := callbackArgsDef.Define(env, []semtypes.SemType{itemTy},
			semtypes.ListMutability(semtypes.CellMutabilityNone))
		callbackReturnTy := semtypes.FunctionReturnType(cx, callbackTy, callbackArgsTy)
		if semtypes.IsZero(callbackReturnTy) || !semtypes.IsSubtype(cx, callbackReturnTy, returnConstraint) {
			semanticError("XML callback has incompatible return type", callbackExpr.GetPosition())
			return model.SymbolRef{}, false
		}
		callbackParamTy := FunctionSemType(env, model.TypedFunctionSignature{
			ParamTypes:    []semtypes.SemType{itemTy},
			RestParamType: semtypes.Never,
			ReturnType:    callbackReturnTy,
			Flags:         callbackFlags,
		})
		if ref, found := ctx.lookupMono(owner, containerTy, callbackReturnTy, callbackParamTy); found {
			return ref, true
		}
		ref, ok := materialize(model.TypedFunctionSignature{
			ParamTypes:    []semtypes.SemType{containerTy, callbackParamTy},
			RestParamType: semtypes.Never,
			ReturnType:    result(callbackReturnTy),
			Flags:         model.FuncSymbolFlagIsolated,
		})
		if !ok {
			return model.SymbolRef{}, false
		}
		ctx.storeMono(owner, ref, containerTy, callbackReturnTy, callbackParamTy)
		return ref, true
	}
}

// monomorphizeArrayReduce takes the accumulator type, which reduce returns, from
// the callback's first parameter.
func monomorphizeArrayReduce(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
	semanticError SemanticError, isolated bool, args []ast.BLangExpression,
	expected semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerTy, memberTy, ok := resolveList(ctx, resolve, semanticError, args, pos)
	if !ok {
		return model.SymbolRef{}, false
	}
	if len(args) < 2 {
		semanticError("missing callback argument", pos)
		return model.SymbolRef{}, false
	}
	cx := ctx.typeContext()
	env := ctx.typeEnv()
	callbackExpr := args[1]
	expectedAccParamTy := semtypes.Never
	expectedAccReturnTy := semtypes.Val
	if !semtypes.IsZero(expected) && !semtypes.IsNever(expected) {
		expectedAccParamTy = expected
		expectedAccReturnTy = expected
	}
	callbackFlags := model.FuncSymbolFlags(0)
	if isolated {
		callbackFlags = model.FuncSymbolFlagIsolated
	}
	callbackTy, ok := resolve(callbackExpr, FunctionSemType(env, model.TypedFunctionSignature{
		ParamTypes:    []semtypes.SemType{expectedAccParamTy, memberTy},
		ReturnType:    expectedAccReturnTy,
		RestParamType: semtypes.Never,
		Flags:         callbackFlags,
	}))
	if !ok {
		return model.SymbolRef{}, false
	}
	callbackParamListTy := semtypes.FunctionParamListType(cx, callbackTy)
	if semtypes.IsZero(callbackParamListTy) {
		semanticError("callback must be a function", callbackExpr.GetPosition())
		return model.SymbolRef{}, false
	}
	accTy := semtypes.ListProj(cx, callbackParamListTy, semtypes.IntConst(0))
	callbackArgsDef := semtypes.NewListDefinition()
	callbackArgsTy := callbackArgsDef.Define(env, []semtypes.SemType{accTy, memberTy},
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	callbackReturnTy := semtypes.FunctionReturnType(cx, callbackTy, callbackArgsTy)
	if semtypes.IsZero(callbackReturnTy) {
		semanticError("callback is not callable with an accumulator and the array member type",
			callbackExpr.GetPosition())
		return model.SymbolRef{}, false
	}
	if !semtypes.IsSubtype(cx, callbackReturnTy, accTy) {
		semanticError("callback return type must be a subtype of its accumulator parameter type",
			callbackExpr.GetPosition())
		return model.SymbolRef{}, false
	}
	callbackParamTy := FunctionSemType(env, model.TypedFunctionSignature{
		ParamTypes:    []semtypes.SemType{accTy, memberTy},
		ReturnType:    callbackReturnTy,
		RestParamType: semtypes.Never,
		Flags:         callbackFlags,
	})
	if ref, found := ctx.lookupMono(owner, containerTy, accTy, callbackParamTy); found {
		return ref, true
	}
	ref, ok := materialize(model.TypedFunctionSignature{
		ParamTypes:    []semtypes.SemType{containerTy, callbackParamTy, accTy},
		RestParamType: semtypes.Never,
		ReturnType:    accTy,
		Flags:         model.FuncSymbolFlagIsolated,
	})
	if !ok {
		return model.SymbolRef{}, false
	}
	ctx.storeMono(owner, ref, containerTy, accTy, callbackParamTy)
	return ref, true
}

// monomorphizeArraySort requires the values sort compares, the members or the
// key function's results, to have an ordered type.
func monomorphizeArraySort(ctx *Context, owner cacheOwner, resolve Resolve, materialize Materialize,
	semanticError SemanticError, isolated bool, args []ast.BLangExpression,
	_ semtypes.SemType, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerTy, memberTy, ok := resolveList(ctx, resolve, semanticError, args, pos)
	if !ok {
		return model.SymbolRef{}, false
	}
	cx := ctx.typeContext()
	env := ctx.typeEnv()
	directionTy := semtypes.Union(semtypes.StringConst("ascending"), semtypes.StringConst("descending"))
	directionArgTy, ok := resolve(args[1], directionTy)
	if !ok {
		return model.SymbolRef{}, false
	}
	if !semtypes.IsSubtype(cx, directionArgTy, directionTy) {
		semanticError("expect direction to be a subtype of array:SortDirection", args[1].GetPosition())
		return model.SymbolRef{}, false
	}
	keyParamTy, sortKeyTys, ok := resolveSortKey(ctx, resolve, semanticError, isolated, args[2], memberTy)
	if !ok {
		return model.SymbolRef{}, false
	}
	for _, sortKeyTy := range sortKeyTys {
		if !semtypes.IsSubtype(cx, sortKeyTy, semtypes.CreateOrdered(cx)) || !semtypes.Comparable(cx, sortKeyTy, sortKeyTy) {
			semanticError("sort key must have an ordered type", pos)
			return model.SymbolRef{}, false
		}
	}
	if ref, found := ctx.lookupMono(owner, containerTy, keyParamTy); found {
		return ref, true
	}
	resultDef := semtypes.NewListDefinition()
	ref, ok := materialize(model.TypedFunctionSignature{
		ParamTypes:    []semtypes.SemType{containerTy, directionTy, keyParamTy},
		RestParamType: semtypes.Never,
		ReturnType:    resultDef.Define(env, nil, semtypes.ListRest(memberTy)),
		Flags:         model.FuncSymbolFlagIsolated,
	})
	if !ok {
		return model.SymbolRef{}, false
	}
	ctx.storeMono(owner, ref, containerTy, keyParamTy)
	return ref, true
}

// resolveSortKey returns the key parameter type and the types sort may compare:
// the members when key may be (), and the key's results when it may be a
// function. A single sort compares only one of these, so each is checked alone.
func resolveSortKey(ctx *Context, resolve Resolve, semanticError SemanticError, isolated bool,
	keyExpr ast.BLangExpression, memberTy semtypes.SemType) (semtypes.SemType, []semtypes.SemType, bool) {
	if _, omitted := keyExpr.(*ast.BLangDefaultArg); omitted {
		// The default resolves to the declared parameter type, so accept that type.
		keyTy, ok := resolve(keyExpr, semtypes.Nil)
		return keyTy, []semtypes.SemType{memberTy}, ok
	}
	cx := ctx.typeContext()
	env := ctx.typeEnv()
	keyFlags := model.FuncSymbolFlags(0)
	if isolated {
		keyFlags = model.FuncSymbolFlagIsolated
	}
	keyTopTy := semtypes.Union(semtypes.Nil, FunctionSemType(env, model.TypedFunctionSignature{
		ParamTypes:    []semtypes.SemType{memberTy},
		ReturnType:    semtypes.CreateOrdered(cx),
		RestParamType: semtypes.Never,
		Flags:         keyFlags,
	}))
	keyTy, ok := resolve(keyExpr, keyTopTy)
	if !ok {
		return semtypes.SemType{}, nil, false
	}
	keyFnTy := semtypes.Diff(keyTy, semtypes.Nil)
	if semtypes.IsNever(keyFnTy) {
		return semtypes.Nil, []semtypes.SemType{memberTy}, true
	}
	keyArgsDef := semtypes.NewListDefinition()
	keyArgsTy := keyArgsDef.Define(env, []semtypes.SemType{memberTy},
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	keyReturnTy := semtypes.FunctionReturnType(cx, keyFnTy, keyArgsTy)
	if semtypes.IsZero(keyReturnTy) {
		semanticError("key function is not callable with the array member type", keyExpr.GetPosition())
		return semtypes.SemType{}, nil, false
	}
	keyParamTy := FunctionSemType(env, model.TypedFunctionSignature{
		ParamTypes:    []semtypes.SemType{memberTy},
		ReturnType:    keyReturnTy,
		RestParamType: semtypes.Never,
		Flags:         keyFlags,
	})
	if semtypes.IsNever(semtypes.Intersect(keyTy, semtypes.Nil)) {
		return keyParamTy, []semtypes.SemType{keyReturnTy}, true
	}
	return semtypes.Union(keyParamTy, semtypes.Nil), []semtypes.SemType{memberTy, keyReturnTy}, true
}

// resolveList resolves the container argument of an array function that only
// reads it, returning the container and member types.
func resolveList(ctx *Context, resolve Resolve, semanticError SemanticError,
	args []ast.BLangExpression, pos diagnostics.Location) (semtypes.SemType, semtypes.SemType, bool) {
	if len(args) == 0 {
		semanticError("missing container argument", pos)
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	containerTy, ok := resolve(args[0], semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	cx := ctx.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.List) {
		semanticError("expect first argument to be a list subtype", args[0].GetPosition())
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	return containerTy, semtypes.ListProj(cx, containerTy, semtypes.Int), true
}

// canShiftMembersDown reports whether each member fits the mandatory slot below it.
func canShiftMembersDown(cx semtypes.Context, atomic *semtypes.ListAtomicType) bool {
	for idx := range atomic.FixedLength() {
		if !semtypes.IsSubtype(cx, atomic.MemberAtInnerVal(idx+1), atomic.MemberAtInnerVal(idx)) {
			return false
		}
	}
	return true
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
