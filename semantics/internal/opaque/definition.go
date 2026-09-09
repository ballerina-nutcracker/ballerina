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

// Package opaque defines the semantic data of opaque functions: lang-library
// functions whose definition cannot be written in Ballerina source, so their
// signature and monomorphization are implemented in Go. It owns each function's
// name, parameters, parameter flags and monomorphizer, and the per-resolver
// caches monomorphization needs. The package is internal to semantics; the
// symbol resolver uses it to give an opaque symbol its untyped signature, and
// the type resolver uses it to monomorphize an invocation.
package opaque

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// Resolve determines the type of an argument expression during
// monomorphization. An expression is resolved once for the whole compilation, so
// the caller's chain of narrowed bindings advances with each call and the effects
// of these resolutions are the only ones the invocation will ever produce. A
// monomorphizer must therefore resolve arguments in the order they are written.
type Resolve func(expr ast.BLangExpression, expected semtypes.SemType) (semtypes.SemType, bool)

// Materialize builds the monomorphic function symbol for sig and returns its
// ref. It is called only on a monomorphization-cache miss.
type Materialize func(signature model.TypedFunctionSignature) (model.SymbolRef, bool)

// SemanticError reports a user-facing failure at location.
type SemanticError func(message string, location diagnostics.Location)

type packageKey struct{ org, pkg string }

type monomorphizer func(
	ctx *Context,
	owner cacheOwner,
	resolve Resolve,
	materialize Materialize,
	semanticError SemanticError,
	isolated bool,
	args []ast.BLangExpression,
	expected semtypes.SemType,
	position diagnostics.Location,
) (model.SymbolRef, bool)

// FunctionDefinition is the single source of truth for one opaque function: its
// lang-library name, its parameters and their flags, and its monomorphizer. It
// is immutable; the package-scoped opaque id is the definition's position in its
// package's table rather than a field.
type FunctionDefinition struct {
	name   string
	params []model.Param
	// sourceDeclared marks a function whose signature is declared in the lang
	// library's own source, in a declaration marked @opaque, instead of by params
	// here. That is how a definition gets parameter defaults: the declared default is
	// parsed, resolved and desugared into a $default$N provider like any other
	// function's, which cannot be done from Go.
	sourceDeclared bool
	monomorphize   monomorphizer
	// owner is the definition's table slot, filled in by LookupFunction. It
	// partitions the monomorphization cache; see cacheOwner.
	owner cacheOwner
}

// Name is the function's lang-library name. It equals the name of the
// corresponding model.OpaqueSymbols entry.
func (d FunctionDefinition) Name() string { return d.name }

// Params returns a fresh copy of the parameter list, so callers cannot mutate
// the definition.
func (d FunctionDefinition) Params() []model.Param {
	params := make([]model.Param, len(d.params))
	copy(params, d.params)
	return params
}

// IsSourceDeclared reports whether the function's signature comes from a
// declaration in the lang library's source rather than from Params.
func (d FunctionDefinition) IsSourceDeclared() bool { return d.sourceDeclared }

// HasRest reports whether the last parameter is a rest parameter.
func (d FunctionDefinition) HasRest() bool {
	if len(d.params) == 0 {
		return false
	}
	return d.params[len(d.params)-1].Flag&model.ParamFlagRestParam != 0
}

// Monomorphize selects the concrete function symbol for a call site. args are
// positional: named arguments have already been lowered into their parameter
// slots. It reports user-facing failures through semanticError and returns
// false. isolated is the call site's isolation state.
func (d FunctionDefinition) Monomorphize(
	ctx *Context,
	resolve Resolve,
	materialize Materialize,
	semanticError SemanticError,
	isolated bool,
	args []ast.BLangExpression,
	expected semtypes.SemType,
	position diagnostics.Location,
) (model.SymbolRef, bool) {
	return d.monomorphize(ctx, d.owner, resolve, materialize, semanticError, isolated, args, expected, position)
}

// LookupFunction returns the definition of the opaque function with the given
// package-scoped id. It reports false for a package with no opaque functions,
// for an out-of-range id, and for an id whose slot holds an opaque type symbol.
// The package version is not part of the key, matching model.OpaqueSymbols.
func LookupFunction(org, pkg string, id int) (FunctionDefinition, bool) {
	key := packageKey{org: org, pkg: pkg}
	definitions, ok := functionDefinitions[key]
	if !ok || id < 0 || id >= len(definitions) {
		return FunctionDefinition{}, false
	}
	definition := definitions[id]
	if definition == nil {
		return FunctionDefinition{}, false
	}
	found := *definition
	found.owner = cacheOwner{pkg: key, id: id}
	return found, true
}
