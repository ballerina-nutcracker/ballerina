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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"sort"
	"strings"
)

// OutputPaths names the generated files.
const (
	TypesFileName = "types_generated.go"
	JSONFileName  = "json_generated.go"
)

// Generate emits the generated Go source for the parsed metamodel.
func Generate(pm *ParsedModel, modelBytes []byte) ([]byte, []byte, error) {
	reg, err := NewTypeRegistry(pm)
	if err != nil {
		return nil, nil, err
	}
	detector := NewRecursionDetector(pm, reg)
	gen := &generator{
		pm:       pm,
		reg:      reg,
		rec:      detector,
		modelSum: sha256sum(modelBytes),
	}
	typesSrc, err := gen.generateTypes()
	if err != nil {
		return nil, nil, fmt.Errorf("generate types: %w", err)
	}
	jsonSrc, err := gen.generateJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("generate json: %w", err)
	}
	return typesSrc, jsonSrc, nil
}

type generator struct {
	pm       *ParsedModel
	reg      *TypeRegistry
	rec      *RecursionDetector
	modelSum string
}

func (g *generator) generateTypes() ([]byte, error) {
	var out bytes.Buffer
	g.writeHeader(&out)
	g.writePresenceWrappers(&out)
	g.writeBaseAliases(&out)
	for _, a := range g.reg.SortedAliases() {
		g.writeAlias(&out, a)
	}
	for _, e := range g.reg.SortedEnumerations() {
		g.writeEnumeration(&out, e)
	}
	for _, t := range g.reg.GeneratedTypes() {
		g.writeGeneratedType(&out, t)
	}
	for _, s := range g.reg.SortedStructures() {
		g.writeStructure(&out, s)
	}
	g.writeRequestNotificationTypes(&out)
	return g.format(out.Bytes())
}

func (g *generator) generateJSON() ([]byte, error) {
	var out bytes.Buffer
	g.writeJSONHeader(&out)
	g.writeUnionJSON(&out)
	return g.format(out.Bytes())
}

func (g *generator) format(src []byte) ([]byte, error) {
	formatted, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("format: %w\n--- source ---\n%s", err, src)
	}
	return formatted, nil
}

func (g *generator) writeHeader(out io.Writer) {
	fmt.Fprintf(out, headerTemplate,
		g.pm.Model.Version.Version,
		"25005c80d9ec5e366c51108a4981ef264fe058e7",
		g.modelSum,
	)
}

const headerTemplate = `// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
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

// Code generated from the Language Server Protocol 3.18 metamodel.
// LSP version: %s
// Source commit: %s
// Metamodel SHA-256: %s
// DO NOT EDIT.

package protocol

`

func (g *generator) writeJSONHeader(out io.Writer) {
	fmt.Fprint(out, `// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
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

// Code generated from the Language Server Protocol 3.18 metamodel.
// DO NOT EDIT.

package protocol

import (
	"encoding/json"
	"fmt"
)

`)
}

func (g *generator) writePresenceWrappers(out io.Writer) {
	fmt.Fprint(out, `// Optional represents an optional non-null field.
type Optional[T any] struct {
	value *T
}

// IsZero reports whether the optional field is unset.
func (o Optional[T]) IsZero() bool {
	return o.value == nil
}

// NewOptional creates an Optional holding v.
func NewOptional[T any](v T) Optional[T] {
	return Optional[T]{value: &v}
}

// IsSet reports whether the optional field was present.
func (o Optional[T]) IsSet() bool {
	return o.value != nil
}

// Value returns the value and true if set; otherwise the zero value and false.
func (o Optional[T]) Value() (T, bool) {
	if o.value == nil {
		var zero T
		return zero, false
	}
	return *o.value, true
}

// Or returns the value if set, otherwise def.
func (o Optional[T]) Or(def T) T {
	if o.value == nil {
		return def
	}
	return *o.value
}

// Nullable represents a required nullable field.
type Nullable[T any] struct {
	value *T
	null  bool
}

// IsZero reports whether the nullable field is unset (neither value nor explicit null).
func (n Nullable[T]) IsZero() bool {
	return n.value == nil && !n.null
}

// NewNullable creates a Nullable holding v.
func NewNullable[T any](v T) Nullable[T] {
	return Nullable[T]{value: &v}
}

// NullNullable returns an explicit JSON null.
func NullNullable[T any]() Nullable[T] {
	return Nullable[T]{null: true}
}

// IsNull reports whether the value is explicit null.
func (n Nullable[T]) IsNull() bool {
	return n.null
}

// IsSet reports whether the nullable holds a concrete value (not null).
func (n Nullable[T]) IsSet() bool {
	return n.value != nil
}

// Value returns the concrete value and true if set.
func (n Nullable[T]) Value() (T, bool) {
	if n.value == nil {
		var zero T
		return zero, false
	}
	return *n.value, true
}

// OptionalNullable represents an optional nullable field.
type OptionalNullable[T any] struct {
	value *T
	null  bool
}

// IsZero reports whether the optional nullable field is unset.
func (o OptionalNullable[T]) IsZero() bool {
	return o.value == nil && !o.null
}

// NewOptionalNullable creates an OptionalNullable holding v.
func NewOptionalNullable[T any](v T) OptionalNullable[T] {
	return OptionalNullable[T]{value: &v}
}

// NullOptionalNullable returns an explicit JSON null optional.
func NullOptionalNullable[T any]() OptionalNullable[T] {
	return OptionalNullable[T]{null: true}
}

// IsSet reports whether the field was present (value or explicit null).
func (o OptionalNullable[T]) IsSet() bool {
	return o.value != nil || o.null
}

// IsNull reports whether the present value is explicit null.
func (o OptionalNullable[T]) IsNull() bool {
	return o.null
}

// Value returns the concrete value and true if set to a value.
func (o OptionalNullable[T]) Value() (T, bool) {
	if o.value == nil {
		var zero T
		return zero, false
	}
	return *o.value, true
}

`)
}

func (g *generator) writeBaseAliases(out io.Writer) {
	fmt.Fprint(out, "// DocumentURI is the LSP DocumentUri base type.\ntype DocumentURI = string\n\n")
}

func (g *generator) writeAlias(out io.Writer, a *TypeAlias) {
	if isAnonymousType(a.Type) {
		// The alias name owns the anonymous underlying type; the declaration is
		// emitted by writeGeneratedType using the alias name.
		return
	}
	g.writeDoc(out, a.Documentation, a.Since, a.Deprecated)
	name := GoName(a.Name)
	underlying := g.reg.nameType(a.Type, []string{a.Name})
	fmt.Fprintf(out, "type %s = %s\n\n", name, underlying)
}

func (g *generator) writeEnumeration(out io.Writer, e *Enumeration) {
	g.writeDoc(out, e.Documentation, e.Since, e.Deprecated)
	name := GoName(e.Name)
	var underlying string
	if e.Type != nil {
		underlying = g.reg.nameType(e.Type, []string{e.Name})
	}
	if underlying == "" {
		underlying = "int32"
	}
	fmt.Fprintf(out, "type %s %s\n\n", name, underlying)
	for _, v := range e.Values {
		g.writeDoc(out, v.Documentation, v.Since, v.Deprecated)
		constName := EnumConstName(e.Name, v.Name)
		fmt.Fprintf(out, "const %s %s = ", constName, name)
		switch val := v.Value.(type) {
		case float64:
			fmt.Fprintf(out, "%v\n", val)
		case string:
			fmt.Fprintf(out, "%q\n", val)
		default:
			fmt.Fprintf(out, "%v\n", v.Value)
		}
	}
	fmt.Fprintln(out)
}

func (g *generator) writeGeneratedType(out io.Writer, t *Type) {
	name, ok := g.reg.GeneratedName(t)
	if !ok {
		return
	}
	switch t.Kind {
	case KindLiteral:
		g.writeLiteralType(out, name, t)
	case KindAnd:
		g.writeIntersectionType(out, name, t)
	case KindOr:
		g.writeUnionType(out, name, t)
	case KindTuple:
		g.writeTupleType(out, name, t)
	}
}

func (g *generator) writeLiteralType(out io.Writer, name string, t *Type) {
	props := LiteralProperties(t)
	fmt.Fprintf(out, "// %s is a generated anonymous literal type.\ntype %s struct {\n", name, name)
	for _, p := range props {
		g.writeDoc(out, p.Documentation, p.Since, p.Deprecated)
		field := FieldName(p.Name)
		goType := g.fieldGoType(p.Type, p.Optional, []string{name, p.Name})
		tag := g.jsonTag(p.Name, ClassifyPresence(p.Optional, p.Type))
		fmt.Fprintf(out, "\t%s %s %s\n", field, goType, tag)
	}
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeIntersectionType(out io.Writer, name string, t *Type) {
	fmt.Fprintf(out, "// %s is a generated intersection type.\ntype %s struct {\n", name, name)
	for _, item := range t.Items {
		itemName, ok := g.reg.GeneratedName(item)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "\t%s\n", itemName)
	}
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeUnionType(out io.Writer, name string, t *Type) {
	fmt.Fprintf(out, "// %s is a generated union type.\ntype %s struct {\n", name, name)
	fmt.Fprintln(out, "\tvalue any")
	fmt.Fprintln(out, "\ttag int")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
	items := NullFreeItems(t)
	for i, item := range items {
		goType := g.itemGoType(item, []string{name, fmt.Sprintf("Item%d", i)})
		if goType == "" {
			goType = "any"
		}
		fmt.Fprintf(out, "// New%s%s constructs the %s variant of %s.\n", name, variantName(item, i), TypeName(item), name)
		fmt.Fprintf(out, "func New%s%s(v %s) %s {\n", name, variantName(item, i), goType, name)
		fmt.Fprintf(out, "\treturn %s{value: v, tag: %d}\n", name, i)
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
}

func (g *generator) writeTupleType(out io.Writer, name string, t *Type) {
	fmt.Fprintf(out, "// %s is a generated tuple type.\ntype %s struct {\n", name, name)
	for i, item := range t.Items {
		goType := g.itemGoType(item, []string{name, fmt.Sprintf("Item%d", i)})
		if goType == "" {
			goType = "any"
		}
		fmt.Fprintf(out, "\tItem%d %s\n", i, goType)
	}
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeStructure(out io.Writer, s *Structure) {
	g.writeDoc(out, s.Documentation, s.Since, s.Deprecated)
	name := GoName(s.Name)
	fmt.Fprintf(out, "type %s struct {\n", name)
	props := g.reg.FlattenedProperties(s.Name)
	for _, p := range props {
		g.writeDoc(out, p.Documentation, p.Since, p.Deprecated)
		field := FieldName(p.Name)
		goType := g.fieldGoType(p.Type, p.Optional, []string{name, p.Name})
		tag := g.jsonTag(p.Name, ClassifyPresence(p.Optional, p.Type))
		fmt.Fprintf(out, "\t%s %s %s\n", field, goType, tag)
	}
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeRequestNotificationTypes(out io.Writer) {
	// Anonymous request/notification params/result types that are not named in the
	// metamodel are already generated via the type registry. This section is reserved
	// for future request/notification metadata constants.
}

func (g *generator) fieldGoType(t *Type, optional bool, ctx []string) string {
	if t == nil {
		return "any"
	}
	p := ClassifyPresence(optional, t)
	inner := g.fieldInnerType(t, ctx)
	return PresenceWrapper(p, inner)
}

func (g *generator) fieldInnerType(t *Type, ctx []string) string {
	if t == nil {
		return "any"
	}
	return g.goTypeWithRecursion(t, ctx, false)
}

func (g *generator) itemGoType(t *Type, ctx []string) string {
	return g.goTypeWithRecursion(t, ctx, false)
}

func (g *generator) goTypeWithRecursion(t *Type, ctx []string, inArray bool) string {
	if t == nil {
		return "any"
	}
	switch t.Kind {
	case KindBase:
		return BaseTypeName(t.Name)
	case KindReference:
		name := GoName(t.Name)
		if g.rec.IsRecursive(name) && !inArray {
			return "*" + name
		}
		return name
	case KindStringLiteral:
		return "string"
	case KindIntegerLiteral, KindBooleanLiteral:
		return fmt.Sprintf("%v", t.Value)
	case KindArray:
		return "[]" + g.goTypeWithRecursion(t.Element, append(ctx, "Elem"), true)
	case KindMap:
		key := "string"
		if t.Key != nil {
			key = g.goTypeWithRecursion(t.Key, nil, false)
		}
		var val string
		if v, ok := t.Value.(*Type); ok {
			val = g.goTypeWithRecursion(v, append(ctx, "Value"), true)
		} else {
			val = "any"
		}
		return "map[" + key + "]" + val
	case KindAnd, KindOr, KindTuple, KindLiteral:
		name, ok := g.reg.GeneratedName(t)
		if !ok {
			return "any"
		}
		return name
	default:
		return "any"
	}
}

func (g *generator) jsonTag(jsonName string, p Presence) string {
	switch p {
	case OptionalNonNull, OptionalNullable:
		return fmt.Sprintf("`json:\"%s,omitzero\"`", jsonName)
	default:
		return fmt.Sprintf("`json:\"%s\"`", jsonName)
	}
}

func (g *generator) writeDoc(out io.Writer, doc, since, deprecated string) {
	lines := normalizeDoc(doc, since, deprecated)
	for _, line := range lines {
		fmt.Fprintf(out, "// %s\n", line)
	}
}

func normalizeDoc(doc, since, deprecated string) []string {
	var lines []string
	appendLines := func(text string) {
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			line = strings.ReplaceAll(line, "{@link ", "")
			line = strings.ReplaceAll(line, "}", "")
			lines = append(lines, line)
		}
	}
	appendLines(doc)
	appendTagged(since, "@since", &lines)
	appendTagged(deprecated, "Deprecated:", &lines)
	return lines
}

func appendTagged(text, prefix string, lines *[]string) {
	if text == "" {
		return
	}
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i == 0 {
			*lines = append(*lines, prefix+" "+part)
		} else {
			*lines = append(*lines, part)
		}
	}
}

func variantName(t *Type, idx int) string {
	switch t.Kind {
	case KindReference:
		return GoName(t.Name)
	case KindBase:
		return GoName(t.Name)
	case KindStringLiteral:
		return GoName(StringLiteralValue(t))
	case KindIntegerLiteral:
		return fmt.Sprintf("Integer%v", IntegerLiteralValue(t))
	case KindBooleanLiteral:
		return fmt.Sprintf("%v", t.Value)
	case KindLiteral:
		return fmt.Sprintf("Literal%d", idx)
	case KindArray:
		return fmt.Sprintf("Array%d", idx)
	case KindMap:
		return fmt.Sprintf("Map%d", idx)
	default:
		return fmt.Sprintf("Variant%d", idx)
	}
}

func sha256sum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sortStrings returns a sorted copy of s.
func sortStrings(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}

// jsonString returns a compact JSON representation of v.
func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
