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
	return e.render(visited, strconv.Quote, func(v BalValue, visited map[uintptr]bool) string {
		return toString(v, visited, false)
	})
}

// BalString returns the Ballerina expression-style representation of the
// error, so nested decimal/float/string detail values keep their
// expression-style forms (a "d" suffix, a "float:" prefix, valid Ballerina
// escaping) instead of String's informal ones.
func (e *Error) BalString(visited map[uintptr]bool) string {
	return e.render(visited, balStringLiteral, BalString)
}

func (e *Error) render(visited map[uintptr]bool, quote func(string) string, format func(BalValue, map[uintptr]bool) string) string {
	var b strings.Builder
	if e.TypeName != "" {
		b.WriteString("error ")
		b.WriteString(e.TypeName)
		b.WriteString(" (")
	} else {
		b.WriteString("error(")
	}
	b.WriteString(quote(e.Message))

	if e.Cause != nil {
		b.WriteByte(',')
		b.WriteString(format(e.Cause, visited))
	}
	if e.Detail != nil {
		for entry := e.Detail.head; entry != nil; entry = entry.next {
			b.WriteByte(',')
			b.WriteString(entry.key)
			b.WriteByte('=')
			b.WriteString(format(entry.value, visited))
		}
	}

	b.WriteByte(')')
	return b.String()
}
