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
	"regexp"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
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

func toYamlString(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	config, _ := args[1].(*values.Map)
	indentation, _ := get(config, "indentationPolicy").(int64)
	if indentation < 0 {
		return createError(ctx, "count is negative: "+strconv.FormatInt(indentation, 10))
	}
	if indentation > maxIndentation {
		return createError(ctx, "indentationPolicy must not be greater than "+strconv.Itoa(maxIndentation))
	}
	w := &yamlWriter{
		options: writeOptions{
			indent:          strings.Repeat(" ", int(indentation)),
			canonical:       get(config, "canonical") == true,
			useSingleQuotes: get(config, "useSingleQuotes") == true,
			forceQuotes:     get(config, "forceQuotes") == true,
			isStream:        get(config, "isStream") == true,
			flowStyle:       get(config, "flowStyle") == true,
		},
		visiting: make(map[values.BalValue]bool),
	}
	w.write(args[0])
	if w.cyclic {
		return createError(ctx, "cannot write a value that has a cyclic reference")
	}
	if w.unindentedNesting {
		return createError(ctx, "indentationPolicy must be greater than 0 to write a nested block collection")
	}
	return strings.Join(w.lines, "\n"), nil
}

const maxIndentation = 1024

type yamlWriter struct {
	options           writeOptions
	lines             []string
	unindentedNesting bool
	// visiting holds the maps and lists being written, to detect a value that contains itself.
	visiting map[values.BalValue]bool
	cyclic   bool
}

// enter marks a container as being written. It returns false, and records the cycle, when the container
// is already being written.
func (w *yamlWriter) enter(v values.BalValue) bool {
	if w.cyclic || w.visiting[v] {
		w.cyclic = true
		return false
	}
	w.visiting[v] = true
	return true
}

func (w *yamlWriter) leave(v values.BalValue) {
	delete(w.visiting, v)
}

// inFlow reports whether v is a collection written in flow style: always with flowStyle, and when empty,
// since a block collection cannot be empty.
func (w *yamlWriter) inFlow(v values.BalValue) bool {
	switch v := v.(type) {
	case *values.Map:
		return w.options.flowStyle || v.Len() == 0
	case *values.List:
		return w.options.flowStyle || v.Len() == 0
	}
	return false
}

// nestedIndent is the indentation of a block collection nested under ws. A sequence under a mapping key
// is written at the key's indentation instead, so it does not use this.
func (w *yamlWriter) nestedIndent(ws string) string {
	if w.options.indent == "" {
		w.unindentedNesting = true
	}
	return ws + w.options.indent
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
	if w.inFlow(v) {
		w.lines = append(w.lines, w.flow(v))
		return
	}
	switch v := v.(type) {
	case *values.Map:
		w.writeBlockMapping(v, "")
	case *values.List:
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
	if !w.enter(m) {
		return
	}
	defer w.leave(m)
	for _, key := range m.Keys() {
		value, _ := m.Get(key)
		prefix := ws + w.scalar(key) + ":"
		if w.inFlow(value) {
			w.lines = append(w.lines, prefix+" "+w.flow(value))
			continue
		}
		switch value := value.(type) {
		case *values.Map:
			w.lines = append(w.lines, prefix)
			w.writeBlockMapping(value, w.nestedIndent(ws))
		case *values.List:
			w.lines = append(w.lines, prefix+w.seqTag())
			w.writeBlockSequence(value, ws)
		default:
			w.lines = append(w.lines, prefix+" "+w.scalar(value))
		}
	}
}

func (w *yamlWriter) writeBlockSequence(list *values.List, ws string) {
	if !w.enter(list) {
		return
	}
	defer w.leave(list)
	for i := 0; i < list.Len(); i++ {
		item := list.Get(i)
		if w.inFlow(item) {
			w.lines = append(w.lines, ws+"- "+w.flow(item))
			continue
		}
		switch item := item.(type) {
		case *values.Map:
			w.lines = append(w.lines, ws+"-")
			w.writeBlockMapping(item, w.nestedIndent(ws))
		case *values.List:
			w.lines = append(w.lines, ws+"-"+w.seqTag())
			w.writeBlockSequence(item, w.nestedIndent(ws))
		default:
			w.lines = append(w.lines, ws+"- "+w.scalar(item))
		}
	}
}

func (w *yamlWriter) flow(v values.BalValue) string {
	switch v := v.(type) {
	case *values.Map:
		if !w.enter(v) {
			return ""
		}
		defer w.leave(v)
		entries := make([]string, 0, v.Len())
		for _, key := range v.Keys() {
			value, _ := v.Get(key)
			entries = append(entries, w.flowScalar(key)+": "+w.flow(value))
		}
		return "{" + strings.Join(entries, ", ") + "}"
	case *values.List:
		if !w.enter(v) {
			return ""
		}
		defer w.leave(v)
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
	return w.flowScalar(v)
}

// scalar writes a scalar as jBallerina's emitter does: its string form, with the !!str tag when
// canonical. Strings are quoted when forceQuotes is set or when they would not read back as the same
// string.
func (w *yamlWriter) scalar(v values.BalValue) string {
	return w.scalarIn(v, false)
}

func (w *yamlWriter) flowScalar(v values.BalValue) string {
	return w.scalarIn(v, true)
}

func (w *yamlWriter) scalarIn(v values.BalValue, inFlow bool) string {
	text, isString := scalarText(v)
	if w.options.forceQuotes || (isString && (needsQuotes(text) || inFlow && strings.ContainsAny(text, ",[]{}"))) {
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
// empty, has leading or trailing white space or control characters, starts with an indicator or the
// document-end marker, contains ": " or " #", or would be read as null, a boolean or a number. Inside a
// flow collection a flow indicator anywhere in the string also needs quotes.
func needsQuotes(text string) bool {
	if text == "" || strings.ContainsFunc(text, isControl) || strings.TrimSpace(text) != text {
		return true
	}
	if strings.ContainsRune("-?:,[]{}#&*!|>'\"%@`", rune(text[0])) {
		return true
	}
	if text == "..." || strings.HasPrefix(text, "... ") {
		return true
	}
	if strings.Contains(text, ": ") || strings.Contains(text, " #") || strings.HasSuffix(text, ":") {
		return true
	}
	if isAnyOf(text, "null", "Null", "NULL", "~", "true", "True", "TRUE", "false", "False", "FALSE") {
		return true
	}
	if coreSchemaNumber.MatchString(text) {
		return true
	}
	if _, ok := parseJavaLong(text); ok {
		return true
	}
	_, ok := parseJavaDouble(text)
	return ok
}

// coreSchemaNumber matches the YAML 1.2 core schema int and float forms, which also cover the JSON schema.
var coreSchemaNumber = regexp.MustCompile(`^(?:[-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+|` +
	`[-+]?(?:\.[0-9]+|[0-9]+(?:\.[0-9]*)?)(?:[eE][-+]?[0-9]+)?|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)
