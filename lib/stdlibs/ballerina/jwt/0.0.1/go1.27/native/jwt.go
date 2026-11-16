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

// Package native implements the extern functions of ballerina/jwt.
package native

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "jwt"
)

type jwtTypes struct {
	byteArrTy  semtypes.SemType
	jsonListTy semtypes.SemType
	jsonMapTy  semtypes.SemType
}

func initJwtModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	jsonTy := semtypes.CreateJSON(semtypes.ContextFrom(env))
	byteArrDef := semtypes.NewListDefinition()
	jsonListDef := semtypes.NewListDefinition()
	jsonMapDef := semtypes.NewMappingDefinition()
	types := jwtTypes{
		byteArrTy:  byteArrDef.Define(env, nil, semtypes.ListRest(semtypes.Byte)),
		jsonListTy: jsonListDef.Define(env, nil, semtypes.ListRest(jsonTy)),
		jsonMapTy:  jsonMapDef.Define(env, nil, jsonTy),
	}
	runtime.RegisterExternFunction(rt, orgName, moduleName, "encodeBase64Url", encodeBase64URL)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externDecodeBase64Url", decodeBase64URL(types))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toJsonString", toJSONString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromJsonString", fromJSONString(types))
}

func init() {
	runtime.RegisterModuleInitializer(initJwtModule)
}

func encodeBase64URL(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return base64.RawURLEncoding.EncodeToString(byteSlice(args[0].(*values.List))), nil
}

// decodeBase64URL accepts padded and unpadded input, as Java's URL decoder does.
func decodeBase64URL(types jwtTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		input := args[0].(string)
		enc := base64.RawURLEncoding
		if strings.HasSuffix(input, "=") {
			enc = base64.URLEncoding
		}
		decoded, err := enc.DecodeString(input)
		if err != nil {
			return values.NewErrorWithMessage("Input is not a valid Base64 URL encoded value. " + err.Error()), nil
		}
		items := make([]values.BalValue, len(decoded))
		for i, b := range decoded {
			items[i] = int64(b)
		}
		atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), types.byteArrTy)
		return values.NewList(types.byteArrTy, atomic, false, nil, len(items), items), nil
	}
}

func byteSlice(list *values.List) []byte {
	out := make([]byte, list.Len())
	for i := range out {
		out[i] = byte(list.Get(i).(int64))
	}
	return out
}

// toJSONString renders a value as jBallerina's toJsonString does: members in insertion order, separated by ", ".
func toJSONString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	var sb strings.Builder
	writeJSON(&sb, args[0])
	return sb.String(), nil
}

func writeJSON(sb *strings.Builder, v values.BalValue) {
	switch t := v.(type) {
	case nil:
		sb.WriteString("null")
	case bool:
		sb.WriteString(strconv.FormatBool(t))
	case int64:
		sb.WriteString(strconv.FormatInt(t, 10))
	case float64:
		sb.WriteString(javaDoubleString(t))
	case *decimal.Decimal:
		sb.WriteString(t.String())
	case string:
		writeJSONString(sb, t)
	case *values.Map:
		sb.WriteByte('{')
		for i, k := range t.Keys() {
			if i > 0 {
				sb.WriteString(", ")
			}
			writeJSONString(sb, k)
			sb.WriteByte(':')
			val, _ := t.Get(k)
			writeJSON(sb, val)
		}
		sb.WriteByte('}')
	case *values.List:
		sb.WriteByte('[')
		for i := range t.Len() {
			if i > 0 {
				sb.WriteString(", ")
			}
			writeJSON(sb, t.Get(i))
		}
		sb.WriteByte(']')
	}
}

func writeJSONString(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '/':
			sb.WriteString(`\/`)
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
}

// javaDoubleString formats a float as Java's Double.toString does, which jBallerina's toJsonString uses.
func javaDoubleString(f float64) string {
	abs := math.Abs(f)
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 || (abs >= 1e-3 && abs < 1e7) {
		return values.FormatFloat(f)
	}
	mantissa, exponent, _ := strings.Cut(strconv.FormatFloat(f, 'E', -1, 64), "E")
	if !strings.Contains(mantissa, ".") {
		mantissa += ".0"
	}
	exp, _ := strconv.Atoi(exponent)
	return mantissa + "E" + strconv.Itoa(exp)
}

// fromJSONString parses JSON as lang.value:fromJsonString does: object members keep their order, integers
// become int and other numbers decimal.
func fromJSONString(types jwtTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		dec := json.NewDecoder(strings.NewReader(args[0].(string)))
		dec.UseNumber()
		p := jsonParser{dec: dec, tc: ctx.TypeCtx(), types: types}
		v, err := p.value()
		if err == nil {
			if _, extra := dec.Token(); extra != io.EOF {
				err = errors.New("unexpected data after the JSON value")
			}
		}
		if err != nil {
			return values.NewErrorWithMessage(err.Error()), nil
		}
		return v, nil
	}
}

type jsonParser struct {
	dec   *json.Decoder
	tc    semtypes.Context
	types jwtTypes
}

func (p *jsonParser) value() (values.BalValue, error) {
	tok, err := p.dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			return p.object()
		}
		return p.array()
	case json.Number:
		return numberValue(t.String())
	default:
		return t, nil
	}
}

func (p *jsonParser) object() (values.BalValue, error) {
	m := values.NewMap(p.types.jsonMapTy, semtypes.ToMappingAtomicType(p.tc, p.types.jsonMapTy), false, nil)
	for p.dec.More() {
		keyTok, err := p.dec.Token()
		if err != nil {
			return nil, err
		}
		item, err := p.value()
		if err != nil {
			return nil, err
		}
		m.Put(p.tc, keyTok.(string), item)
	}
	if _, err := p.dec.Token(); err != nil {
		return nil, err
	}
	return m, nil
}

func (p *jsonParser) array() (values.BalValue, error) {
	var items []values.BalValue
	for p.dec.More() {
		item, err := p.value()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if _, err := p.dec.Token(); err != nil {
		return nil, err
	}
	return values.NewList(p.types.jsonListTy, semtypes.ToListAtomicType(p.tc.Env(), p.types.jsonListTy), false, nil, 0, items), nil
}

func numberValue(s string) (values.BalValue, error) {
	if !strings.ContainsAny(s, ".eE") {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, errors.New("number '" + s + "' is out of the int range")
		}
		return i, nil
	}
	d, derr := decimal.FromString(s)
	if derr != nil {
		return nil, errors.New("number '" + s + "' is out of the decimal range")
	}
	return d, nil
}
