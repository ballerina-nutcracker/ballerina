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

// Converts goavro's decoded native Go value into a generic Ballerina value
// graph. Narrowing to the caller's target type happens afterwards in
// bindToTarget.
//
// fromAvroExtern calls goavro.Codec.NativeFromBinary exactly once to turn the
// wire bytes into a tree of map[string]any/[]any/scalars (already fully
// decoded — goavro owns the wire format end to end); decodeValue then walks
// that tree alongside the same shape tree encodeValue used, unwrapping each
// union to the value its selected branch carries and converting every
// primitive to its Ballerina representation.

package native

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// decodeValue converts native (goavro's decoded value for s) into a generic
// Ballerina value: map<anydata> for record/map, anydata[] for array, byte[]
// for bytes/fixed, and the target-appropriate scalar otherwise — the shapes
// CloneWithType can narrow.
func decodeValue(tc semtypes.Context, types *avroTypes, s *shape, native any) (values.BalValue, error) {
	switch s.kind {
	case shapeNull:
		return nil, nil
	case shapeBoolean:
		return decodeAs[bool](s, native)
	case shapeInt:
		value, err := decodeAs[int32](s, native)
		return int64(value), err
	case shapeLong:
		return decodeAs[int64](s, native)
	case shapeFloat:
		value, err := decodeAs[float32](s, native)
		return float64(value), err
	case shapeDouble:
		return decodeAs[float64](s, native)
	case shapeString, shapeEnum:
		return decodeAs[string](s, native)
	case shapeBytes, shapeFixed:
		raw, err := decodeAs[[]byte](s, native)
		if err != nil {
			return nil, err
		}
		return values.ByteSliceToList(types.byteArrTy, types.env, raw), nil
	case shapeRecord:
		return decodeRecord(tc, types, s, native)
	case shapeMap:
		return decodeMap(tc, types, s, native)
	case shapeArray:
		return decodeArray(tc, types, s, native)
	case shapeUnion:
		return decodeUnion(tc, types, s, native)
	default:
		return nil, fmt.Errorf("unsupported Avro schema kind: %v", s.kind)
	}
}

// decodeAs type-asserts native to T, or reports the mismatch as a decode
// error. goavro always hands back the Go type its own codec for the matching
// schema position produces, so a mismatch here means the shape tree and
// goavro's own parse of the same schema disagree.
func decodeAs[T any](s *shape, native any) (T, error) {
	value, ok := native.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("expected a decoded %T for schema kind %v, found %T", zero, s.kind, native)
	}
	return value, nil
}

func decodeRecord(tc semtypes.Context, types *avroTypes, s *shape, native any) (values.BalValue, error) {
	fields, ok := native.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a decoded record, found %T", native)
	}
	record := values.NewMap(types.anydataMapTy, types.anydataMapAt, false, nil)
	for _, field := range s.fields {
		value, err := decodeValue(tc, types, field.shape, fields[field.name])
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", field.name, err)
		}
		record.Put(tc, field.name, value)
	}
	return record, nil
}

func decodeMap(tc semtypes.Context, types *avroTypes, s *shape, native any) (values.BalValue, error) {
	entries, ok := native.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a decoded map, found %T", native)
	}
	result := values.NewMap(types.anydataMapTy, types.anydataMapAt, false, nil)
	for key, value := range entries {
		decoded, err := decodeValue(tc, types, s.value, value)
		if err != nil {
			return nil, fmt.Errorf("key '%s': %w", key, err)
		}
		result.Put(tc, key, decoded)
	}
	return result, nil
}

func decodeArray(tc semtypes.Context, types *avroTypes, s *shape, native any) (values.BalValue, error) {
	items, ok := native.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a decoded array, found %T", native)
	}
	decoded := make([]values.BalValue, len(items))
	for i, item := range items {
		value, err := decodeValue(tc, types, s.item, item)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		decoded[i] = value
	}
	return values.NewList(types.anydataListTy, types.anydataListAt, false, nil, 0, decoded), nil
}

// decodeUnion unwraps goavro's union representation: a bare nil for the null
// branch, or a single-entry map keyed by the selected branch's exact goavro
// type name (the same fullName encodeUnion used to build it).
func decodeUnion(tc semtypes.Context, types *avroTypes, s *shape, native any) (values.BalValue, error) {
	if native == nil {
		return nil, nil
	}
	wrapped, ok := native.(map[string]any)
	if !ok || len(wrapped) != 1 {
		return nil, fmt.Errorf("expected a decoded union, found %T", native)
	}
	for branchName, value := range wrapped {
		for _, branch := range s.branches {
			if branch.fullName == branchName {
				return decodeValue(tc, types, branch, value)
			}
		}
		return nil, fmt.Errorf("unknown union branch: %q", branchName)
	}
	return nil, nil // unreachable: len(wrapped) == 1
}
