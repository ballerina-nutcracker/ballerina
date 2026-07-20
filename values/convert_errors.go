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
	"fmt"
	"strings"

	"ballerina-lang-go/semtypes"
)

const conversionErrorTypeName = "{ballerina/lang.value}ConversionError"

// conversionFailure is either a leaf (detailMessage set, no children) describing a single
// conversion failure, or a union group (children set, detailMessage empty) describing why
// each member of a union target was rejected. Rendering happens once, in Error, so nested
// groups get correct depth-based indentation instead of being pre-rendered and re-embedded.
type conversionFailure struct {
	detailMessage string
	children      []*conversionFailure
}

func wrapConversionError(err *conversionFailure) *Error {
	message := err.Error()
	detailMap := NewMap(semtypes.MAPPING, &semtypes.MAPPING_ATOMIC_INNER, true, []MapEntry{
		{Key: "message", Value: message},
	})
	return NewError(semtypes.ERROR, message, nil, conversionErrorTypeName, detailMap)
}

func incompatibleConversion(tc semtypes.Context, value BalValue, targetType semtypes.SemType) *conversionFailure {
	sourceTy := SemTypeForValue(value)
	return newConversionFailure(fmt.Sprintf("'%s' value cannot be converted to '%s'",
		semtypes.ToString(tc, sourceTy), semtypes.ToString(tc, targetType)))
}

func cannotConvertNil(tc semtypes.Context, targetType semtypes.SemType) *conversionFailure {
	return newConversionFailure(fmt.Sprintf("'()' value cannot be converted to '%s'", semtypes.ToString(tc, targetType)))
}

// missingRequiredField reports a required field absent from the source value. Note this
// fires whether or not the field declares a default in `t`: default-value injection is not
// yet implemented, so a declared default does not make the field any less required here.
func missingRequiredField(tc semtypes.Context, value BalValue, targetType semtypes.SemType, fieldName string) *conversionFailure {
	sourceTy := SemTypeForValue(value)
	return newConversionFailure(fmt.Sprintf(
		"'%s' value cannot be converted to '%s': field '%s' not present in value, and default values are not supported",
		semtypes.ToString(tc, sourceTy), semtypes.ToString(tc, targetType), fieldName))
}

func (e *conversionFailure) Error() string {
	if len(e.children) == 0 {
		return e.detailMessage
	}
	var b strings.Builder
	b.WriteString("\n\t\t")
	e.render(&b, 0)
	return b.String()
}

func newConversionFailure(message string) *conversionFailure {
	return &conversionFailure{detailMessage: message}
}

// newUnionConversionFailure reports that every member of a union target was rejected, one
// failure per member (in try order). A member failure that is itself a union group renders
// as a nested, correctly-indented block instead of a pre-rendered string.
func newUnionConversionFailure(children []*conversionFailure) *conversionFailure {
	return &conversionFailure{children: children}
}

// render writes this failure at the given nesting depth, assuming the caller has already
// positioned the cursor at the start of a line; it emits no leading line break itself so
// callers control the break between siblings.
func (e *conversionFailure) render(b *strings.Builder, tabs int) {
	if len(e.children) == 0 {
		b.WriteString(e.detailMessage)
		return
	}
	indent := strings.Repeat("  ", tabs)
	b.WriteByte('{')
	for i, child := range e.children {
		b.WriteString("\n\t\t")
		b.WriteString(indent)
		b.WriteString("  ")
		child.render(b, tabs+1)
		if i < len(e.children)-1 {
			b.WriteString("\n\t\t")
			b.WriteString(indent)
			b.WriteString("  or")
		}
	}
	b.WriteString("\n\t\t")
	b.WriteString(indent)
	b.WriteByte('}')
}
