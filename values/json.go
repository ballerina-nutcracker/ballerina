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

package values

import (
	"encoding/json"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/semtypes"
)

// ToJSONByteArray serializes a Ballerina JSON value to its JSON byte
// representation. It is shared by stdlib I/O paths that write JSON to the wire
// or to files (e.g. io:fileWriteJson, http request/response payloads). Decimals
// are serialized in their exact string form so precision is preserved.
func ToJSONByteArray(v BalValue) ([]byte, error) {
	return json.Marshal(balToGoJSON(v))
}

// balToGoJSON converts a Ballerina JSON value to a Go value suitable for
// json.Marshal. Decimals are emitted as their exact string form so marshalling
// preserves precision. Values outside the json type (and unrecognised types)
// map to nil.
func balToGoJSON(v BalValue) any {
	switch t := v.(type) {
	case nil:
		return nil
	case bool:
		return t
	case int64:
		return t
	case float64:
		return t
	case *decimal.Decimal:
		return json.RawMessage(t.String())
	case string:
		return t
	case *Map:
		m := make(map[string]any, t.Len())
		for _, k := range t.Keys() {
			val, _ := t.Get(k)
			m[k] = balToGoJSON(val)
		}
		return m
	case *List:
		s := make([]any, t.Len())
		for i := range t.Len() {
			s[i] = balToGoJSON(t.Get(i))
		}
		return s
	default:
		return nil
	}
}

// GoToBalValue converts a Go value decoded from JSON into a Ballerina value.
// The decoder must be configured with UseNumber so numeric values arrive as
// json.Number; integers that fit in int64 become int, otherwise float.
// jsonListTy and jsonMapTy are the list/mapping types used for decoded arrays
// and objects.
func GoToBalValue(tc semtypes.Context, v any, jsonListTy, jsonMapTy semtypes.SemType) BalValue {
	switch v := v.(type) {
	case nil:
		return nil
	case bool:
		return v
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		f, _ := v.Float64()
		return f
	case string:
		return v
	case []any:
		items := make([]BalValue, len(v))
		for i, elem := range v {
			items[i] = GoToBalValue(tc, elem, jsonListTy, jsonMapTy)
		}
		return NewList(jsonListTy, semtypes.ToListAtomicType(tc, jsonListTy), false, nil, 0, items)
	case map[string]any:
		m := NewMap(jsonMapTy, semtypes.ToMappingAtomicType(tc, jsonMapTy), false, nil)
		for k, val := range v {
			m.Put(tc, k, GoToBalValue(tc, val, jsonListTy, jsonMapTy))
		}
		return m
	default:
		return nil
	}
}
