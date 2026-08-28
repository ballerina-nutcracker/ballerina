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

// Package values has value representations use in runtime.
package values

import (
	"strconv"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Currently this is just an alias on any but I think we will need to add methods to this like type
type BalValue any

type Function struct {
	Type      semtypes.SemType
	LookupKey string
	// TODO: fix type here and remove unwanted casts
	ParentFrame any // *exec.Frame at runtime, nil for non-closures
}

// TypeDesc is the runtime representation of a typedesc value — a thin wrapper
// around a semtype.
type TypeDesc struct {
	Type        semtypes.SemType
	Annotations AnnotationValues
}

// NewTypeDesc returns a fully initialized TypeDesc.
func NewTypeDesc(ty semtypes.SemType, annotations AnnotationValues) *TypeDesc {
	if annotations == nil {
		annotations = NewAnnotationValues()
	}
	return &TypeDesc{
		Type:        ty,
		Annotations: annotations,
	}
}

// FillerFactory produces a fresh filler value each time it is invoked.
// Mutable filler values (lists, mappings) must be unique per slot, so they
// cannot be cached as a single shared value.
type FillerFactory func() BalValue

// FillerValue returns the runtime value representation for filler value according to https://ballerina.io/spec/lang/master/#FillMember
func FillerValue(cx semtypes.Context, t semtypes.SemType) (BalValue, bool) {
	factory, ok := FillerFactoryFor(cx, t)
	if !ok {
		return nil, false
	}
	return factory(), true
}

// FillerFactoryFor returns a factory that produces a fresh filler value for
// the given type on each invocation.
func FillerFactoryFor(cx semtypes.Context, t semtypes.SemType) (FillerFactory, bool) {
	filler, ok := semtypes.FillerValue(cx, t)
	if !ok {
		return nil, false
	}
	return fillerFactoryFromDesc(cx, filler)
}

func fillerFactoryFromDesc(cx semtypes.Context, f semtypes.Filler) (FillerFactory, bool) {
	switch f := f.(type) {
	case semtypes.SingleValueFiller:
		v := f.Value
		return func() BalValue { return v }, true
	case semtypes.MappingFiller:
		ty := f.Type
		atomic := f.Atomic
		readonly := semtypes.IsSubtype(cx, ty, semtypes.ValReadonly)
		return func() BalValue { return NewMap(ty, atomic, readonly, nil) }, true
	case semtypes.ListFiller:
		return listFillerFactory(cx, f)
	case semtypes.XMLFiller:
		return func() BalValue { return &XMLText{} }, true
	case semtypes.ObjectFiller, semtypes.StreamFiller, semtypes.TableFiller:
		return nil, false
	default:
		panic("unknown filler kind")
	}
}

func listFillerFactory(cx semtypes.Context, f semtypes.ListFiller) (FillerFactory, bool) {
	memberFactories := make([]FillerFactory, len(f.Members))
	for i, m := range f.Members {
		var ok bool
		memberFactories[i], ok = fillerFactoryFromDesc(cx, m)
		if !ok {
			return nil, false
		}
	}
	ty := f.Type
	atomic := f.Atomic
	readonly := semtypes.IsSubtype(cx, ty, semtypes.ValReadonly)
	restType := f.Atomic.Rest()
	// Resolve the rest filler factory lazily so that recursive types (e.g.
	// `type A A[]`) do not blow the stack while building the factory graph.
	var restFactory FillerFactory
	restResolved := false
	getRestFactory := func() FillerFactory {
		if !restResolved {
			restFactory, _ = FillerFactoryFor(cx, restType)
			restResolved = true
		}
		return restFactory
	}
	return func() BalValue {
		initial := make([]BalValue, len(memberFactories))
		for i, mf := range memberFactories {
			initial[i] = mf()
		}
		return NewList(ty, atomic, readonly, getRestFactory(), len(memberFactories), initial)
	}, true
}

func SemTypeForValue(v BalValue) semtypes.SemType {
	switch v := v.(type) {
	case nil:
		return semtypes.Nil
	case bool:
		return semtypes.BooleanConst(v)
	case int64:
		return semtypes.IntConst(v)
	case float64:
		return semtypes.FloatConst(v)
	case string:
		return semtypes.StringConst(v)
	case *decimal.Decimal:
		return semtypes.DecimalConst(*v)
	case *List:
		return v.Type
	case *Map:
		return v.Type
	case *Error:
		return v.Type
	case *Function:
		return v.Type
	case *Object:
		return v.Type
	case *Stream:
		return v.Type
	case *Future:
		return v.Type
	case XMLValue:
		return v.Type()
	case *TypeDesc:
		return semtypes.Typedesc
	default:
		return semtypes.Any
	}
}

func String(v BalValue, visited map[uintptr]bool) string {
	if v == nil {
		return ""
	}
	return toString(v, visited, true)
}

func toString(v BalValue, visited map[uintptr]bool, isDirect bool) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		if isDirect {
			return t
		}
		return strconv.Quote(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return FormatFloat(t)
	case bool:
		return strconv.FormatBool(t)
	case *decimal.Decimal:
		return t.FormatBallerina()
	case *List:
		return t.String(visited)
	case *Map:
		return t.String(visited)
	case *Error:
		return t.String(visited)
	case *Function:
		return "function " + t.LookupKey
	case *Object:
		return "object"
	case *Stream:
		return "stream"
	case *Future:
		return "future"
	case *TypeDesc:
		return "typedesc"
	case XMLValue:
		return t.XMLString()
	default:
		return "<unsupported>"
	}
}

// BalString renders v in Ballerina's expression style, as used by
// lang.value:toBalString: unlike String, strings are quoted at every level
// (including the top), nil renders as "()" rather than the empty string,
// decimals carry a trailing "d", and non-finite floats carry a "float:" prefix.
func BalString(v BalValue, visited map[uintptr]bool) string {
	switch t := v.(type) {
	case nil:
		return "()"
	case string:
		return strconv.Quote(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		formatted := FormatFloat(t)
		switch formatted {
		case "NaN", "Infinity", "-Infinity":
			return "float:" + formatted
		default:
			return formatted
		}
	case bool:
		return strconv.FormatBool(t)
	case *decimal.Decimal:
		return t.FormatBallerina() + "d"
	case *List:
		return t.BalString(visited)
	case *Map:
		return t.BalString(visited)
	// Error/object/function/stream/typedesc/xml expression-style rendering is
	// out of scope for the persist use case this backs; fall back to the
	// informal renderer rather than inventing an unverified format.
	case *Error:
		return t.String(visited)
	case *Function:
		return "function " + t.LookupKey
	case *Object:
		return "object"
	case *Stream:
		return "stream"
	case *TypeDesc:
		return "typedesc"
	case XMLValue:
		return t.XMLString()
	default:
		return "<unsupported>"
	}
}
