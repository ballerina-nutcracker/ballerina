// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// parseJSON decodes a single JSON document into a Ballerina json value, keeping object
// member order (the shared values.GoToBalValue goes through a Go map and loses it).
func parseJSON(tc semtypes.Context, types *ftpTypes, data []byte) (values.BalValue, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeJSONValue(tc, types, dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("invalid JSON document: unexpected content after the JSON value")
	}
	return v, nil
}

func decodeJSONValue(tc semtypes.Context, types *ftpTypes, dec *json.Decoder) (values.BalValue, error) {
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("invalid JSON document: unexpected end of input")
		}
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '[' {
			var items []values.BalValue
			for dec.More() {
				item, err := decodeJSONValue(tc, types, dec)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return values.NewList(types.jsonListTy, types.jsonListAtom, false, nil, 0, items), nil
		}
		m := values.NewMap(types.jsonMapTy, types.jsonMapAtom, false, nil)
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			value, err := decodeJSONValue(tc, types, dec)
			if err != nil {
				return nil, err
			}
			m.Put(tc, keyTok.(string), value)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return m, nil
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i, nil
		}
		f, _ := t.Float64()
		return f, nil
	case string, bool, nil:
		return t, nil
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}

// toJSONString renders a value the way jBallerina's toJsonString does: members are separated
// by ", " and strings escape only quotes, backslashes and control characters.
func toJSONString(v values.BalValue) string {
	var b strings.Builder
	writeJSON(&b, v)
	return b.String()
}

func writeJSON(b *strings.Builder, v values.BalValue) {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case string:
		writeJSONString(b, t)
	case float64:
		b.WriteString(values.FormatFloat(t))
	case *values.List:
		b.WriteByte('[')
		for i := 0; i < t.Len(); i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			writeJSON(b, t.Get(i))
		}
		b.WriteByte(']')
	case *values.Map:
		b.WriteByte('{')
		for i, key := range t.Keys() {
			if i > 0 {
				b.WriteString(", ")
			}
			writeJSONString(b, key)
			b.WriteByte(':')
			value, _ := t.Get(key)
			writeJSON(b, value)
		}
		b.WriteByte('}')
	default:
		b.WriteString(values.String(t, nil))
	}
}

func writeJSONString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				b.WriteString(`\u` + fmt.Sprintf("%04x", r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

func contentBindingError(message string, cause error, filePath string, content []byte) *ftpError {
	return &ftpError{kind: kindContentBinding, message: message, cause: cause, filePath: &filePath, content: content}
}

// bindJSON parses JSON content and binds it to target. Binding uses value:cloneWithType
// semantics, standing in for data.jsondata which is not available yet.
func bindJSON(tc semtypes.Context, types *ftpTypes, data []byte, target semtypes.SemType, filePath string, lax bool) (values.BalValue, error) {
	value, err := parseJSON(tc, types, data)
	if err != nil {
		return nil, contentBindingError(err.Error(), nil, filePath, data)
	}
	if semtypes.IsSubtype(tc, types.jsonTy, target) {
		return value, nil
	}
	if lax {
		value = relaxForBinding(tc, types, value, target)
	}
	bound, cloneErr := values.CloneWithType(tc, value, target)
	if cloneErr != nil {
		return nil, contentBindingError(cloneErr.Message, nil, filePath, data)
	}
	return bound, nil
}

func bindXML(tc semtypes.Context, data []byte, target semtypes.SemType, filePath string) (values.BalValue, error) {
	if !semtypes.IsSubtype(tc, semtypes.XML, target) && semtypes.IsNever(semtypes.Intersect(semtypes.XML, target)) {
		return nil, newError(kindError, "Binding XML content to a record type is not yet supported")
	}
	value, err := values.ParseAsXMLValue(tc, values.FromBytes(data), values.XMLLenientMode)
	if err != nil {
		return nil, contentBindingError(err.Error(), nil, filePath, data)
	}
	return value, nil
}

// csvCell mirrors CSVUtils.escapeCsvValue: a cell is quoted only when it contains a comma, a
// quote or a line break, and embedded quotes are doubled.
func csvCell(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func csvValue(v values.BalValue) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	}
	return values.String(v, nil)
}

func csvListRow(row *values.List) string {
	cells := make([]string, row.Len())
	for i := range cells {
		cells[i] = csvCell(csvValue(row.Get(i)))
	}
	return strings.Join(cells, ",")
}

func csvHeaderRow(keys []string) string {
	cells := make([]string, len(keys))
	for i, k := range keys {
		cells[i] = csvCell(k)
	}
	return strings.Join(cells, ",")
}

func csvRecordRow(record *values.Map, keys []string) string {
	cells := make([]string, len(keys))
	for i, k := range keys {
		v, _ := record.Get(k)
		cells[i] = csvCell(csvValue(v))
	}
	return strings.Join(cells, ",")
}

// csvContent mirrors CSVUtils.convertToCsv: a record array takes its columns from the first
// record, and every row (the last included) ends with the platform line separator.
func csvContent(content *values.List, addHeader bool) string {
	if content.Len() == 0 {
		return ""
	}
	var b strings.Builder
	if _, isList := content.Get(0).(*values.List); isList {
		for i := 0; i < content.Len(); i++ {
			b.WriteString(csvListRow(content.Get(i).(*values.List)))
			b.WriteString(lineSeparator)
		}
		return b.String()
	}
	keys := content.Get(0).(*values.Map).Keys()
	if addHeader {
		b.WriteString(csvHeaderRow(keys))
		b.WriteString(lineSeparator)
	}
	for i := 0; i < content.Len(); i++ {
		b.WriteString(csvRecordRow(content.Get(i).(*values.Map), keys))
		b.WriteString(lineSeparator)
	}
	return b.String()
}
