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
	stdruntime "runtime"
	"sync"
	"weak"

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

type jsonTypePair struct {
	listTy semtypes.SemType
	mapTy  semtypes.SemType
}

// jsonTypesByEnv associates a weak pointer to a semtypes.Env with the canonical
// json[]/map<json> semtypes built for it, self-cleaning once the Env is unreachable.
var jsonTypesByEnv sync.Map // weak.Pointer[env-pointee] -> jsonTypePair

// jsonTypesInitMu serializes the cache-miss (slow) path of JSONListAndMapTypes.
// Without it, two goroutines racing on the same env could both miss the
// cache and each register its own atom into the shared environment -
// precisely the atom-table-shift hazard this cache exists to prevent.
var jsonTypesInitMu sync.Mutex

// JSONListAndMapTypes returns the canonical json[]/map<json> semtypes for a context's
// environment, memoized per environment. semtypes.ContextFrom builds a fresh Context
// (with empty memo maps) on every call, so semtypes.CreateJSON's own per-Context memo
// does not stop two independent callers (e.g. two stdlibs) building separate
// ListDefinition/MappingDefinition instances for "the same" json list/map type — each
// registers its own atom into the shared environment, which is otherwise-harmless but
// shifts how unrelated recursive types print in that environment (extra atoms shift
// atom-table numbering). Every caller that needs these types for GoToBalValue must go
// through this shared accessor instead of building its own.
func JSONListAndMapTypes(ctx semtypes.Context) (semtypes.SemType, semtypes.SemType) {
	env := ctx.Env()
	// Boxed as `any` immediately: env's pointee type is unexported in semtypes, so this
	// package can only name weak.Pointer[...] for it via type inference, not explicitly —
	// boxing lets the AddCleanup callback below stay a plain func(any).
	key := any(weak.Make(env))
	if v, ok := jsonTypesByEnv.Load(key); ok {
		p := v.(jsonTypePair)
		return p.listTy, p.mapTy
	}
	jsonTypesInitMu.Lock()
	defer jsonTypesInitMu.Unlock()
	if v, ok := jsonTypesByEnv.Load(key); ok {
		p := v.(jsonTypePair)
		return p.listTy, p.mapTy
	}
	jsonTy := semtypes.CreateJSON(ctx)
	listLd := semtypes.NewListDefinition()
	mapMd := semtypes.NewMappingDefinition()
	listTy := listLd.DefineListTypeWrappedWithEnvSemType(env, jsonTy)
	mapTy := mapMd.DefineMappingTypeWrapped(env, nil, jsonTy)
	p := jsonTypePair{listTy, mapTy}
	jsonTypesByEnv.Store(key, p)
	stdruntime.AddCleanup(env, cleanupJSONTypes, key)
	return listTy, mapTy
}

func cleanupJSONTypes(key any) {
	jsonTypesByEnv.Delete(key)
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
