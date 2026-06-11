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

// Package values provides the public API for Ballerina runtime values.
package values

import (
	"ballerina-lang-go/semtypes"
	cmp "ballerina-lang-go/values/compare"
	"ballerina-lang-go/values/convert"
	"ballerina-lang-go/values/core"
)

// Type aliases — preserves the existing API surface for all callers.
type (
	BalValue               = core.BalValue
	Function               = core.Function
	TypeDesc               = core.TypeDesc
	FillerFactory          = core.FillerFactory
	List                   = core.List
	Map                    = core.Map
	MapEntry               = core.MapEntry
	Object                 = core.Object
	ResourceEntry          = core.ResourceEntry
	ResourcePathSegmentDef = core.ResourcePathSegmentDef
	Stream                 = core.Stream
	Error                  = core.Error

	XMLValue                 = core.XMLValue
	XMLElement               = core.XMLElement
	XMLSequence              = core.XMLSequence
	XMLProcessingInstruction = core.XMLProcessingInstruction
	XMLText                  = core.XMLText
	XMLComment               = core.XMLComment

	CompareResult = cmp.CompareResult
)

// Constants re-exported from compare.
const (
	CmpLT CompareResult = cmp.CmpLT
	CmpEQ CompareResult = cmp.CmpEQ
	CmpGT CompareResult = cmp.CmpGT
	CmpUN CompareResult = cmp.CmpUN
)

// NeverValue is the sentinel for the Ballerina never type.
var NeverValue = core.NeverValue

// Constructors

func NewList(ty semtypes.SemType, atomic *semtypes.ListAtomicType, isReadonly bool, filler FillerFactory, size int, initial []BalValue) *List {
	return core.NewList(ty, atomic, isReadonly, filler, size, initial)
}

func NewMap(ty semtypes.SemType, atomic *semtypes.MappingAtomicType, isReadonly bool, entries []MapEntry) *Map {
	return core.NewMap(ty, atomic, isReadonly, entries)
}

// NewTypeDesc returns a fully initialized TypeDesc.
func NewTypeDesc(ty semtypes.SemType, annotations AnnotationValues) *TypeDesc {
	return core.NewTypeDesc(ty, annotations)
}

func NewObject(typ semtypes.SemType, fieldValues map[string]BalValue, methodKeys map[string]string, rtable map[string][]ResourceEntry) *Object {
	return core.NewObject(typ, fieldValues, methodKeys, rtable)
}

func NewStream(typ semtypes.SemType, next, close func() BalValue) *Stream {
	return core.NewStream(typ, next, close)
}

func NewError(t semtypes.SemType, message string, cause BalValue, typeName string, detail *Map) *Error {
	return core.NewError(t, message, cause, typeName, detail)
}

func NewErrorWithMessage(message string) *Error {
	return core.NewErrorWithMessage(message)
}

func NewXMLElement(name string, attrs, namespaces *Map, children XMLValue, isReadonly bool) *XMLElement {
	return core.NewXMLElement(name, attrs, namespaces, children, isReadonly)
}

func NewXMLProcessingInstruction(target, data string, isReadonly bool) *XMLProcessingInstruction {
	return core.NewXMLProcessingInstruction(target, data, isReadonly)
}

func NewXMLText(body string) *XMLText {
	return core.NewXMLText(body)
}

func NewXMLComment(body string, isReadonly bool) *XMLComment {
	return core.NewXMLComment(body, isReadonly)
}

func NewNormalizedXMLSequence(items []XMLValue) *XMLSequence {
	return core.NewNormalizedXMLSequence(items)
}

func NewXMLConcatSequence(items ...XMLValue) *XMLSequence {
	return core.NewXMLConcatSequence(items...)
}

func EscapeXMLContent(s string) string {
	return core.EscapeXMLContent(s)
}

func EscapeXMLAttribute(s string) string {
	return core.EscapeXMLAttribute(s)
}

// Value utilities

func SemTypeForValue(v BalValue) semtypes.SemType {
	return core.SemTypeForValue(v)
}

func String(v BalValue, visited map[uintptr]bool) string {
	return core.String(v, visited)
}

func FillerValue(cx semtypes.Context, t semtypes.SemType) (BalValue, bool) {
	return core.FillerValue(cx, t)
}

func FillerFactoryFor(cx semtypes.Context, t semtypes.SemType) (FillerFactory, bool) {
	return core.FillerFactoryFor(cx, t)
}

// Float utilities

func FloatExactEqual(a, b float64) bool {
	return core.FloatExactEqual(a, b)
}

// DeepEquals implements the Ballerina DeepEquals abstract operation.
func DeepEquals(v1, v2 BalValue) bool {
	return core.DeepEquals(v1, v2)
}

// Compare functions

func Compare(x, y BalValue) CompareResult {
	return cmp.Compare(x, y)
}

func CompareK(x, y BalValue, ascending bool) CompareResult {
	return cmp.CompareK(x, y, ascending)
}

// Convert converts a JSON value to the given target type.
// On failure it returns a lang.value ConversionError as *Error.
func Convert(tc semtypes.Context, value BalValue, targetType semtypes.SemType) (BalValue, *Error) {
	return convert.Convert(tc, value, targetType)
}
