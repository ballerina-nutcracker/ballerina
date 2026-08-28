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
	"io"
	"sort"
	"strings"
)

func (g *generator) writeUnionJSON(out io.Writer) {
	g.writeWrapperJSON(out)
	g.writeJSONHelpers(out)
	for _, s := range g.reg.SortedStructures() {
		g.writeStructureMatcher(out, GoName(s.Name), s)
	}
	for _, e := range g.reg.SortedEnumerations() {
		g.writeEnumerationMatcher(out, GoName(e.Name), e)
	}
	for _, a := range g.reg.SortedAliases() {
		g.writeAliasMatcher(out, a)
	}
	for _, t := range g.reg.GeneratedTypes() {
		if t.Kind == KindOr {
			g.writeUnionMarshalUnmarshal(out, t)
		}
	}
}

func (g *generator) writeWrapperJSON(out io.Writer) {
	fmt.Fprint(out, `func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.value == nil {
		return nil, fmt.Errorf("optional field is not set")
	}
	return json.Marshal(*o.value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return fmt.Errorf("optional field cannot be null or empty")
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = Optional[T]{value: &v}
	return nil
}

func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.null {
		return []byte("null"), nil
	}
	if n.value == nil {
		return nil, fmt.Errorf("nullable field has no value and is not null")
	}
	return json.Marshal(*n.value)
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("nullable field cannot be empty")
	}
	if string(data) == "null" {
		*n = Nullable[T]{null: true}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = NewNullable(v)
	return nil
}

func (o OptionalNullable[T]) MarshalJSON() ([]byte, error) {
	if !o.IsSet() {
		return nil, fmt.Errorf("optional nullable field is not set")
	}
	if o.null {
		return []byte("null"), nil
	}
	if o.value == nil {
		return nil, fmt.Errorf("optional nullable field is set without value or null")
	}
	return json.Marshal(*o.value)
}

func (o *OptionalNullable[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("optional nullable field cannot be empty")
	}
	if string(data) == "null" {
		*o = OptionalNullable[T]{null: true}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = NewOptionalNullable(v)
	return nil
}

`)
}

func (g *generator) writeJSONHelpers(out io.Writer) {
	fmt.Fprint(out, `func isString(raw any) bool {
	_, ok := raw.(string)
	return ok
}

func isNumber(raw any) bool {
	_, ok := raw.(float64)
	return ok
}

func isBool(raw any) bool {
	_, ok := raw.(bool)
	return ok
}

func isObject(raw any) bool {
	_, ok := raw.(map[string]any)
	return ok
}

func isArray(raw any) bool {
	_, ok := raw.([]any)
	return ok
}

func hasKey(raw any, key string) bool {
	obj, ok := raw.(map[string]any)
	if !ok || obj == nil {
		return false
	}
	_, has := obj[key]
	return has
}

func isStringLiteral(raw any, want string) bool {
	s, ok := raw.(string)
	return ok && s == want
}

func isIntegerLiteral(raw any, want int64) bool {
	n, ok := raw.(float64)
	return ok && int64(n) == want
}

`)
}

func requiredProps(props []Property) []string {
	var names []string
	for _, p := range props {
		if !p.Optional {
			names = append(names, p.Name)
		}
	}
	sort.Strings(names)
	return names
}

func (g *generator) writeStructureMatcher(out io.Writer, refName string, s *Structure) {
	props := g.reg.FlattenedProperties(s.Name)
	required := requiredProps(props)
	fmt.Fprintf(out, "func %sMatches(raw any) bool {\n", lowerFirst(refName))
	if len(required) == 0 {
		fmt.Fprintln(out, "\treturn isObject(raw)")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
		return
	}
	fmt.Fprintf(out, "\tobj, ok := raw.(map[string]any)\n")
	fmt.Fprintln(out, "\tif !ok { return false }")
	for _, name := range required {
		fmt.Fprintf(out, "\tif _, has := obj[%q]; !has { return false }\n", name)
	}
	fmt.Fprintln(out, "\treturn true")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeEnumerationMatcher(out io.Writer, refName string, e *Enumeration) {
	fmt.Fprintf(out, "func %sMatches(raw any) bool {\n", lowerFirst(refName))
	for _, v := range e.Values {
		switch val := v.Value.(type) {
		case float64:
			fmt.Fprintf(out, "\tif n, ok := raw.(float64); ok && int64(n) == %v { return true }\n", int64(val))
		case string:
			fmt.Fprintf(out, "\tif s, ok := raw.(string); ok && s == %q { return true }\n", val)
		}
	}
	fmt.Fprintln(out, "\treturn false")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeAliasMatcher(out io.Writer, a *TypeAlias) {
	fmt.Fprintf(out, "func %sMatches(raw any) bool {\n", lowerFirst(GoName(a.Name)))
	if a.Type.Kind == KindOr {
		for _, item := range NullFreeItems(a.Type) {
			fmt.Fprintf(out, "\tif %s { return true }\n", g.inlineVariantCondition(item, "raw"))
		}
		fmt.Fprintln(out, "\treturn false")
	} else {
		fmt.Fprintf(out, "\treturn %s\n", g.inlineVariantCondition(a.Type, "raw"))
	}
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func (g *generator) writeUnionMarshalUnmarshal(out io.Writer, t *Type) {
	name, ok := g.reg.GeneratedName(t)
	if !ok {
		return
	}
	items := NullFreeItems(t)

	fmt.Fprintf(out, "func (u %s) MarshalJSON() ([]byte, error) {\n", name)
	fmt.Fprintln(out, "\tswitch u.tag {")
	for i := range items {
		fmt.Fprintf(out, "\tcase %d:\n", i)
		fmt.Fprintln(out, "\t\treturn json.Marshal(u.value)")
	}
	fmt.Fprintln(out, "\tdefault:")
	fmt.Fprintln(out, "\t\treturn nil, fmt.Errorf(\"union has no selected variant\")")
	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)

	fmt.Fprintf(out, "func (u *%s) UnmarshalJSON(data []byte) error {\n", name)
	fmt.Fprintln(out, "\tif len(data) == 0 {")
	fmt.Fprintln(out, "\t\treturn fmt.Errorf(\"union cannot unmarshal empty data\")")
	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "\tvar raw any")
	fmt.Fprintln(out, "\tif err := json.Unmarshal(data, &raw); err != nil {")
	fmt.Fprintln(out, "\t\treturn err")
	fmt.Fprintln(out, "\t}")
	for i, item := range items {
		goType := g.itemGoType(item, []string{name, fmt.Sprintf("Item%d", i)})
		if goType == "" {
			goType = "any"
		}
		g.writeInlineVariantCheck(out, item, i, name, goType)
	}
	fmt.Fprintf(out, "\treturn fmt.Errorf(\"data does not match any variant of %%s\", string(data))\n")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)

	for i, item := range items {
		goType := g.itemGoType(item, []string{name, fmt.Sprintf("Item%d", i)})
		if goType == "" {
			goType = "any"
		}
		vn := variantName(item, i)
		fmt.Fprintf(out, "// %s returns the %s variant value and true if selected.\n", vn, TypeName(item))
		fmt.Fprintf(out, "func (u %s) %s() (%s, bool) {\n", name, vn, goType)
		fmt.Fprintf(out, "\tif u.tag != %d {\n", i)
		fmt.Fprintf(out, "\t\tvar zero %s\n", goType)
		fmt.Fprintln(out, "\t\treturn zero, false")
		fmt.Fprintln(out, "\t}")
		fmt.Fprintf(out, "\treturn u.value.(%s), true\n", goType)
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
}

func (g *generator) writeInlineVariantCheck(out io.Writer, item *Type, idx int, unionName, goType string) {
	cond := g.inlineVariantCondition(item, "raw")
	vn := variantName(item, idx)
	fmt.Fprintf(out, "\tif %s {\n", cond)
	fmt.Fprintf(out, "\t\tvar v %s\n", goType)
	fmt.Fprintln(out, "\t\tif err := json.Unmarshal(data, &v); err != nil {")
	fmt.Fprintln(out, "\t\t\treturn err")
	fmt.Fprintln(out, "\t\t}")
	fmt.Fprintf(out, "\t\t*u = New%s%s(v)\n", unionName, vn)
	fmt.Fprintln(out, "\t\treturn nil")
	fmt.Fprintln(out, "\t}")
}

func (g *generator) inlineVariantCondition(t *Type, rawExpr string) string {
	switch t.Kind {
	case KindBase:
		switch t.Name {
		case "string", "URI", "DocumentUri", "RegExp":
			return fmt.Sprintf("isString(%s)", rawExpr)
		case "integer", "uinteger", "decimal":
			return fmt.Sprintf("isNumber(%s)", rawExpr)
		case "boolean":
			return fmt.Sprintf("isBool(%s)", rawExpr)
		case "null":
			return fmt.Sprintf("%s == nil", rawExpr)
		}
	case KindReference:
		return fmt.Sprintf("%sMatches(%s)", lowerFirst(GoName(t.Name)), rawExpr)
	case KindStringLiteral:
		return fmt.Sprintf("isStringLiteral(%s, %q)", rawExpr, StringLiteralValue(t))
	case KindIntegerLiteral:
		return fmt.Sprintf("isIntegerLiteral(%s, %v)", rawExpr, IntegerLiteralValue(t))
	case KindBooleanLiteral:
		return fmt.Sprintf("isBool(%s) && %s.(bool) == %v", rawExpr, rawExpr, t.Value)
	case KindArray:
		return g.inlineArrayMatch(t, rawExpr)
	case KindMap:
		return fmt.Sprintf("isObject(%s)", rawExpr)
	case KindLiteral:
		return g.inlineLiteralMatch(t, rawExpr)
	case KindAnd:
		return g.inlineAndMatch(t, rawExpr)
	case KindOr:
		return g.inlineOrMatch(t, rawExpr)
	case KindTuple:
		return g.inlineTupleMatch(t, rawExpr)
	}
	return "false"
}

func (g *generator) inlineArrayMatch(t *Type, rawExpr string) string {
	elemExpr := fmt.Sprintf("%s.([]any)[0]", rawExpr)
	var elemCheck string
	switch t.Element.Kind {
	case KindBase:
		switch t.Element.Name {
		case "string", "URI", "DocumentUri", "RegExp":
			elemCheck = fmt.Sprintf("isString(%s)", elemExpr)
		case "integer", "uinteger", "decimal":
			elemCheck = fmt.Sprintf("isNumber(%s)", elemExpr)
		case "boolean":
			elemCheck = fmt.Sprintf("isBool(%s)", elemExpr)
		default:
			elemCheck = "true"
		}
	case KindReference:
		elemCheck = fmt.Sprintf("%sMatches(%s)", lowerFirst(GoName(t.Element.Name)), elemExpr)
	default:
		elemCheck = "true"
	}
	return fmt.Sprintf("isArray(%s) && (len(%s.([]any)) == 0 || %s)", rawExpr, rawExpr, elemCheck)
}

func (g *generator) inlineLiteralMatch(t *Type, rawExpr string) string {
	props := LiteralProperties(t)
	required := requiredProps(props)
	if len(required) == 0 {
		return fmt.Sprintf("isObject(%s)", rawExpr)
	}
	var checks []string
	checks = append(checks, fmt.Sprintf("isObject(%s)", rawExpr))
	for _, name := range required {
		checks = append(checks, fmt.Sprintf("hasKey(%s, %q)", rawExpr, name))
	}
	return strings.Join(checks, " && ")
}

func (g *generator) inlineAndMatch(t *Type, rawExpr string) string {
	var checks []string
	checks = append(checks, fmt.Sprintf("isObject(%s)", rawExpr))
	for _, item := range t.Items {
		if item.Kind != KindReference {
			continue
		}
		checks = append(checks, fmt.Sprintf("%sMatches(%s)", lowerFirst(GoName(item.Name)), rawExpr))
	}
	return strings.Join(checks, " && ")
}

func (g *generator) inlineOrMatch(t *Type, rawExpr string) string {
	var parts []string
	for _, item := range t.Items {
		parts = append(parts, g.inlineVariantCondition(item, rawExpr))
	}
	return strings.Join(parts, " || ")
}

func (g *generator) inlineTupleMatch(t *Type, rawExpr string) string {
	var checks []string
	checks = append(checks, fmt.Sprintf("isArray(%s) && len(%s.([]any)) == %d", rawExpr, rawExpr, len(t.Items)))
	for i, item := range t.Items {
		checks = append(checks, g.inlineVariantCondition(item, fmt.Sprintf("%s.([]any)[%d]", rawExpr, i)))
	}
	return strings.Join(checks, " && ")
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
