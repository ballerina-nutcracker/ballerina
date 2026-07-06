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

package model

import (
	"ballerina-lang-go/semtypes"
)

// OpaqueSymbol is a symbol whose definition cannot be written in Ballerina
// source, so its real form is built in Go by OpaqueSymbols. It is serialized by
// its package-scoped index (OpaqueID) rather than by its underlying symbol.
type OpaqueSymbol interface {
	Symbol
	// OpaqueID is the symbol's index into OpaqueSymbols(pkg PackageIdentifier) for its
	// package. Together with the owning package it uniquely identifies the
	// symbol and is the serialization handle.
	OpaqueID() int
}

// OpaqueFunctionSymbol represents a function that uses `typeParam` annotation.
// Actualy type validation and resolution of these functions can't be represented within
// normal ballerina typing rules. Instead we have logic implemented for these in the type resolver.
type OpaqueFunctionSymbol struct {
	name        string
	ID          int          // per-package opaque id; serialization handle and (with the package) selects the monomorphizer
	SymbolSpace *SymbolSpace // space the monomorphized function is added to
	// Monomorphization cache functions, if function it self don't support caching then function pointers are nil
	Lookup func(keys ...semtypes.SemType) (SymbolRef, bool)
	Store  func(ref SymbolRef, keys ...semtypes.SemType)
}

const (
	// lang.array
	OpaqueFnArrayPush = 0
	// lang.map
	OpaqueFnMapRemove = 0
	// lang.xml
	OpaqueFnXMLIterator = 4
)

func newOpaqueFunctionSymbol(name string, id int) *OpaqueFunctionSymbol {
	return &OpaqueFunctionSymbol{name: name, ID: id}
}

func (s *OpaqueFunctionSymbol) Name() string     { return s.name }
func (s *OpaqueFunctionSymbol) OpaqueID() int    { return s.ID }
func (s *OpaqueFunctionSymbol) Kind() SymbolKind { return SymbolKindFunction }
func (s *OpaqueFunctionSymbol) IsPublic() bool   { return true }
func (s *OpaqueFunctionSymbol) Type() semtypes.SemType {
	panic("opaque function must be monomorphized")
}

func (s *OpaqueFunctionSymbol) SetType(semtypes.SemType) {
	panic("opaque function must be monomorphized")
}

func (s *OpaqueFunctionSymbol) Copy() Symbol {
	panic("opaque function must be monomorphized")
}

var (
	_ Symbol       = &OpaqueFunctionSymbol{}
	_ OpaqueSymbol = &OpaqueFunctionSymbol{}
)

// OpaqueTypeSymbol wraps the real TypeSymbol built for a builtin lang library
// and carries its package-scoped opaque id. Embedding TypeSymbol forwards
// Name/Type/Kind/IsPublic/SetType/Copy to the wrapped symbol.
type OpaqueTypeSymbol struct {
	TypeSymbol
	opaqueID int
}

func (o *OpaqueTypeSymbol) OpaqueID() int { return o.opaqueID }

var (
	_ Symbol       = &OpaqueTypeSymbol{}
	_ OpaqueSymbol = &OpaqueTypeSymbol{}
)

func newOpaqueTypeSymbol(name string, ty semtypes.SemType, index int) *OpaqueTypeSymbol {
	ts := NewTypeSymbol(name, true)
	ts.SetType(ty)
	return &OpaqueTypeSymbol{TypeSymbol: ts, opaqueID: index}
}

// OpaqueSymbols returns the Go-defined symbols for a builtin lang library,
// built against env. The slice index is the symbol's opaque id, scoped to the
// package. Returns nil for a non-builtin package. The dispatch matches on
// organization and package name only, so it is independent of the bundle
// version.
func OpaqueSymbols(pkg PackageIdentifier) []Symbol {
	if pkg.Organization != "ballerina" {
		return nil
	}
	switch pkg.Package {
	case "lang.int":
		return langIntOpaqueSymbols()
	case "lang.string":
		return langStringOpaqueSymbols()
	case "lang.xml":
		return langXMLOpaqueSymbols()
	case "lang.array":
		return []Symbol{newOpaqueFunctionSymbol("push", OpaqueFnArrayPush)}
	case "lang.map":
		return []Symbol{newOpaqueFunctionSymbol("remove", OpaqueFnMapRemove)}
	default:
		return nil
	}
}

func langIntOpaqueSymbols() []Symbol {
	defs := []struct {
		name string
		ty   semtypes.SemType
	}{
		{"Signed8", semtypes.SINT8},
		{"Signed16", semtypes.SINT16},
		{"Signed32", semtypes.SINT32},
		{"Unsigned8", semtypes.UINT8},
		{"Unsigned16", semtypes.UINT16},
		{"Unsigned32", semtypes.UINT32},
	}
	syms := make([]Symbol, len(defs))
	for i, def := range defs {
		syms[i] = newOpaqueTypeSymbol(def.name, def.ty, i)
	}
	return syms
}

func langStringOpaqueSymbols() []Symbol {
	return []Symbol{newOpaqueTypeSymbol("Char", semtypes.CHAR, 0)}
}

func langXMLOpaqueSymbols() []Symbol {
	defs := []struct {
		name string
		ty   semtypes.SemType
	}{
		{"Element", semtypes.XML_ELEMENT},
		{"Comment", semtypes.XML_COMMENT},
		{"Text", semtypes.XML_TEXT},
		{"ProcessingInstruction", semtypes.XML_PI},
	}
	syms := make([]Symbol, len(defs)+1)
	for i, def := range defs {
		syms[i] = newOpaqueTypeSymbol(def.name, def.ty, i)
	}
	syms[OpaqueFnXMLIterator] = newOpaqueFunctionSymbol("iterator", OpaqueFnXMLIterator)
	return syms
}
