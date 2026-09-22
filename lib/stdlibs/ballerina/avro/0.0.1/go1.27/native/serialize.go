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

// Schema-directed serialization of Ballerina values into goavro's native Go
// representation, mirroring jBallerina's SerializeVisitor.
//
// encodeValue only ever builds the Go value goavro.Codec.BinaryFromNative
// expects (map[string]any for record/map, []any for array, a wrapped
// map[string]any{branchName: value} for a non-null union member, and a Go
// type matching each primitive exactly); it never touches the wire format
// itself. The one call to BinaryFromNative in toAvroExtern does that, and its
// own type/membership/size checks are a backstop that should not normally
// fire — encodeValue has already done the semantic work.

package native

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// encodeValue converts data to goavro's native representation for s. The
// schema drives the walk; each leaf decides for itself which Ballerina values
// it accepts, and how far it will coerce them, so that the result matches
// what jBallerina's Avro writer produces for the same value. tc backs the
// semtype checks encodeBytes and encodeUnion's branch selection need to tell
// a byte[] apart from any other list, and a record apart from a plain map.
func encodeValue(tc semtypes.Context, s *shape, data values.BalValue) (any, error) {
	switch s.kind {
	case shapeNull:
		return encodeNull(data)
	case shapeBoolean:
		return encodeBoolean(data)
	case shapeInt:
		return encodeInt(data)
	case shapeLong:
		return encodeLong(data)
	case shapeFloat:
		return encodeFloat(data)
	case shapeDouble:
		return encodeDouble(data)
	case shapeString:
		return encodeString(data)
	case shapeBytes:
		return encodeBytes(tc, data)
	case shapeFixed:
		return encodeBytes(tc, data)
	case shapeEnum:
		return encodeEnum(data)
	case shapeRecord:
		return encodeRecord(tc, s, data)
	case shapeMap:
		return encodeMap(tc, s, data)
	case shapeArray:
		return encodeArray(tc, s, data)
	case shapeUnion:
		return encodeUnion(tc, s, data)
	default:
		return nil, fmt.Errorf("unsupported Avro schema kind: %v", s.kind)
	}
}

// The message matches jBallerina's NullSerializer.
func encodeNull(data values.BalValue) (any, error) {
	if data != nil {
		return nil, fmt.Errorf("the value does not match with the null schema")
	}
	return nil, nil
}

func encodeBoolean(data values.BalValue) (any, error) {
	value, ok := data.(bool)
	if !ok {
		return nil, typeMismatch("boolean", data)
	}
	return value, nil
}

// Ballerina int is 64 bits wide and the Avro int is 32, so the value is
// narrowed with the same wrapping `Long.intValue()` conversion jBallerina
// applies. Truncating here — rather than handing goavro the raw int64 — is
// what preserves the wrap: goavro's own int encoder rejects a value that
// would lose precision instead of wrapping it. A float is not accepted:
// nothing in jBallerina converts one here, it only reaches Avro's `Number`
// cast by accident.
func encodeInt(data values.BalValue) (any, error) {
	value, ok := data.(int64)
	if !ok {
		return nil, typeMismatch("int", data)
	}
	return int32(value), nil
}

func encodeLong(data values.BalValue) (any, error) {
	value, ok := data.(int64)
	if !ok {
		return nil, typeMismatch("long", data)
	}
	return value, nil
}

// An int widens into either floating-point schema, as jBallerina's DOUBLE
// branch does explicitly.
func encodeFloat(data values.BalValue) (any, error) {
	switch value := data.(type) {
	case float64:
		return float32(value), nil
	case int64:
		return float32(value), nil
	default:
		return nil, typeMismatch("float", data)
	}
}

// decimal is accepted here — and only here — because jBallerina's DOUBLE branch
// converts it before the writer sees it.
func encodeDouble(data values.BalValue) (any, error) {
	switch value := data.(type) {
	case float64:
		return value, nil
	case int64:
		return float64(value), nil
	case *decimal.Decimal:
		return value.Float64(), nil
	default:
		return nil, typeMismatch("double", data)
	}
}

// Any value is stringified, matching jBallerina's `data.toString()`. Nil is the
// one exception — it throws there, so it is an error here too.
func encodeString(data values.BalValue) (any, error) {
	if data == nil {
		return nil, typeMismatch("string", data)
	}
	if value, ok := data.(string); ok {
		return value, nil
	}
	return stringOf(data), nil
}

// stringOf is values.String with the cycle-tracking map it needs: passing nil
// panics as soon as the value is a list or a mapping.
func stringOf(data values.BalValue) string {
	return values.String(data, map[uintptr]bool{})
}

// encodeBytes also serves fixed: both take a byte[] as-is and leave fixed's
// exact-length check to goavro, which already implements and tests it.
// isByteList validates against the value's own type rather than its
// elements: an int[] is rejected even when every element happens to fit in
// a byte, the same way jBallerina rejects it (though jBallerina panics with
// an uncaught NullPointerException instead of reporting a clean error).
func encodeBytes(tc semtypes.Context, data values.BalValue) (any, error) {
	value, ok := data.(*values.List)
	if !ok || !isByteList(tc, value) {
		return nil, typeMismatch("bytes", data)
	}
	return value.ToByteSlice(), nil
}

func encodeEnum(data values.BalValue) (any, error) {
	value, ok := data.(string)
	if !ok {
		return nil, typeMismatch("enum", data)
	}
	return value, nil
}

// Fields are converted in schema order and every field key is always set —
// even to a Go nil when the Ballerina value has no such key — so a missing
// field always reaches the field's own type check rather than silently
// picking up the schema's default value. jBallerina's RecordSerializer never
// applies write-side defaults either.
func encodeRecord(tc semtypes.Context, s *shape, data values.BalValue) (any, error) {
	record, ok := data.(*values.Map)
	if !ok {
		return nil, typeMismatch("record", data)
	}
	fields := make(map[string]any, len(s.fields))
	for _, field := range s.fields {
		value, _ := record.Get(field.name)
		encoded, err := encodeValue(tc, field.shape, value)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", field.name, err)
		}
		fields[field.name] = encoded
	}
	return fields, nil
}

func encodeMap(tc semtypes.Context, s *shape, data values.BalValue) (any, error) {
	entries, ok := data.(*values.Map)
	if !ok {
		return nil, typeMismatch("map", data)
	}
	result := make(map[string]any, entries.Len())
	for _, key := range entries.Keys() {
		value, _ := entries.Get(key)
		encoded, err := encodeValue(tc, s.value, value)
		if err != nil {
			return nil, fmt.Errorf("key '%s': %w", key, err)
		}
		result[key] = encoded
	}
	return result, nil
}

func encodeArray(tc semtypes.Context, s *shape, data values.BalValue) (any, error) {
	items, ok := data.(*values.List)
	if !ok {
		return nil, typeMismatch("array", data)
	}
	result := make([]any, items.Len())
	for i := range items.Len() {
		encoded, err := encodeValue(tc, s.item, items.Get(i))
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		result[i] = encoded
	}
	return result, nil
}

// Branch selection runs in two passes: first the branch whose Avro type is the
// natural encoding of the value, then — only if nothing matched — the wider
// claims of jBallerina's tag table. Within each pass the union's declared order
// decides, so selection stays order-sensitive as in jBallerina.
//
// jBallerina applies the tag table alone, and only for unions inside a record
// field; a top-level union skips its UnionSerializer entirely and falls through
// to Avro's own resolveUnion, which rejects strings, byte arrays, records,
// enums, arrays and fixed values. Running one rule at every position is what
// makes a top-level `["null","string"]` work here.
//
// A chosen null branch encodes to a bare Go nil — goavro's union encoder reads
// that as the null-branch signal — while every other branch is wrapped in a
// single-entry map keyed by its exact goavro type name.
func encodeUnion(tc semtypes.Context, s *shape, data values.BalValue) (any, error) {
	amb := branchAmbiguity(s.branches)
	for _, claims := range []func(semtypes.Context, *shape, values.BalValue, ambiguity) bool{naturalBranch, widenedBranch} {
		for _, branch := range s.branches {
			if !claims(tc, branch, data, amb) {
				continue
			}
			if branch.kind == shapeNull {
				return nil, nil
			}
			encoded, err := encodeValue(tc, branch, data)
			if err != nil {
				return nil, err
			}
			return map[string]any{branch.fullName: encoded}, nil
		}
	}
	return nil, fmt.Errorf("value does not match with the Avro union types")
}

// ambiguity records which container-level clashes s's branches actually
// contain, so naturalBranch only pays for a semtype-based disambiguation
// where a clash is possible.
type ambiguity struct {
	// bytesVsArray is set when the union has both a bytes/fixed branch and
	// an array branch — Avro permits this since they're different kinds,
	// but both store a Ballerina value as *values.List.
	bytesVsArray bool
	// recordVsMap is set when the union has both a record branch and a map
	// branch — legal for the same reason, and both store a Ballerina value
	// as *values.Map.
	recordVsMap bool
}

func branchAmbiguity(branches []*shape) ambiguity {
	var hasBytesLike, hasArray, hasRecord, hasMap bool
	for _, branch := range branches {
		switch branch.kind {
		case shapeBytes, shapeFixed:
			hasBytesLike = true
		case shapeArray:
			hasArray = true
		case shapeRecord:
			hasRecord = true
		case shapeMap:
			hasMap = true
		}
	}
	return ambiguity{
		bytesVsArray: hasBytesLike && hasArray,
		recordVsMap:  hasRecord && hasMap,
	}
}

// naturalBranch reports whether branch is the Avro type a Ballerina value
// encodes to without conversion, so a lossless branch always wins over a
// widening one.
//
// bytes/fixed and array both store a Ballerina value as *values.List, and
// record and map both store one as *values.Map — the Go container alone
// can't tell them apart. When a union actually combines one of each (legal
// in Avro, since they're different kinds), the amb flags it carries force a
// look at the value's own inherent type — isByteList/isRecordLike — the
// same signal jBallerina's runtime type tag uses to pick the same branch
// regardless of declaration order. Outside that clash the plain container
// check is kept: a bare `anydata` argument built from a mapping-constructor
// literal (`schema.toAvro({x: 1})`, with no schema-derived expected type to
// resolve against) always has the generic, field-less map<anydata> inherent
// type in this interpreter — same as in jBallerina — so requiring
// isRecordLike unconditionally would reject it even against a schema with
// no competing map branch to confuse it with.
func naturalBranch(tc semtypes.Context, branch *shape, data values.BalValue, amb ambiguity) bool {
	switch branch.kind {
	case shapeNull:
		return data == nil
	case shapeBoolean:
		_, ok := data.(bool)
		return ok
	case shapeInt, shapeLong:
		_, ok := data.(int64)
		return ok
	case shapeFloat, shapeDouble:
		_, ok := data.(float64)
		return ok
	case shapeString, shapeEnum:
		_, ok := data.(string)
		return ok
	case shapeBytes, shapeFixed:
		list, ok := data.(*values.List)
		return ok && (!amb.bytesVsArray || isByteList(tc, list))
	case shapeArray:
		list, ok := data.(*values.List)
		return ok && (!amb.bytesVsArray || !isByteList(tc, list))
	case shapeRecord:
		m, ok := data.(*values.Map)
		return ok && (!amb.recordVsMap || isRecordLike(tc, m))
	case shapeMap:
		m, ok := data.(*values.Map)
		return ok && (!amb.recordVsMap || !isRecordLike(tc, m))
	}
	return false
}

// isByteList reports whether every member list.Type admits — its fixed
// slots and its rest — is a subtype of byte, i.e. whether list is genuinely
// a byte[] rather than some other list sharing the same Go representation.
func isByteList(tc semtypes.Context, list *values.List) bool {
	atomic := semtypes.ToListAtomicType(tc.Env(), list.Type)
	if atomic == nil {
		return false
	}
	for i := range atomic.FixedLength() {
		if !semtypes.IsSubtype(tc, atomic.MemberAt(i), semtypes.Byte) {
			return false
		}
	}
	return semtypes.IsSubtype(tc, atomic.Rest(), semtypes.Byte)
}

// isRecordLike reports whether m's inherent type carries any declared field
// names. A plain map<T> has none — every member type is reached through the
// mapping's open "rest" type — while a record (even one flowing through as
// anydata) always has its fields named individually.
func isRecordLike(tc semtypes.Context, m *values.Map) bool {
	atomic := semtypes.ToMappingAtomicType(tc, m.Type)
	return atomic != nil && len(atomic.FieldNames()) > 0
}

// widenedBranch follows jBallerina's SerializeVisitor.deriveBallerinaTag, where
// the floating-point branches also claim values they can only encode by
// converting. It claims exactly what the matching encoder accepts, so a branch
// chosen here always encodes; jBallerina's table is wider than its encoders and
// can pick a branch that then fails.
func widenedBranch(_ semtypes.Context, branch *shape, data values.BalValue, _ ambiguity) bool {
	switch branch.kind {
	case shapeFloat:
		_, ok := data.(int64)
		return ok
	case shapeDouble:
		switch data.(type) {
		case int64, *decimal.Decimal:
			return true
		}
	}
	return false
}

func typeMismatch(expected string, data values.BalValue) error {
	return fmt.Errorf("expected a value matching the '%s' schema, found '%s'",
		expected, stringOf(data))
}
