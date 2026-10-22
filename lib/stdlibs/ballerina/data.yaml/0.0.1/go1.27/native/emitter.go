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

// Port of jBallerina's Serializer and Emitter: toYamlString writes a value as YAML lines.

package native

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type writeOptions struct {
	indent          string
	canonical       bool
	useSingleQuotes bool
	forceQuotes     bool
	isStream        bool
	flowStyle       bool
}

func toYamlStringArray(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	config, _ := args[1].(*values.Map)
	indentation, _ := get(config, "indentationPolicy").(int64)
	if indentation < 0 {
		return createError(ctx, "count is negative: "+strconv.FormatInt(indentation, 10))
	}
	w := &yamlWriter{options: writeOptions{
		indent:          strings.Repeat(" ", int(indentation)),
		canonical:       get(config, "canonical") == true,
		useSingleQuotes: get(config, "useSingleQuotes") == true,
		forceQuotes:     get(config, "forceQuotes") == true,
		isStream:        get(config, "isStream") == true,
		flowStyle:       get(config, "flowStyle") == true,
	}}
	w.write(args[0])
	lines := make([]values.BalValue, len(w.lines))
	for i, line := range w.lines {
		lines[i] = line
	}
	env := ctx.TypeEnv()
	ty := defineArray(env, semtypes.String)
	return values.NewList(ty, semtypes.ToListAtomicType(env, ty), false, nil, len(lines), lines), nil
}

type yamlWriter struct {
	options writeOptions
	lines   []string
}

func (w *yamlWriter) write(v values.BalValue) {
	if list, ok := v.(*values.List); ok && w.options.isStream {
		for i := 0; i < list.Len(); i++ {
			if i > 0 {
				w.lines = append(w.lines, "---")
			}
			w.writeDocument(list.Get(i))
		}
		return
	}
	w.writeDocument(v)
}

func (w *yamlWriter) writeDocument(v values.BalValue) {
	switch v := v.(type) {
	case *values.Map:
		if w.options.flowStyle {
			w.lines = append(w.lines, w.flow(v))
			return
		}
		w.writeBlockMapping(v, "")
	case *values.List:
		if w.options.flowStyle {
			w.lines = append(w.lines, w.flow(v))
			return
		}
		w.writeBlockSequence(v, "")
	default:
		w.lines = append(w.lines, w.scalar(v))
	}
}

func (w *yamlWriter) seqTag() string {
	if w.options.canonical {
		return " !!seq"
	}
	return ""
}

func (w *yamlWriter) writeBlockMapping(m *values.Map, ws string) {
	for _, key := range m.Keys() {
		value, _ := m.Get(key)
		prefix := ws + w.scalar(key) + ":"
		switch value := value.(type) {
		case *values.Map:
			if w.options.flowStyle {
				w.lines = append(w.lines, prefix+" "+w.flow(value))
				continue
			}
			w.lines = append(w.lines, prefix)
			w.writeBlockMapping(value, ws+w.options.indent)
		case *values.List:
			if w.options.flowStyle {
				w.lines = append(w.lines, prefix+" "+w.flow(value))
				continue
			}
			w.lines = append(w.lines, prefix+w.seqTag())
			w.writeBlockSequence(value, ws)
		default:
			w.lines = append(w.lines, prefix+" "+w.scalar(value))
		}
	}
}

func (w *yamlWriter) writeBlockSequence(list *values.List, ws string) {
	if list.Len() == 0 {
		w.lines = append(w.lines, ws+"-"+w.seqTag())
		return
	}
	for i := 0; i < list.Len(); i++ {
		switch item := list.Get(i).(type) {
		case *values.Map:
			if w.options.flowStyle {
				w.lines = append(w.lines, ws+"- "+w.flow(item))
				continue
			}
			w.lines = append(w.lines, ws+"-")
			w.writeBlockMapping(item, ws+w.options.indent)
		case *values.List:
			if w.options.flowStyle {
				w.lines = append(w.lines, ws+"- "+w.flow(item))
				continue
			}
			w.lines = append(w.lines, ws+"-"+w.seqTag())
			w.writeBlockSequence(item, ws+w.options.indent)
		default:
			w.lines = append(w.lines, ws+"- "+w.scalar(item))
		}
	}
}

func (w *yamlWriter) flow(v values.BalValue) string {
	switch v := v.(type) {
	case *values.Map:
		entries := make([]string, 0, v.Len())
		for _, key := range v.Keys() {
			value, _ := v.Get(key)
			entries = append(entries, w.scalar(key)+": "+w.flow(value))
		}
		return "{" + strings.Join(entries, ", ") + "}"
	case *values.List:
		items := make([]string, v.Len())
		for i := range items {
			items[i] = w.flow(v.Get(i))
		}
		prefix := ""
		if w.options.canonical {
			prefix = "!!seq "
		}
		return prefix + "[" + strings.Join(items, ", ") + "]"
	}
	return w.scalar(v)
}

// scalar writes a scalar as jBallerina's emitter does: its string form, with the !!str tag when
// canonical. Strings are quoted when forceQuotes is set or when they would not read back as the same
// string.
func (w *yamlWriter) scalar(v values.BalValue) string {
	text, isString := scalarText(v)
	if w.options.forceQuotes || (isString && needsQuotes(text)) {
		text = w.quote(text)
	}
	if w.options.canonical {
		return "!!str " + text
	}
	return text
}

func scalarText(v values.BalValue) (string, bool) {
	switch v := v.(type) {
	case nil:
		return "null", false
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), false
	case int64:
		return strconv.FormatInt(v, 10), false
	case float64:
		return javaDoubleString(v), false
	case *decimal.Decimal:
		return v.String(), false
	}
	return values.String(v, make(map[uintptr]bool)), false
}

func (w *yamlWriter) quote(text string) string {
	if w.options.useSingleQuotes && !strings.ContainsFunc(text, isControl) {
		return "'" + strings.ReplaceAll(text, "'", "''") + "'"
	}
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range text {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if isControl(r) {
				fmt.Fprintf(&sb, `\u%04X`, r)
				continue
			}
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func isControl(r rune) bool {
	return r < 0x20 || r == 0x7F || (r >= 0x80 && r < 0xA0) || r == '\uFEFF'
}

// needsQuotes reports whether a string written as a plain scalar would be read back differently: it is
// empty, has leading or trailing white space or control characters, starts with an indicator, contains
// ": " or " #", or would be read as null, a boolean or a number.
func needsQuotes(text string) bool {
	if text == "" || strings.ContainsFunc(text, isControl) || strings.TrimSpace(text) != text {
		return true
	}
	if strings.ContainsRune("-?:,[]{}#&*!|>'\"%@`", rune(text[0])) {
		return true
	}
	if strings.Contains(text, ": ") || strings.Contains(text, " #") || strings.HasSuffix(text, ":") {
		return true
	}
	if isAnyOf(text, "null", "Null", "NULL", "~", "true", "True", "TRUE", "false", "False", "FALSE") {
		return true
	}
	if _, ok := parseJavaLong(text); ok {
		return true
	}
	_, ok := parseJavaDouble(text)
	return ok
}
