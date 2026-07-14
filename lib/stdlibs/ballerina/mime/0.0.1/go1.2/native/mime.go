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

package native

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"strconv"
	"strings"
	"sync"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "mime"
)

// EntityBodyKind identifies the type of body content stored on an Entity.
// Exported so sibling stdlibs (e.g. http) that construct or read mime:Entity
// values natively can do so without duplicating this representation.
type EntityBodyKind int

const (
	BodyNone EntityBodyKind = iota
	BodyText
	BodyJSON
	BodyBytes
	BodyParts
)

// EntityBody holds the body payload attached to a Ballerina Entity object.
type EntityBody struct {
	Kind  EntityBodyKind
	Text  string
	JSON  values.BalValue
	Bytes []byte
	Parts []*values.Object
}

const entityBodyField = "$mimeBody"

// GetEntityBody returns the native body state attached to a mime:Entity object, or nil if unset.
func GetEntityBody(obj *values.Object) *EntityBody {
	v, ok := obj.Get(entityBodyField)
	if !ok {
		return nil
	}
	b, _ := v.(*EntityBody)
	return b
}

// SetEntityBody attaches native body state to a mime:Entity object.
func SetEntityBody(obj *values.Object, body *EntityBody) {
	obj.Put(entityBodyField, body)
}


func mimeError(typeName, msg string) values.BalValue {
	return values.NewError(semtypes.ERROR, msg, nil, typeName, nil)
}

// BytesForBody returns the byte representation of an Entity's body regardless of which
// setter populated it, matching jBallerina's model where every accessor lazily converts
// from the entity's underlying data source rather than requiring an exact kind match.
// Exported so sibling stdlibs (e.g. http, serializing multipart parts) can reuse it.
func BytesForBody(body *EntityBody) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf("Entity body is not a byte[] value")
	}
	switch body.Kind {
	case BodyBytes:
		return body.Bytes, nil
	case BodyText:
		return []byte(body.Text), nil
	case BodyJSON:
		return values.ToJSONByteArray(body.JSON)
	default:
		return nil, fmt.Errorf("Entity body is not a byte[] value")
	}
}

// stringForBody returns the string representation of an Entity's body regardless of
// which setter populated it, mirroring BytesForBody for the text accessor.
func stringForBody(body *EntityBody) (string, error) {
	if body == nil {
		return "", fmt.Errorf("Entity body is not a text value")
	}
	switch body.Kind {
	case BodyText:
		return body.Text, nil
	case BodyBytes:
		return string(body.Bytes), nil
	case BodyJSON:
		b, err := values.ToJSONByteArray(body.JSON)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("Entity body is not a text value")
	}
}

// mimeEncode produces MIME-compatible base64 (76-char line length, \r\n separators),
// matching Java's Base64.getMimeEncoder() default behaviour.
func mimeEncode(data []byte) string {
	const lineLen = 76
	encoded := base64.StdEncoding.EncodeToString(data)
	if len(encoded) <= lineLen {
		return encoded
	}
	var sb strings.Builder
	for i := 0; i < len(encoded); i++ {
		if i > 0 && i%lineLen == 0 {
			sb.WriteString("\r\n")
		}
		sb.WriteByte(encoded[i])
	}
	return sb.String()
}

// mimeDecode strips MIME whitespace (\r, \n, space, tab) before decoding,
// matching Java's Base64.getMimeDecoder() leniency.
func mimeDecode(s string) ([]byte, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	return base64.StdEncoding.DecodeString(cleaned)
}

// listToBytes converts a Ballerina byte[] to a Go []byte.
func listToBytes(list *values.List) []byte {
	b := make([]byte, list.Len())
	for i := range list.Len() {
		b[i] = byte(list.Get(i).(int64))
	}
	return b
}

// bytesToList converts a []byte to a Ballerina byte[] list value.
func bytesToList(ctx *extern.Context, data []byte) *values.List {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	bld := semtypes.NewListDefinition()
	ty := bld.DefineListTypeWrappedWithEnvSemType(ctx.Env.TypeEnv, semtypes.BYTE)
	return values.NewList(ty, semtypes.ToListAtomicType(ctx.TypeCtx, ty), false, nil, 0, items)
}

// buildParamsMap creates a Ballerina map<string> from a Go string map.
func buildParamsMap(tc semtypes.Context, env semtypes.Env, params map[string]string) *values.Map {
	mmd := semtypes.NewMappingDefinition()
	ty := mmd.DefineMappingTypeWrapped(env, nil, semtypes.STRING)
	entries := make([]values.MapEntry, 0, len(params))
	for k, v := range params {
		entries = append(entries, values.MapEntry{Key: k, Value: v})
	}
	return values.NewMap(ty, semtypes.ToMappingAtomicType(tc, ty), false, entries)
}

// formatParam quotes a parameter value if it contains chars that require quoting.
func formatParam(val string) string {
	for _, c := range val {
		if c == ' ' || c == ',' || c == ';' || c == '"' || c == '\\' || c == '(' || c == ')' || c == '<' || c == '>' || c == '@' || c == ':' || c == '/' || c == '[' || c == ']' || c == '?' || c == '=' {
			escaped := strings.ReplaceAll(val, `\`, `\\`)
			escaped = strings.ReplaceAll(escaped, `"`, `\"`)
			return `"` + escaped + `"`
		}
	}
	return val
}

func initMimeModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	jsonListType, jsonMapType := values.JSONListAndMapTypes(semtypes.ContextFrom(env))

	var (
		once          sync.Once
		byteArrayType semtypes.SemType
	)
	ensureTypes := func(ctx *extern.Context) {
		once.Do(func() {
			bld := semtypes.NewListDefinition()
			byteArrayType = bld.DefineListTypeWrappedWithEnvSemType(ctx.Env.TypeEnv, semtypes.BYTE)
		})
	}

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetJson",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			SetEntityBody(obj, &EntityBody{Kind: BodyJSON, JSON: args[1]})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetJson",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			body := GetEntityBody(obj)
			if body != nil && body.Kind == BodyJSON {
				return body.JSON, nil
			}
			text, err := stringForBody(body)
			if err != nil {
				return mimeError("ParserError", "Entity body is not a JSON value"), nil
			}
			dec := json.NewDecoder(strings.NewReader(text))
			v, err := values.DecodeJSON(dec, ctx.TypeCtx, jsonListType, jsonMapType)
			if err != nil {
				return mimeError("ParserError", "Error occurred while retrieving the json payload from the entity: "+err.Error()), nil
			}
			return v, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetText",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			text, _ := args[1].(string)
			SetEntityBody(obj, &EntityBody{Kind: BodyText, Text: text})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetText",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			body := GetEntityBody(obj)
			text, err := stringForBody(body)
			if err != nil {
				return mimeError("ParserError", "Entity body is not a text value"), nil
			}
			return text, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetByteArray",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			list, ok := args[1].(*values.List)
			if !ok {
				return nil, fmt.Errorf("second argument must be a byte array")
			}
			SetEntityBody(obj, &EntityBody{Kind: BodyBytes, Bytes: listToBytes(list)})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetByteArray",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			ensureTypes(ctx)
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			body := GetEntityBody(obj)
			data, err := BytesForBody(body)
			if err != nil {
				return mimeError("ParserError", err.Error()), nil
			}
			items := make([]values.BalValue, len(data))
			for i, b := range data {
				items[i] = int64(b)
			}
			return values.NewList(byteArrayType, semtypes.ToListAtomicType(ctx.TypeCtx, byteArrayType), false, nil, 0, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externIntToString",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			n, ok := args[0].(int64)
			if !ok {
				return nil, fmt.Errorf("argument must be an int")
			}
			return strconv.FormatInt(n, 10), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseInt",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("argument must be a string")
			}
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return values.NewErrorWithMessage("'int' from string: invalid number format: " + s), nil
			}
			return n, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getMediaType",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			contentType, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("argument must be a string")
			}
			mediatype, params, err := mime.ParseMediaType(contentType)
			if err != nil {
				return mimeError("InvalidContentTypeError", "Invalid content-type: "+contentType), nil
			}
			primaryType := ""
			subType := ""
			suffix := ""
			parts := strings.SplitN(mediatype, "/", 2)
			if len(parts) >= 1 {
				primaryType = parts[0]
			}
			if len(parts) == 2 {
				subParts := strings.SplitN(parts[1], "+", 2)
				subType = subParts[0]
				if len(subParts) == 2 {
					suffix = subParts[1]
				}
			}
			paramsMap := buildParamsMap(ctx.TypeCtx, ctx.Env.TypeEnv, params)
			return values.NewObject(
				semtypes.OBJECT,
				map[string]values.BalValue{
					"primaryType": primaryType,
					"subType":     subType,
					"suffix":      suffix,
					"parameters":  paramsMap,
				},
				map[string]string{
					"getBaseType": "ballerina/mime:MediaType.getBaseType",
					"toString":    "ballerina/mime:MediaType.toString",
				},
				nil,
			), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getContentDispositionObject",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			contentDisposition, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("argument must be a string")
			}
			disposition, params, _ := mime.ParseMediaType(contentDisposition)
			fileName := params["filename"]
			name := params["name"]
			delete(params, "filename")
			delete(params, "name")
			paramsMap := buildParamsMap(ctx.TypeCtx, ctx.Env.TypeEnv, params)
			return values.NewObject(
				semtypes.OBJECT,
				map[string]values.BalValue{
					"fileName":    fileName,
					"disposition": disposition,
					"name":        name,
					"parameters":  paramsMap,
				},
				map[string]string{
					"toString": "ballerina/mime:ContentDisposition.toString",
				},
				nil,
			), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "convertContentDispositionToString",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("argument must be a ContentDisposition object")
			}
			dispVal, _ := obj.Get("disposition")
			disposition, _ := dispVal.(string)
			if disposition == "" {
				return "", nil
			}
			result := disposition
			nameVal, _ := obj.Get("name")
			if name, ok := nameVal.(string); ok && name != "" {
				result += "; name=" + formatParam(name)
			}
			fileNameVal, _ := obj.Get("fileName")
			if fileName, ok := fileNameVal.(string); ok && fileName != "" {
				result += "; filename=" + formatParam(fileName)
			}
			paramsVal, _ := obj.Get("parameters")
			if paramsMap, ok := paramsVal.(*values.Map); ok {
				for _, k := range paramsMap.Keys() {
					v, _ := paramsMap.Get(k)
					vStr, _ := v.(string)
					result += "; " + k + "=" + formatParam(vStr)
				}
			}
			return result, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "base64Encode",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			charset := "utf-8"
			if len(args) > 1 {
				if cs, ok := args[1].(string); ok {
					charset = cs
				}
			}
			switch v := args[0].(type) {
			case string:
				_ = charset
				encoded := mimeEncode([]byte(v))
				return encoded, nil
			case *values.List:
				data := listToBytes(v)
				encodedStr := mimeEncode(data)
				return bytesToList(ctx, []byte(encodedStr)), nil
			default:
				return mimeError("EncodeError", "unsupported content type for base64 encoding"), nil
			}
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "base64Decode",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			charset := "utf-8"
			if len(args) > 1 {
				if cs, ok := args[1].(string); ok {
					charset = cs
				}
			}
			switch v := args[0].(type) {
			case string:
				_ = charset
				decoded, err := mimeDecode(v)
				if err != nil {
					return mimeError("DecodeError", "base64 decoding failed: "+err.Error()), nil
				}
				return string(decoded), nil
			case *values.List:
				data := listToBytes(v)
				decoded, err := mimeDecode(string(data))
				if err != nil {
					return mimeError("DecodeError", "base64 decoding failed: "+err.Error()), nil
				}
				return bytesToList(ctx, decoded), nil
			default:
				return mimeError("DecodeError", "unsupported content type for base64 decoding"), nil
			}
		})
}

func init() {
	runtime.RegisterModuleInitializer(initMimeModule)
}
