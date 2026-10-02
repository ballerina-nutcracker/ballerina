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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "mime"
)

type bodyKind int

const (
	bodyText bodyKind = iota
	bodyJSON
	bodyBytes
	bodyParts
	bodyXML
	bodyChannel
)

// entityBody holds the body payload attached to a Ballerina Entity object.
type entityBody struct {
	kind    bodyKind
	text    string
	json    values.BalValue
	bytes   []byte
	parts   *values.List
	xml     values.XMLValue
	channel *values.Object
}

const entityBodyField = "$mimeBody"

type mimeTypes struct {
	byteArrayTy   semtypes.SemType
	byteArrayAtom *semtypes.ListAtomicType
	stringMapTy   semtypes.SemType
	stringMapAtom *semtypes.MappingAtomicType
	jsonListTy    semtypes.SemType
	jsonListAtom  *semtypes.ListAtomicType
	jsonMapTy     semtypes.SemType
	jsonMapAtom   *semtypes.MappingAtomicType
}

func newMimeTypes(env semtypes.Env) *mimeTypes {
	tc := semtypes.ContextFrom(env)
	jsonTy := semtypes.CreateJSON(tc)
	byteArrayLd := semtypes.NewListDefinition()
	stringMapMd := semtypes.NewMappingDefinition()
	jsonListLd := semtypes.NewListDefinition()
	jsonMapMd := semtypes.NewMappingDefinition()
	byteArrayTy := byteArrayLd.Define(env, nil, semtypes.ListRest(semtypes.Byte))
	stringMapTy := stringMapMd.Define(env, nil, semtypes.String)
	jsonListTy := jsonListLd.Define(env, nil, semtypes.ListRest(jsonTy))
	jsonMapTy := jsonMapMd.Define(env, nil, jsonTy)
	return &mimeTypes{
		byteArrayTy:   byteArrayTy,
		byteArrayAtom: semtypes.ToListAtomicType(env, byteArrayTy),
		stringMapTy:   stringMapTy,
		stringMapAtom: semtypes.ToMappingAtomicType(tc, stringMapTy),
		jsonListTy:    jsonListTy,
		jsonListAtom:  semtypes.ToListAtomicType(env, jsonListTy),
		jsonMapTy:     jsonMapTy,
		jsonMapAtom:   semtypes.ToMappingAtomicType(tc, jsonMapTy),
	}
}

func (t *mimeTypes) byteList(data []byte) *values.List {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return values.NewList(t.byteArrayTy, t.byteArrayAtom, false, nil, 0, items)
}

func (t *mimeTypes) stringMap(params []headerParam) *values.Map {
	entries := make([]values.MapEntry, len(params))
	for i, p := range params {
		entries[i] = values.MapEntry{Key: p.name, Value: p.value}
	}
	return values.NewMap(t.stringMapTy, t.stringMapAtom, false, entries)
}

// getEntityBody returns the native body state attached to a mime:Entity object, or nil if unset.
func getEntityBody(obj *values.Object) *entityBody {
	v, ok := obj.Get(entityBodyField)
	if !ok {
		return nil
	}
	b, _ := v.(*entityBody)
	return b
}

func setEntityBody(obj *values.Object, body *entityBody) {
	obj.Put(entityBodyField, body)
}

func mimeError(typeName, msg string) values.BalValue {
	return values.NewError(semtypes.Error, msg, nil, typeName, nil)
}

// invokeChannelMethod calls a no-argument io:ReadableByteChannel method through the
// interpreter's normal object dispatch, so mime never reaches into io's native state.
func invokeChannelMethod(ctx *extern.Context, channel *values.Object, name string) (values.BalValue, error) {
	handle, ok := ctx.LookupObjectMethod(channel, name)
	if !ok {
		return nil, fmt.Errorf("byte channel does not support %s", name)
	}
	result, err := ctx.InvokeMethod(handle, []values.BalValue{channel})
	if err != nil {
		return nil, err
	}
	if errVal, ok := result.(*values.Error); ok {
		return nil, errors.New(errVal.Message)
	}
	return result, nil
}

func readAllFromChannel(ctx *extern.Context, channel *values.Object) ([]byte, error) {
	result, err := invokeChannelMethod(ctx, channel, "readAll")
	if err != nil {
		return nil, err
	}
	list, ok := result.(*values.List)
	if !ok {
		return nil, fmt.Errorf("unexpected result from byte channel readAll")
	}
	return listToBytes(list), nil
}

// materializeBody drains and closes a lazy byte-channel body, caching the bytes back
// onto the entity (jBallerina's EntityBodyHandler does the same): a channel can only
// be drained once, so later accessors must reuse the cached value.
func materializeBody(ctx *extern.Context, obj *values.Object) (*entityBody, error) {
	body := getEntityBody(obj)
	if body == nil || body.kind != bodyChannel {
		return body, nil
	}
	data, readErr := readAllFromChannel(ctx, body.channel)
	_, closeErr := invokeChannelMethod(ctx, body.channel, "close")
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	cached := &entityBody{kind: bodyBytes, bytes: data}
	setEntityBody(obj, cached)
	return cached, nil
}

// bytesForBody returns the byte representation of an Entity's body regardless of which
// setter populated it, matching jBallerina's model where every accessor lazily converts
// from the entity's underlying data source. An absent or multipart body reads as empty.
func bytesForBody(ctx *extern.Context, obj *values.Object) ([]byte, error) {
	body, err := materializeBody(ctx, obj)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return []byte{}, nil
	}
	switch body.kind {
	case bodyBytes:
		return body.bytes, nil
	case bodyText:
		return []byte(body.text), nil
	case bodyJSON:
		return values.ToJSONByteArray(body.json)
	case bodyXML:
		return []byte(body.xml.XMLString()), nil
	default:
		return []byte{}, nil
	}
}

// stringForBody returns the string representation of an Entity's body regardless of
// which setter populated it, mirroring bytesForBody for the text accessor.
func stringForBody(ctx *extern.Context, obj *values.Object) (string, error) {
	body, err := materializeBody(ctx, obj)
	if err != nil {
		return "", err
	}
	if body == nil {
		//nolint:staticcheck // error text mirrors jBallerina's runtime message verbatim
		return "", fmt.Errorf("Entity body is not a text value")
	}
	switch body.kind {
	case bodyText:
		return body.text, nil
	case bodyBytes:
		return string(body.bytes), nil
	case bodyJSON:
		b, err := values.ToJSONByteArray(body.json)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case bodyXML:
		return jsonQuoteXMLText(body.xml.XMLString()), nil
	default:
		//nolint:staticcheck // error text mirrors jBallerina's runtime message verbatim
		return "", fmt.Errorf("Entity body is not a text value")
	}
}

// jsonQuoteXMLText reproduces jBallerina's text form of an XML body: the serialized XML
// as a JSON string, where '/' is escaped only if some other character needs escaping.
func jsonQuoteXMLText(s string) string {
	if !strings.ContainsFunc(s, needsJSONEscape) {
		return `"` + s + `"`
	}
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\', '/':
			sb.WriteByte('\\')
			sb.WriteRune(r)
		case '\t':
			sb.WriteString(`\t`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&sb, `\u%04x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func needsJSONEscape(r rune) bool {
	return r == '"' || r == '\\' || r < 0x20
}

// xmlForBody parses the body's string form as an XML document, matching jBallerina's
// getXml: an empty body is an empty sequence, otherwise exactly one root element is required.
func xmlForBody(ctx *extern.Context, obj *values.Object) (values.XMLValue, error) {
	body, err := materializeBody(ctx, obj)
	if err != nil {
		return nil, err
	}
	if body != nil && body.kind == bodyXML {
		return body.xml, nil
	}
	text, err := stringForBody(ctx, obj)
	if err != nil {
		return nil, err
	}
	xmlVal, err := values.ParseAsXMLValue(ctx.TypeCtx(), text, values.XMLLenientMode)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) != "" && !hasSingleRootElement(xmlVal) {
		return nil, errors.New("xml content must have exactly one root element")
	}
	return xmlVal, nil
}

func hasSingleRootElement(xmlVal values.XMLValue) bool {
	elements := 0
	for _, item := range xmlVal.IterItems() {
		switch item := item.(type) {
		case *values.XMLElement:
			elements++
		case *values.XMLText:
			if strings.TrimSpace(item.XMLString()) != "" {
				return false
			}
		}
	}
	return elements == 1
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

// lookupCharset resolves an IANA charset name (e.g. "utf-8", "iso-8859-1") to its
// x/text encoding, for transcoding a base64 payload's string form to/from raw bytes.
func lookupCharset(charset string) (encoding.Encoding, error) {
	enc, err := ianaindex.IANA.Encoding(charset)
	if err != nil || enc == nil {
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}
	return enc, nil
}

// mimeDecode strips MIME whitespace (\r, \n, space, tab) and accepts missing `=`
// padding before decoding, matching Java's Base64.getMimeDecoder() leniency.
func mimeDecode(s string) ([]byte, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	return base64.RawStdEncoding.DecodeString(strings.TrimRight(cleaned, "="))
}

// listToBytes converts a Ballerina byte[] to a Go []byte.
func listToBytes(list *values.List) []byte {
	b := make([]byte, list.Len())
	for i := range list.Len() {
		b[i] = byte(list.Get(i).(int64))
	}
	return b
}

func initMimeModule(rt *runtime.Runtime) {
	t := newMimeTypes(rt.GetTypeEnv())
	registerBodyExterns(rt, t)
	registerHeaderExterns(rt, t)
	registerBase64Externs(rt, t)
	registerMultipartExterns(rt)
}

func registerBodyExterns(rt *runtime.Runtime, t *mimeTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetByteChannel",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			channel, ok := args[1].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("second argument must be a ReadableByteChannel object")
			}
			setEntityBody(obj, &entityBody{kind: bodyChannel, channel: channel})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetJson",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			setEntityBody(obj, &entityBody{kind: bodyJSON, json: args[1]})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetJson",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			body := getEntityBody(obj)
			if body != nil && body.kind == bodyJSON {
				return body.json, nil
			}
			text, err := stringForBody(ctx, obj)
			if err != nil {
				return mimeError("ParserError", "Entity body is not a JSON value"), nil
			}
			v, err := t.parseJSON(text)
			if err != nil {
				return mimeError("ParserError", "Error occurred while extracting json data from entity: "+err.Error()), nil
			}
			return v, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetXml",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			xmlContent, ok := args[1].(values.XMLValue)
			if !ok {
				return nil, fmt.Errorf("second argument must be an xml value")
			}
			setEntityBody(obj, &entityBody{kind: bodyXML, xml: xmlContent})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetXml",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			xmlVal, err := xmlForBody(ctx, obj)
			if err != nil {
				return mimeError("ParserError", "Error occurred while extracting xml data from entity: "+err.Error()), nil
			}
			return xmlVal, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetText",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			text, _ := args[1].(string)
			setEntityBody(obj, &entityBody{kind: bodyText, text: text})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetText",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			text, err := stringForBody(ctx, obj)
			if err != nil {
				return mimeError("ParserError", "Entity body is not a text value"), nil
			}
			return text, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetByteArray",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			list, ok := args[1].(*values.List)
			if !ok {
				return nil, fmt.Errorf("second argument must be a byte array")
			}
			setEntityBody(obj, &entityBody{kind: bodyBytes, bytes: listToBytes(list)})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetByteArray",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			data, err := bytesForBody(ctx, obj)
			if err != nil {
				return mimeError("ParserError", err.Error()), nil
			}
			return t.byteList(data), nil
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
}

// registerHeaderExterns fills MediaType/ContentDisposition objects that the .bal side
// constructs with `new`, so they carry their real class type for `is` checks.
func registerHeaderExterns(rt *runtime.Runtime, t *mimeTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseMediaType",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be a MediaType object")
			}
			contentType, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("second argument must be a string")
			}
			mt, ok := parseMediaType(contentType)
			if !ok {
				return mimeError("InvalidContentTypeError", "Invalid content-type: "+contentType), nil
			}
			obj.Put("primaryType", mt.primaryType)
			obj.Put("subType", mt.subType)
			obj.Put("suffix", mt.suffix)
			obj.Put("parameters", t.stringMap(mt.params))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseContentDisposition",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be a ContentDisposition object")
			}
			value, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("second argument must be a string")
			}
			cd := parseContentDisposition(value)
			obj.Put("disposition", cd.disposition)
			obj.Put("name", cd.name)
			obj.Put("fileName", cd.fileName)
			obj.Put("parameters", t.stringMap(cd.params))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "convertContentDispositionToString",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("argument must be a ContentDisposition object")
			}
			return contentDispositionOf(obj).String(), nil
		})
}

func contentDispositionOf(obj *values.Object) contentDisposition {
	stringField := func(name string) string {
		v, _ := obj.Get(name)
		s, _ := v.(string)
		return s
	}
	cd := contentDisposition{
		disposition: stringField("disposition"),
		name:        stringField("name"),
		fileName:    stringField("fileName"),
	}
	paramsVal, _ := obj.Get("parameters")
	if params, ok := paramsVal.(*values.Map); ok {
		for _, k := range params.Keys() {
			v, _ := params.Get(k)
			s, _ := v.(string)
			cd.params = append(cd.params, headerParam{name: k, value: s})
		}
	}
	return cd
}

func registerBase64Externs(rt *runtime.Runtime, t *mimeTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externBase64Encode",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			charset := charsetArg(args)
			switch v := args[0].(type) {
			case string:
				enc, err := lookupCharset(charset)
				if err != nil {
					return mimeError("EncodeError", err.Error()), nil
				}
				data, err := enc.NewEncoder().String(v)
				if err != nil {
					return mimeError("EncodeError", "base64 encoding failed: "+err.Error()), nil
				}
				return mimeEncode([]byte(data)), nil
			case *values.List:
				return t.byteList([]byte(mimeEncode(listToBytes(v)))), nil
			default:
				return mimeError("EncodeError", "unsupported content type for base64 encoding"), nil
			}
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externBase64Decode",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			charset := charsetArg(args)
			switch v := args[0].(type) {
			case string:
				decoded, err := mimeDecode(v)
				if err != nil {
					return mimeError("DecodeError", "base64 decoding failed: "+err.Error()), nil
				}
				dec, err := lookupCharset(charset)
				if err != nil {
					return mimeError("DecodeError", err.Error()), nil
				}
				out, err := dec.NewDecoder().String(string(decoded))
				if err != nil {
					return mimeError("DecodeError", "base64 decoding failed: "+err.Error()), nil
				}
				return out, nil
			case *values.List:
				decoded, err := mimeDecode(string(listToBytes(v)))
				if err != nil {
					return mimeError("DecodeError", "base64 decoding failed: "+err.Error()), nil
				}
				return t.byteList(decoded), nil
			default:
				return mimeError("DecodeError", "unsupported content type for base64 decoding"), nil
			}
		})
}

func charsetArg(args []values.BalValue) string {
	if len(args) > 1 {
		if cs, ok := args[1].(string); ok {
			return cs
		}
	}
	return "utf-8"
}

func init() {
	runtime.RegisterModuleInitializer(initMimeModule)
}
