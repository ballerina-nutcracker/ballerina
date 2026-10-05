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

package signature

import (
	stdcontext "context"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Help contains byte ranges within Label, independent of wire coordinates.
type Help struct {
	Label           string
	Parameters      [][2]int
	ActiveParameter *int
}

func At(ctx stdcontext.Context, sm compile.SealedModule, uri workspace.DocumentURI, offset int) (Help, bool, error) {
	if err := ctx.Err(); err != nil {
		return Help{}, false, err
	}
	text, ok := sm.SourceText(uri)
	if !ok || offset < 0 || offset > len(text) {
		return Help{}, false, nil
	}
	fileIndex := sm.Context().DiagnosticEnv().FileIndex(uri.Path())
	paths := nodeChainsAtOffset(sm.PackageNode(), offset, fileIndex)
	node, args, empty := callAtPaths(paths, fileIndex)
	if node == nil {
		return Help{}, false, nil
	}
	ref, name, sig, ok := callSignature(sm, node)
	if !ok {
		return Help{}, false, nil
	}
	argument := 0
	if !empty {
		argument, ok = argumentIndex(node, args, paths[0], offset, fileIndex)
		if !ok {
			return Help{}, false, nil
		}
		argument, ok = parameterIndex(args, argument, sig)
		if !ok {
			return Help{}, false, nil
		}
	}
	help := render(sm, ref, name, sig, argument)
	return help, true, ctx.Err()
}

func callAtPaths(paths [][]ast.BLangNode, fileIndex int) (ast.BLangNode, []ast.BLangExpression, bool) {
	if len(paths) == 0 {
		return nil, nil, false
	}
	node, args := innermostCall(paths[0])
	empty := emptyCallContext(paths, node, args, fileIndex)
	if len(paths) > 1 && !empty {
		return nil, nil, false
	}
	return node, args, empty
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
			if !callWideDefault(path[i], loc) {
				return false
			}
		}
	}
	return true
}

func callWideDefault(node ast.BLangNode, callLoc ast.Location) bool {
	if _, ok := node.(*ast.BLangDefaultArg); !ok {
		return false
	}
	loc := node.GetPosition()
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

func argumentIndex(call ast.BLangNode, args []ast.BLangExpression, chain []ast.BLangNode, offset, fileIndex int) (int, bool) {
	callLoc := call.GetPosition()
	index := -1
	for i, arg := range args {
		if arg == nil {
			continue
		}
		if _, bad := arg.(*ast.BLangBadExprOrAction); bad {
			continue
		}
		loc := arg.GetPosition()
		if !locationHasUsableOffsets(loc) || loc.FileIndex() != fileIndex || !locationContains(loc, offset) ||
			loc.StartOffset() < callLoc.StartOffset() || loc.EndOffset() > callLoc.EndOffset() ||
			loc.StartOffset() == callLoc.StartOffset() && loc.EndOffset() == callLoc.EndOffset() {
			continue
		}
		if index >= 0 {
			return 0, false
		}
		index = i
	}
	if index < 0 {
		return 0, false
	}
	for _, node := range chain {
		if node == args[index] {
			return index, true
		}
	}
	return 0, false
}

func parameterIndex(args []ast.BLangExpression, argument int, sig model.UntypedFunctionSignature) (int, bool) {
	seenNamed := false
	used := make(map[int]bool)
	active := argument
	for i, arg := range args {
		named, ok := arg.(*ast.BLangNamedArgsExpression)
		if !ok {
			if seenNamed {
				return 0, false
			}
			used[i] = true
			continue
		}
		seenNamed = true
		index, ok := namedParameterIndex(named, sig)
		if !ok || used[index] {
			return 0, false
		}
		used[index] = true
		if i == argument {
			active = index
		}
	}
	return active, true
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

func callSignature(sm compile.SealedModule, node ast.BLangNode) (model.SymbolRef, string, model.UntypedFunctionSignature, bool) {
	cx := sm.Context()
	var raw model.Symbol
	switch n := node.(type) {
	case *ast.BLangInvocation:
		raw = n.RawSymbol
	case *ast.BLangRemoteMethodCallAction:
		raw = n.RawSymbol
	case *ast.BLangNewExpression:
		ty := n.GetDeterminedType()
		if !n.ClassSymbol.IsEmpty() {
			ty = cx.SymbolType(n.ClassSymbol)
		}
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
	return *ref, model.StripRemotePrefix(cx.SymbolName(*ref)), sig, ok
}

func render(sm compile.SealedModule, ref model.SymbolRef, name string, sig model.UntypedFunctionSignature, argument int) Help {
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
	help := Help{Parameters: make([][2]int, 0, len(sig.ParamNames))}
	for i, name := range sig.ParamNames {
		if i > 0 {
			label.WriteString(", ")
		}
		start := label.Len()
		rest := sig.HasRest && i == len(sig.ParamNames)-1
		if ready {
			ty := semtypes.SemType{}
			if rest {
				ty = typed.RestParamType
			} else if i < len(typed.ParamTypes) {
				ty = typed.ParamTypes[i]
			}
			if !semtypes.IsZero(ty) {
				label.WriteString(semtypes.ToString(tc, ty))
				if !rest {
					label.WriteByte(' ')
				}
			}
		}
		if rest {
			label.WriteString("...")
			if ready && !semtypes.IsZero(typed.RestParamType) {
				label.WriteByte(' ')
			}
		}
		label.WriteString(name)
		help.Parameters = append(help.Parameters, [2]int{start, label.Len()})
	}
	label.WriteByte(')')
	help.Label = label.String()
	if len(sig.ParamNames) > 0 {
		active := min(argument, len(sig.ParamNames)-1)
		help.ActiveParameter = &active
	}
	return help
}
