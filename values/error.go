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

package values

import (
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"strconv"
	"strings"
	"unicode"
)

// Error represents a Ballerina error value at runtime.
type Error struct {
	Type     semtypes.SemType
	Message  string
	Cause    BalValue
	Detail   *Map
	TypeName string
}

func NewError(t semtypes.SemType, message string, cause BalValue, typeName string, detail *Map) *Error {
	if detail == nil {
		detail = NewMap(semtypes.Mapping, &semtypes.MappingAtomicInner, true, nil)
	}
	return &Error{
		Type:     t,
		Message:  message,
		Cause:    cause,
		Detail:   detail,
		TypeName: typeName,
	}
}

func NewErrorWithMessage(message string) *Error {
	return NewError(semtypes.Error, message, nil, "", nil)
}

// String returns the Ballerina string representation of the error.
func (e *Error) String(visited map[uintptr]bool) string {
	var b strings.Builder
	writeTypeName(&b, e.TypeName)
	b.WriteString(strconv.Quote(e.Message))
	if e.Cause != nil {
		b.WriteByte(',')
		b.WriteString(toString(e.Cause, visited, false))
	}
	writeDetail(&b, e.Detail, visited, func(v BalValue, visited map[uintptr]bool) string {
		return toString(v, visited, false)
	})
	b.WriteByte(')')
	return b.String()
}

func (e *Error) BalString(visited map[uintptr]bool) string {
	var b strings.Builder
	writeTypeName(&b, e.TypeName)
	b.WriteString(balStringLiteral(e.Message))
	if e.Cause != nil {
		b.WriteByte(',')
		b.WriteString(BalString(e.Cause, visited))
	}
	writeDetail(&b, e.Detail, visited, BalString)
	b.WriteByte(')')
	return b.String()
}

func writeTypeName(b *strings.Builder, typeName string) {
	if typeName != "" {
		b.WriteString("error ")
		b.WriteString(typeName)
		b.WriteString(" (")
	} else {
		b.WriteString("error(")
	}
}

func writeDetail(b *strings.Builder, detail *Map, visited map[uintptr]bool, format func(BalValue, map[uintptr]bool) string) {
	if detail == nil {
		return
	}
	for entry := detail.head; entry != nil; entry = entry.next {
		b.WriteByte(',')
		b.WriteString(balDetailKey(entry.key))
		b.WriteByte('=')
		b.WriteString(format(entry.value, visited))
	}
}

// balDetailKey renders a detail-map key as a Ballerina identifier, since
// it's written bare before "=" in error(...) syntax. A key that isn't
// already a valid identifier (e.g. from a quoted-identifier named arg) is
// rendered as a quoted one instead, with non-identifier runes escaped.
func balDetailKey(key string) string {
	if isBareIdentifier(key) {
		return key
	}
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range key {
		if isIdentifierRune(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('\\')
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isBareIdentifier(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if i == 0 {
			if !isIdentifierInitialRune(r) {
				return false
			}
			continue
		}
		if !isIdentifierRune(r) {
			return false
		}
	}
	return true
}

func isIdentifierInitialRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentifierRune(r rune) bool {
	return isIdentifierInitialRune(r) || unicode.IsDigit(r)
}
