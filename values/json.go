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
	"bytes"
	"encoding/json"
	"fmt"
	stdruntime "runtime"
	"sync"
	"weak"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/semtypes"
)

// ToJSONByteArray serializes a Ballerina JSON value to its JSON byte
// representation. It is shared by stdlib I/O paths that write JSON to the wire
// or to files (e.g. io:fileWriteJson, http request/response payloads). Decimals
// are serialized in their exact string form so precision is preserved. Map
// fields are written in the map's insertion order, matching Ballerina's
// map<json>/json object semantics (encoding/json's Marshal would otherwise
// alphabetize map keys).
func ToJSONByteArray(v BalValue) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeJSON appends the JSON encoding of a Ballerina JSON value to buf,
// preserving *Map insertion order. Values outside the json type (and
// unrecognised types) are written as null.
func writeJSON(buf *bytes.Buffer, v BalValue) error {
	switch t := v.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case int64:
		return writeJSONMarshaled(buf, t)
	case float64:
		return writeJSONMarshaled(buf, t)
	case *decimal.Decimal:
		buf.WriteString(t.String())
		return nil
	case string:
		return writeJSONMarshaled(buf, t)
	case *Map:
		buf.WriteByte('{')
		for i, k := range t.Keys() {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSONMarshaled(buf, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			val, _ := t.Get(k)
			if err := writeJSON(buf, val); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case *List:
		buf.WriteByte('[')
		for i := range t.Len() {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSON(buf, t.Get(i)); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	default:
		buf.WriteString("null")
		return nil
	}
}

func writeJSONMarshaled(buf *bytes.Buffer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	buf.Write(b)
	return nil
}

type jsonTypePair struct {
	listTy semtypes.SemType
	mapTy  semtypes.SemType
}

// jsonTypesByEnv associates a weak pointer to a semtypes.Env with the canonical
// json[]/map<json> semtypes built for it, self-cleaning once the Env is unreachable.
var jsonTypesByEnv sync.Map // weak.Pointer[env-pointee] -> jsonTypePair

// JSONListAndMapTypes returns the canonical json[]/map<json> semtypes for a context's
// environment, memoized per environment. semtypes.ContextFrom builds a fresh Context
// (with empty memo maps) on every call, so semtypes.CreateJSON's own per-Context memo
// does not stop two independent callers (e.g. two stdlibs) building separate
// ListDefinition/MappingDefinition instances for "the same" json list/map type — each
// registers its own atom into the shared environment, which is otherwise-harmless but
// shifts how unrelated recursive types print in that environment (extra atoms shift
// atom-table numbering). Every caller that needs these types for DecodeJSON must go
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

// DecodeJSON reads exactly one JSON value from dec and converts it to a
// Ballerina value. Object keys are inserted into the resulting *Map in the
// order they appear on the wire, matching Ballerina's map<json>/json object
// insertion-order semantics (decoding into a plain Go map first, as
// encoding/json's Decode(&any{}) does, would discard that order since Go map
// iteration is randomized). jsonListTy and jsonMapTy are the list/mapping
// types used for decoded arrays and objects.
func DecodeJSON(dec *json.Decoder, tc semtypes.Context, jsonListTy, jsonMapTy semtypes.SemType) (BalValue, error) {
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeJSONToken(tok, dec, tc, jsonListTy, jsonMapTy)
}

func decodeJSONValue(dec *json.Decoder, tc semtypes.Context, jsonListTy, jsonMapTy semtypes.SemType) (BalValue, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeJSONToken(tok, dec, tc, jsonListTy, jsonMapTy)
}

func decodeJSONToken(tok json.Token, dec *json.Decoder, tc semtypes.Context, jsonListTy, jsonMapTy semtypes.SemType) (BalValue, error) {
	switch t := tok.(type) {
	case nil:
		return nil, nil
	case bool:
		return t, nil
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i, nil
		}
		f, err := t.Float64()
		if err != nil {
			return nil, err
		}
		return f, nil
	case string:
		return t, nil
	case json.Delim:
		switch t {
		case '[':
			items := make([]BalValue, 0)
			for dec.More() {
				item, err := decodeJSONValue(dec, tc, jsonListTy, jsonMapTy)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
			if _, err := dec.Token(); err != nil { // consume closing ']'
				return nil, err
			}
			return NewList(jsonListTy, semtypes.ToListAtomicType(tc, jsonListTy), false, nil, 0, items), nil
		case '{':
			m := NewMap(jsonMapTy, semtypes.ToMappingAtomicType(tc, jsonMapTy), false, nil)
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("expected JSON object key, got %v", keyTok)
				}
				val, err := decodeJSONValue(dec, tc, jsonListTy, jsonMapTy)
				if err != nil {
					return nil, err
				}
				m.Put(tc, key, val)
			}
			if _, err := dec.Token(); err != nil { // consume closing '}'
				return nil, err
			}
			return m, nil
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter: %v", t)
		}
	default:
		return nil, fmt.Errorf("unexpected JSON token: %v", tok)
	}
}
