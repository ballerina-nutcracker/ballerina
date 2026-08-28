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

package generate

import (
	"fmt"
	"strings"
	"unicode"
)

// GoName converts a metamodel identifier into an exported Go identifier.
func GoName(s string) string {
	if s == "" {
		return ""
	}
	// LSP id/uri conventions
	switch s {
	case "id":
		return "ID"
	case "uri":
		return "URI"
	}
	if strings.HasSuffix(s, "Id") {
		s = s[:len(s)-2] + "ID"
	} else if strings.HasSuffix(s, "Uri") {
		s = s[:len(s)-3] + "URI"
	}
	// Export by uppercasing first rune.
	runes := []rune(s)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

// FieldName converts a metamodel property name into a Go struct field name.
func FieldName(s string) string {
	return GoName(s)
}

// EnumConstName returns the prefixed enum constant name.
func EnumConstName(enumName, entryName string) string {
	return GoName(enumName) + GoName(entryName)
}

// ContextName builds a deterministic contextual name for an anonymous type.
func ContextName(prefix string, path []string) string {
	parts := make([]string, 0, len(path)+1)
	parts = append(parts, prefix)
	for _, p := range path {
		// Sanitize path components into identifier pieces.
		parts = append(parts, sanitizePathPart(p))
	}
	nm := strings.Join(parts, "")
	// Avoid leading digits from numeric path parts.
	if len(nm) > 0 && nm[0] >= '0' && nm[0] <= '9' {
		nm = "X" + nm
	}
	return nm
}

func sanitizePathPart(s string) string {
	// Keep alphanumerics; capitalize after separators.
	var b strings.Builder
	capNext := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capNext {
				b.WriteRune(unicode.ToUpper(r))
				capNext = false
			} else {
				b.WriteRune(r)
			}
		} else {
			capNext = true
		}
	}
	if b.Len() == 0 {
		return "X"
	}
	return b.String()
}

// BaseTypeName maps LSP base type names to Go type expressions.
func BaseTypeName(name string) string {
	switch name {
	case "URI":
		return "string"
	case "DocumentUri":
		return "DocumentURI"
	case "integer":
		return "int32"
	case "uinteger":
		return "uint32"
	case "decimal":
		return "float64"
	case "RegExp":
		return "string"
	case "string":
		return "string"
	case "boolean":
		return "bool"
	case "null":
		return ""
	default:
		return name
	}
}

// Presence classifies how a property's Go type should be wrapped.
type Presence int

const (
	RequiredNonNull Presence = iota
	OptionalNonNull
	RequiredNullable
	OptionalNullable
)

// ClassifyPresence determines the presence wrapper for a property type.
func ClassifyPresence(optional bool, t *Type) Presence {
	nullable := IsNullableUnion(t)
	switch {
	case optional && nullable:
		return OptionalNullable
	case optional:
		return OptionalNonNull
	case nullable:
		return RequiredNullable
	default:
		return RequiredNonNull
	}
}

// PresenceWrapper returns the Go wrapper type name for a presence class.
func PresenceWrapper(p Presence, inner string) string {
	switch p {
	case OptionalNonNull:
		return "Optional[" + inner + "]"
	case RequiredNullable:
		return "Nullable[" + inner + "]"
	case OptionalNullable:
		return "OptionalNullable[" + inner + "]"
	default:
		return inner
	}
}

// IsDirectScalar reports whether a Go type expression names a scalar kind
// that is passed by value and can be stored directly inside an Optional.
func IsDirectScalar(goType string) bool {
	switch goType {
	case "bool", "string", "int", "int32", "int64", "uint32", "uint64", "float64", "DocumentURI":
		return true
	}
	return false
}

// IsEnumTypeName is used during generation to decide whether a type
// expression resolves to an enum. It is populated by the type graph.
var IsEnumTypeName func(name string) bool

// GoTypeFor returns the Go type expression for a resolved type, applying
// the requested presence wrapper when needed. nameCtx carries contextual
// naming path components for anonymous types.
func GoTypeFor(t *Type, p Presence, nameCtx []string, reg *TypeRegistry) string {
	if t == nil {
		return "any"
	}
	inner := goTypeNoPresence(t, nameCtx, reg)
	if inner == "" {
		return "any"
	}
	return PresenceWrapper(p, inner)
}

func goTypeNoPresence(t *Type, nameCtx []string, reg *TypeRegistry) string {
	if t == nil {
		return "any"
	}
	switch t.Kind {
	case KindBase:
		return BaseTypeName(t.Name)
	case KindReference:
		return GoName(t.Name)
	case KindStringLiteral:
		return "string"
	case KindIntegerLiteral, KindBooleanLiteral:
		return fmt.Sprintf("%v", t.Value)
	case KindArray:
		return "[]" + goTypeNoPresence(t.Element, append(nameCtx, "Elem"), reg)
	case KindMap:
		key := "string"
		if t.Key != nil {
			key = goTypeNoPresence(t.Key, nil, reg)
		}
		var val string
		if v, ok := t.Value.(*Type); ok {
			val = goTypeNoPresence(v, append(nameCtx, "Value"), reg)
		} else {
			val = "any"
		}
		return "map[" + key + "]" + val
	case KindAnd, KindOr, KindTuple, KindLiteral:
		if reg == nil {
			return "any"
		}
		name, ok := reg.GeneratedName(t)
		if !ok {
			return "any"
		}
		return name
	default:
		return "any"
	}
}
