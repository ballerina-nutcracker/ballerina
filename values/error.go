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
	"ballerina/semtypes"
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
		detail = NewMap(semtypes.MAPPING, &semtypes.MAPPING_ATOMIC_INNER, true, nil)
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
	return NewError(semtypes.ERROR, message, nil, "", nil)
}

// String returns the Ballerina string representation of the error.
func (e *Error) String(visited map[uintptr]bool) string {
	var b strings.Builder
	if e.TypeName != "" {
		b.WriteString("error ")
		b.WriteString(e.TypeName)
		b.WriteString(" (")
	} else {
		b.WriteString("error(")
	}
	b.WriteString(strconv.Quote(e.Message))

	if e.Cause != nil {
		b.WriteByte(',')
		b.WriteString(toString(e.Cause, visited, false))
	}
	if e.Detail != nil {
		for entry := e.Detail.head; entry != nil; entry = entry.next {
			b.WriteByte(',')
			b.WriteString(entry.key)
			b.WriteByte('=')
			b.WriteString(toString(entry.value, visited, false))
		}
	}

	b.WriteByte(')')
	return b.String()
}
