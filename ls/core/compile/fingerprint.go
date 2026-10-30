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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Invalid fingerprints never compare equal, including to themselves.
type fingerprint struct {
	value string
	valid bool
}

func (a fingerprint) equal(b fingerprint) bool {
	return a.valid && b.valid && a.value == b.value
}

type functionContract struct {
	Name             string
	Kind             model.SymbolKind
	Flags            model.FuncSymbolFlags
	DeclarationFlags model.Flag
	Type             string
	Parameters       []parameterContract
	Return           string
}

type parameterContract struct {
	Name             string
	Type             string
	Flags            model.ParamFlag
	DeclarationFlags model.Flag
	Defaultable      bool
	Rest             bool
}

// computeFingerprint accepts only plain functions over builtin int/string.
// Rendering arbitrary semantic types is not a complete public-contract encoding.
func computeFingerprint(ctx *context.CompilerContext, exported model.ExportedSymbolSpace, pkgNode *ast.BLangPackage) (fp fingerprint) {
	defer func() {
		if recover() != nil {
			fp = fingerprint{}
		}
	}()
	if ctx == nil || ctx.HasErrors() || pkgNode == nil || len(exported.MainSpaces) == 0 {
		return fingerprint{}
	}
	declarations := make(map[model.SymbolRef]*ast.BLangFunction)
	for _, fn := range pkgNode.Functions {
		if fn == nil || fn.Symbol().IsEmpty() {
			continue
		}
		if _, duplicate := declarations[fn.Symbol()]; duplicate {
			return fingerprint{}
		}
		declarations[fn.Symbol()] = fn
	}
	for _, space := range exported.AnnotationSpaces {
		for ref := range space.Symbols() {
			if ctx.SymbolIsPublic(ref) {
				return fingerprint{}
			}
		}
	}
	tcx := semtypes.TypeCheckContext(ctx.GetTypeEnv())
	contracts := make([]functionContract, 0)
	for ref := range exported.PublicMainSymbols() {
		if !ctx.SymbolIsPublic(ref) {
			continue
		}
		contract, ok := supportedFunctionContract(ctx, tcx, ref, declarations[ref])
		if !ok {
			return fingerprint{}
		}
		contracts = append(contracts, contract)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].Name < contracts[j].Name })
	encoded, err := json.Marshal(contracts)
	if err != nil {
		return fingerprint{}
	}
	sum := sha256.Sum256(encoded)
	return fingerprint{value: hex.EncodeToString(sum[:]), valid: true}
}

func supportedFunctionContract(ctx *context.CompilerContext, tcx semtypes.Context, ref model.SymbolRef, fn *ast.BLangFunction) (functionContract, bool) {
	if ctx.SymbolKind(ref) != model.SymbolKindFunction || fn == nil || fn.IsAttached() || fn.IsResource() || len(fn.AnnAttachments) != 0 {
		return functionContract{}, false
	}
	if body, ok := fn.Body.(*ast.BLangExternFunctionBody); ok && len(body.AnnAttachments) != 0 {
		return functionContract{}, false
	}
	ret := fn.GetReturnTypeDescriptor()
	if ret == nil || len(ret.AnnAttachments) != 0 {
		return functionContract{}, false
	}
	typed, ok := ctx.FunctionTypedSignature(ref)
	if !ok || typed.Flags != fn.FuncSymbolFlags() {
		return functionContract{}, false
	}
	untyped, ok := ctx.GetFunctionSignature(ref)
	count := len(fn.RequiredParams)
	if fn.RestParam != nil {
		count++
	}
	if !ok || len(typed.ParamTypes) != len(fn.RequiredParams) || len(untyped.ParamNames) != count || len(untyped.ParamFlags) != count || len(untyped.Default) != count || len(untyped.IncludedRecordMetadata) != count || untyped.HasRest != (fn.RestParam != nil) {
		return functionContract{}, false
	}
	returnName, ok := supportedBuiltin(tcx, ret.TypeDescriptor, typed.ReturnType)
	if !ok {
		return functionContract{}, false
	}
	contract := functionContract{Name: ctx.SymbolName(ref), Kind: ctx.SymbolKind(ref), Flags: typed.Flags, DeclarationFlags: fn.Flags(), Parameters: make([]parameterContract, 0, count), Return: returnName}
	if fn.Name == nil || fn.Name.GetValue() != contract.Name {
		return functionContract{}, false
	}
	for i := 0; i < count; i++ {
		rest := i == len(fn.RequiredParams)
		param := fn.RestParam
		ty := typed.RestParamType
		if !rest {
			param = &fn.RequiredParams[i]
			ty = typed.ParamTypes[i]
		}
		if param == nil || len(param.AnnAttachments) != 0 || param.IsIncludedRecordParam() || untyped.IncludedRecordMetadata[i] != nil {
			return functionContract{}, false
		}
		name, ok := supportedBuiltin(tcx, param.TypeNode(), ty)
		if !ok || param.Name == nil || param.Name.GetValue() != untyped.ParamNames[i] || untyped.ParamNames[i] == "" {
			return functionContract{}, false
		}
		flags := untyped.ParamFlags[i]
		defaultable := flags&model.ParamFlagDefaultable != 0
		if flags&model.ParamFlagIncludedRecordParam != 0 || (flags&model.ParamFlagRestParam != 0) != rest || defaultable != (untyped.Default[i] != nil) || defaultable != param.IsDefaultableParam() || defaultable != (param.Expr != nil) || rest != param.IsRestParam() {
			return functionContract{}, false
		}
		if untyped.Default[i] != nil && untyped.Default[i].Kind != model.DefaultableParamKindExpr {
			return functionContract{}, false
		}
		contract.Parameters = append(contract.Parameters, parameterContract{Name: untyped.ParamNames[i], Type: name, Flags: flags, DeclarationFlags: param.Flags(), Defaultable: defaultable, Rest: rest})
	}
	if fn.RestParam == nil && !semtypes.IsNever(typed.RestParamType) {
		return functionContract{}, false
	}
	// Validate the structural type before using its rendering. The bounded shape
	// has no aliases, recursion, composites, or renderer fallback cases.
	env := ctx.GetTypeEnv()
	listDefinition := semtypes.NewListDefinition()
	params := listDefinition.Define(env, typed.ParamTypes, semtypes.ListRest(typed.RestParamType), semtypes.ListMutability(semtypes.CellMutabilityNone))
	functionDefinition := semtypes.NewFunctionDefinition()
	expected := functionDefinition.Define(env, params, typed.ReturnType, semtypes.FunctionQualifiersFrom(env, fn.IsIsolated(), fn.IsTransactional()))
	if !semtypes.IsSameType(tcx, ctx.SymbolType(ref), expected) {
		return functionContract{}, false
	}
	contract.Type = semtypes.ToString(tcx, ctx.SymbolType(ref))
	return contract, true
}

func supportedBuiltin(tcx semtypes.Context, node ast.BType, ty semtypes.SemType) (string, bool) {
	builtin, ok := node.(*ast.BLangValueType)
	if !ok {
		return "", false
	}
	switch builtin.TypeKind {
	case ast.TypeKindInt:
		return "int", semtypes.IsSameType(tcx, ty, semtypes.Int)
	case ast.TypeKindString:
		return "string", semtypes.IsSameType(tcx, ty, semtypes.String)
	default:
		return "", false
	}
}
