// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package signature renders call signatures from sealed compiler modules.
package signature

import (
	"cmp"
	stdcontext "context"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Help is a rendered call signature and its active parameter.
type Help struct {
	Label       string
	Description string
	// Parameters contains byte ranges within Label, not wire coordinates.
	Parameters             [][2]int
	ParameterDocumentation []parameterDocumentation
	ActiveParameter        *int
}

// At returns signature help at a byte offset in uri's source.
// The boolean is false when no supported call signature can be determined.
func At(ctx stdcontext.Context, sm compile.SealedModule, uri workspace.DocumentURI, offset int) (Help, bool, error) {
	if err := ctx.Err(); err != nil {
		return Help{}, false, err
	}
	text, ok := sm.SourceText(uri)
	if !ok || offset < 0 || offset > len(text) {
		return Help{}, false, nil
	}
	name, ok := sm.NameOf(uri)
	if !ok {
		return Help{}, false, nil
	}
	fileIndex := sm.Context().DiagnosticEnv().FileIndex(name)
	paths := nodeChainsAtOffset(sm.PackageNode(), offset, fileIndex)
	node, args := callAtPaths(paths, fileIndex)
	if node == nil {
		return Help{}, false, nil
	}
	ref, name, sig, ok := callSignature(sm, node, paths[0])
	if !ok {
		return Help{}, false, nil
	}
	parameter, ok := activeParameter(node, args, sig, offset, fileIndex)
	if !ok {
		return Help{}, false, nil
	}
	help := render(sm, ref, name, sig, parameter)
	return help, true, ctx.Err()
}

func callAtPaths(paths [][]ast.BLangNode, fileIndex int) (ast.BLangNode, []ast.BLangExpression) {
	if len(paths) == 0 {
		return nil, nil
	}
	node, args := innermostCall(paths[0])
	if len(paths) > 1 && !emptyCallContext(paths, node, args, fileIndex) {
		return nil, nil
	}
	return node, args
}

func emptyCallContext(paths [][]ast.BLangNode, call ast.BLangNode, args []ast.BLangExpression, fileIndex int) bool {
	switch call.(type) {
	case *ast.BLangInvocation, *ast.BLangRemoteMethodCallAction, *ast.BLangNewExpression:
	default:
		return false
	}
	loc := call.GetPosition()
	if !locationHasUsableOffsets(loc) || loc.FileIndex() != fileIndex {
		return false
	}
	alias, lowered := loweredMethodCallAlias(call)
	if lowered {
		args = args[1:]
	}
	for _, arg := range args {
		if !callWideDefault(arg, loc) {
			return false
		}
	}
	for _, path := range paths {
		inner, _ := innermostCall(path)
		if inner != call {
			return false
		}
		for i := len(path) - 1; path[i] != call; i-- {
			if !callWideDefault(path[i], loc) && (!lowered || path[i] != alias) {
				return false
			}
		}
	}
	return true
}

// Langlib lowering rewrites x.f(args) to f(x, args...) with a call-wide package alias.
func loweredMethodCallAlias(call ast.BLangNode) (ast.BLangNode, bool) {
	inv, ok := call.(*ast.BLangInvocation)
	if !ok || inv.Expr != nil || inv.PkgAlias == nil || len(inv.ArgExprs) == 0 {
		return nil, false
	}
	alias, ok := inv.PkgAlias.(ast.BLangNode)
	if !ok {
		return nil, false
	}
	aliasLoc, loc := alias.GetPosition(), inv.GetPosition()
	spansCall := locationHasUsableOffsets(aliasLoc) && aliasLoc.FileIndex() == loc.FileIndex() &&
		aliasLoc.StartOffset() == loc.StartOffset() && aliasLoc.EndOffset() == loc.EndOffset()
	return alias, spansCall
}

func callWideDefault(node ast.BLangNode, callLoc ast.Location) bool {
	if _, ok := node.(*ast.BLangDefaultArg); !ok {
		return false
	}
	return spansCall(node.GetPosition(), callLoc)
}

func spansCall(loc, callLoc ast.Location) bool {
	return locationHasUsableOffsets(loc) && loc.FileIndex() == callLoc.FileIndex() &&
		loc.StartOffset() == callLoc.StartOffset() && loc.EndOffset() == callLoc.EndOffset()
}

func innermostCall(chain []ast.BLangNode) (ast.BLangNode, []ast.BLangExpression) {
	for i := len(chain) - 1; i >= 0; i-- {
		switch n := chain[i].(type) {
		case *ast.BLangInvocation:
			return n, n.ArgExprs
		case *ast.BLangRemoteMethodCallAction:
			return n, n.ArgExprs
		case *ast.BLangNewExpression:
			return n, n.ArgsExprs
		case *ast.BLangClientResourceAccessAction:
			return n, nil
		}
	}
	return nil, nil
}

// unsupportedParameter marks a malformed argument, such as an unimplemented spread argument.
const unsupportedParameter = -2

type argumentSlot struct {
	start, end int
	parameter  int
}

// activeParameter returns the parameter index at offset, or -1 when no parameter matches.
// Type resolution moves argument values into parameter order, so the slots are sorted back
// into source order by position. The boolean is false when offset is not inside the argument
// list, is in a malformed argument, or is on an included-record parameter that the type
// resolver synthesized from named arguments.
func activeParameter(call ast.BLangNode, args []ast.BLangExpression, sig model.UntypedFunctionSignature, offset, fileIndex int) (int, bool) {
	if end, ok := calleeEnd(call, fileIndex); ok && offset < end {
		return 0, false
	}
	receivers := 0
	if _, lowered := loweredMethodCallAlias(call); lowered {
		receivers = 1
	}
	slots, named, synthesized := argumentSlots(call, args, receivers, sig, fileIndex)
	index := 0
	for index < len(slots) && slots[index].end < offset {
		index++
	}
	parameter := -1
	switch {
	case index < len(slots):
		parameter = slots[index].parameter
	case !named:
		parameter = positionalParameter(receivers+index, sig)
	}
	return parameter, parameter != unsupportedParameter && !synthesized[parameter]
}

// calleeEnd returns the end offset of the call's name or class, which precedes the argument list.
func calleeEnd(call ast.BLangNode, fileIndex int) (int, bool) {
	var callee ast.BLangNode
	switch n := call.(type) {
	case *ast.BLangInvocation:
		callee, _ = n.Name.(ast.BLangNode)
	case *ast.BLangRemoteMethodCallAction:
		callee, _ = n.Name.(ast.BLangNode)
	case *ast.BLangNewExpression:
		if n.TypeDescriptor == nil {
			loc := n.GetPosition()
			return loc.StartOffset() + len("new"), locationHasUsableOffsets(loc) && loc.FileIndex() == fileIndex
		}
		callee = n.TypeDescriptor
	}
	if callee == nil {
		return 0, false
	}
	loc := callee.GetPosition()
	return loc.EndOffset(), locationHasUsableOffsets(loc) && loc.FileIndex() == fileIndex
}

func argumentSlots(call ast.BLangNode, args []ast.BLangExpression, receivers int, sig model.UntypedFunctionSignature, fileIndex int) ([]argumentSlot, bool, map[int]bool) {
	callLoc := call.GetPosition()
	slots := make([]argumentSlot, 0, len(args))
	synthesized := make(map[int]bool)
	named := false
	for i := receivers; i < len(args); i++ {
		arg := args[i]
		if arg == nil {
			continue
		}
		loc := arg.GetPosition()
		if !locationHasUsableOffsets(loc) || loc.FileIndex() != fileIndex ||
			loc.StartOffset() < callLoc.StartOffset() || loc.EndOffset() > callLoc.EndOffset() {
			continue
		}
		if isMalformedArgument(arg) {
			slots = append(slots, argumentSlot{start: loc.StartOffset(), end: loc.EndOffset(), parameter: unsupportedParameter})
			continue
		}
		if spansCall(loc, callLoc) {
			if record, ok := arg.(*ast.BLangMappingConstructorExpr); ok {
				synthesized[i] = true
				slots = appendFieldSlots(slots, record, i, fileIndex)
			}
			continue
		}
		parameter := -1
		if namedArg, ok := arg.(*ast.BLangNamedArgsExpression); ok {
			named = true
			if index, ok := namedParameterIndex(namedArg, sig); ok {
				parameter = index
			}
		} else if !named {
			parameter = positionalParameter(i, sig)
		}
		slots = append(slots, argumentSlot{start: loc.StartOffset(), end: loc.EndOffset(), parameter: parameter})
	}
	slices.SortStableFunc(slots, func(a, b argumentSlot) int { return cmp.Compare(a.start, b.start) })
	return slots, named, synthesized
}

func appendFieldSlots(slots []argumentSlot, record *ast.BLangMappingConstructorExpr, parameter, fileIndex int) []argumentSlot {
	for _, field := range record.Fields {
		loc := field.GetPosition()
		if locationHasUsableOffsets(loc) && loc.FileIndex() == fileIndex {
			slots = append(slots, argumentSlot{start: loc.StartOffset(), end: loc.EndOffset(), parameter: parameter})
		}
	}
	return slots
}

func positionalParameter(index int, sig model.UntypedFunctionSignature) int {
	if index < sig.FixedParamCount() {
		return index
	}
	if sig.HasRest {
		return len(sig.ParamNames) - 1
	}
	return -1
}

// Zero-width bad expressions can mark missing arguments, as in foo(1, ).
func isMalformedArgument(arg ast.BLangExpression) bool {
	if _, bad := arg.(*ast.BLangBadExprOrAction); !bad {
		return false
	}
	loc := arg.GetPosition()
	return !locationHasUsableOffsets(loc) || loc.StartOffset() != loc.EndOffset()
}

func namedParameterIndex(named *ast.BLangNamedArgsExpression, sig model.UntypedFunctionSignature) (int, bool) {
	if named.Name == nil {
		return 0, false
	}
	name := strings.TrimPrefix(named.Name.GetValue(), "'")
	index := -1
	for i, formal := range sig.ParamNames {
		if strings.TrimPrefix(formal, "'") == name {
			if index >= 0 {
				return 0, false
			}
			index = i
		}
	}
	return index, index >= 0
}

// Failed initializer checks give the expression type never. Its explicit or declared
// target still identifies the constructor without committing compiler trial state.
func newExpressionType(cx *context.CompilerContext, node *ast.BLangNewExpression, chain []ast.BLangNode) semtypes.SemType {
	if !node.ClassSymbol.IsEmpty() {
		return cx.SymbolType(node.ClassSymbol)
	}
	if ty := node.Type(); !semtypes.IsZero(ty) && !semtypes.IsNever(ty) {
		return ty
	}
	if node.TypeDescriptor != nil {
		return node.TypeDescriptor.Type()
	}
	for _, parent := range chain {
		if variable, ok := parent.(*ast.BLangVariable); ok && variable.GetInitialExpression() == node {
			return variable.GetAssociatedType()
		}
	}
	return semtypes.SemType{}
}

func methodReference(cx *context.CompilerContext, receiver ast.BLangExpression, name string) (model.SymbolRef, bool) {
	if receiver == nil || semtypes.IsZero(receiver.Type()) || semtypes.IsNever(receiver.Type()) {
		return model.SymbolRef{}, false
	}
	atom, unique := semtypes.ObjectPositiveAtom(semtypes.ContextFrom(cx.GetTypeEnv()), receiver.Type())
	if !unique {
		return model.SymbolRef{}, false
	}
	table, ok := cx.ObjectMethodTable(atom)
	if !ok {
		return model.SymbolRef{}, false
	}
	ref, ok := table.Methods[name]
	return ref, ok
}

func callSignature(sm compile.SealedModule, node ast.BLangNode, chain []ast.BLangNode) (model.SymbolRef, string, model.UntypedFunctionSignature, bool) {
	cx := sm.Context()
	var raw model.Symbol
	switch n := node.(type) {
	case *ast.BLangInvocation:
		raw = n.RawSymbol
		if _, resolved := raw.(*model.SymbolRef); !resolved && n.Name != nil {
			if ref, ok := methodReference(cx, n.Expr, n.Name.GetValue()); ok {
				raw = &ref
			}
		}
	case *ast.BLangRemoteMethodCallAction:
		raw = n.RawSymbol
		if _, resolved := raw.(*model.SymbolRef); !resolved && n.Name != nil {
			if ref, ok := methodReference(cx, n.Expr, model.RemoteMethodName(n.Name.GetValue())); ok {
				raw = &ref
			}
		}
	case *ast.BLangNewExpression:
		ty := newExpressionType(cx, n, chain)
		if semtypes.IsZero(ty) {
			return model.SymbolRef{}, "", model.UntypedFunctionSignature{}, false
		}
		tc := semtypes.ContextFrom(cx.GetTypeEnv())
		atom, unique := semtypes.ObjectPositiveAtom(tc, ty)
		if !unique {
			return model.SymbolRef{}, "", model.UntypedFunctionSignature{}, false
		}
		table, ok := cx.ObjectMethodTable(atom)
		if !ok || table.Owner.IsEmpty() {
			return model.SymbolRef{}, "", model.UntypedFunctionSignature{}, false
		}
		if _, ok := cx.GetSymbol(table.Owner).(model.ClassSymbol); !ok {
			return model.SymbolRef{}, "", model.UntypedFunctionSignature{}, false
		}
		name := "new " + cx.SymbolName(table.Owner)
		ref, hasInit := table.Methods["init"]
		if !hasInit {
			return model.SymbolRef{}, name, model.UntypedFunctionSignature{}, true
		}
		if ref.IsEmpty() {
			return model.SymbolRef{}, "", model.UntypedFunctionSignature{}, false
		}
		sig, ok := cx.GetFunctionSignature(ref)
		return ref, name, sig, ok
	}
	ref, ok := raw.(*model.SymbolRef)
	if !ok || ref == nil || ref.IsEmpty() {
		return model.SymbolRef{}, "", model.UntypedFunctionSignature{}, false
	}
	sig, ok := cx.GetFunctionSignature(*ref)
	return *ref, model.StripRemotePrefix(cx.SymbolName(originRef(cx, *ref))), sig, ok
}

func originRef(cx *context.CompilerContext, ref model.SymbolRef) model.SymbolRef {
	if mono, ok := cx.GetSymbol(ref).(model.MonomorphicFunctionSymbol); ok {
		return mono.PolymorphicSymbol()
	}
	return ref
}

func render(sm compile.SealedModule, ref model.SymbolRef, name string, sig model.UntypedFunctionSignature, parameter int) Help {
	var typed model.TypedFunctionSignature
	ready := false
	if !ref.IsEmpty() && sm.Stage() >= compile.StageTopLevelTypeResolved {
		typed, ready = sm.Context().FunctionTypedSignature(ref)
	}
	if ready {
		count := len(sig.ParamNames)
		if sig.HasRest {
			count--
			ready = !semtypes.IsZero(typed.RestParamType)
		}
		ready = ready && len(typed.ParamTypes) == count
		for _, ty := range typed.ParamTypes {
			ready = ready && !semtypes.IsZero(ty)
		}
	}
	tc := semtypes.ContextFrom(sm.Context().GetTypeEnv())
	var label strings.Builder
	label.WriteString(name)
	label.WriteByte('(')
	comment := documentation(sm, ref)
	parameterDocs := parameterDescriptions(comment)
	help := Help{
		Description:            description(comment),
		Parameters:             make([][2]int, 0, len(sig.ParamNames)),
		ParameterDocumentation: make([]parameterDocumentation, 0, len(sig.ParamNames)),
	}
	for i, name := range sig.ParamNames {
		if i > 0 {
			label.WriteString(", ")
		}
		start := label.Len()
		doc := parameterDocumentation{Name: name, Description: parameterDocs[name]}
		rest := sig.HasRest && i == len(sig.ParamNames)-1
		if ready {
			ty := semtypes.SemType{}
			if rest {
				ty = typed.RestParamType
			} else if i < len(typed.ParamTypes) {
				ty = typed.ParamTypes[i]
			}
			if !semtypes.IsZero(ty) {
				doc.Type = semtypes.ToString(tc, ty)
				label.WriteString(doc.Type)
				if !rest {
					label.WriteByte(' ')
				}
			}
		}
		if rest {
			label.WriteString("...")
			if doc.Type != "" {
				doc.Type += "..."
			}
			if ready && !semtypes.IsZero(typed.RestParamType) {
				label.WriteByte(' ')
			}
		}
		label.WriteString(name)
		help.Parameters = append(help.Parameters, [2]int{start, label.Len()})
		help.ParameterDocumentation = append(help.ParameterDocumentation, doc)
	}
	label.WriteByte(')')
	help.Label = label.String()
	if parameter >= 0 && parameter < len(sig.ParamNames) {
		help.ActiveParameter = &parameter
	}
	return help
}
