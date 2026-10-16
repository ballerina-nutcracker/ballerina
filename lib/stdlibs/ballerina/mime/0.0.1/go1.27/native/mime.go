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
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"

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
// fromChannel marks bytes read from a byte channel (a file body or a decoded part),
// which jBallerina decodes in the Content-Type charset rather than as UTF-8.
type entityBody struct {
	kind        bodyKind
	text        string
	json        values.BalValue
	bytes       []byte
	parts       *values.List
	xml         values.XMLValue
	channel     *values.Object
	fromChannel bool
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

func mimeErrorWithCause(typeName, msg string, cause error) values.BalValue {
	return values.NewError(semtypes.Error, msg, values.NewErrorWithMessage(cause.Error()), typeName, nil)
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
	cached := &entityBody{kind: bodyBytes, bytes: data, fromChannel: true}
	setEntityBody(obj, cached)
	return cached, nil
}

// bytesForBody returns the byte representation of an Entity's body regardless of which
// setter populated it, matching jBallerina's model where every accessor lazily converts
// from the entity's underlying data source. An absent or multipart body reads as empty,
// and a text, JSON or XML body is encoded in the Content-Type charset.
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
		return encodeCharset(body.text, contentTypeCharset(obj))
	case bodyJSON:
		if body.json == nil {
			return []byte{}, nil
		}
		return encodeCharset(formatJSON(body.json, ","), contentTypeCharset(obj))
	case bodyXML:
		return encodeCharset(body.xml.XMLString(), contentTypeCharset(obj))
	default:
		return []byte{}, nil
	}
}

func contentTypeCharset(obj *values.Object) string {
	contentType, _ := entityHeaderValue(obj, "content-type")
	mt, err := parseMediaType(contentType)
	if err != nil {
		return ""
	}
	return mt.param("charset")
}

// encodeCharset encodes s as Java's String.getBytes(charset) does: UTF-8 when no charset
// is given, and '?' for a character the charset cannot represent.
func encodeCharset(s, charset string) ([]byte, error) {
	if charset == "" {
		return []byte(s), nil
	}
	enc, err := lookupCharset(charset)
	if err != nil {
		// Java's UnsupportedEncodingException message is the charset name alone.
		return nil, errors.New(charset)
	}
	if out, err := enc.NewEncoder().Bytes([]byte(s)); err == nil {
		return out, nil
	}
	encoder := enc.NewEncoder()
	var out []byte
	for _, r := range s {
		b, err := encoder.Bytes([]byte(string(r)))
		if err != nil {
			b = []byte{'?'}
		}
		out = append(out, b...)
	}
	return out, nil
}

// decodeUTF8 replaces each malformed sequence with U+FFFD, as Java's UTF-8 decoder does.
func decodeUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	decoded, _ := unicode.UTF8.NewDecoder().Bytes(data)
	return string(decoded)
}

// decodeCharset decodes data as Java's new String(bytes, charset) does: UTF-8 when no
// charset is given, and U+FFFD for each malformed sequence.
func decodeCharset(data []byte, charset string) (string, error) {
	if charset == "" {
		return decodeUTF8(data), nil
	}
	enc, err := lookupCharset(charset)
	if err != nil {
		return "", errors.New(charset)
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// stringForBody returns the string representation of an Entity's body regardless of
// which setter populated it, mirroring bytesForBody for the text accessor.
func stringForBody(ctx *extern.Context, obj *values.Object) (string, error) {
	body, err := materializeBody(ctx, obj)
	if err != nil {
		return "", err
	}
	if body == nil {
		//nolint:staticcheck // cause text mirrors jBallerina's verbatim
		return "", errors.New("String payload is null")
	}
	switch body.kind {
	case bodyText:
		return body.text, nil
	case bodyBytes:
		if body.fromChannel {
			return decodeCharset(body.bytes, contentTypeCharset(obj))
		}
		return decodeUTF8(body.bytes), nil
	case bodyJSON:
		if body.json == nil {
			//nolint:staticcheck // cause text mirrors jBallerina's verbatim
			return "", errors.New("String payload is null")
		}
		return formatJSON(body.json, ", "), nil
	case bodyXML:
		return jsonQuote(body.xml.XMLString()), nil
	default:
		//nolint:staticcheck // cause text mirrors jBallerina's verbatim
		return "", errors.New("String payload is null")
	}
}

// formatJSON serializes a JSON body as jBallerina does: keys in insertion order, numbers in
// Ballerina notation, and a top-level string unquoted. sep separates members and elements.
func formatJSON(v values.BalValue, sep string) string {
	if s, ok := v.(string); ok {
		return s
	}
	var sb strings.Builder
	writeJSON(&sb, v, sep)
	return sb.String()
}

func writeJSON(sb *strings.Builder, v values.BalValue, sep string) {
	switch v := v.(type) {
	case nil:
		sb.WriteString("null")
	case string:
		sb.WriteString(jsonQuote(v))
	case *values.Map:
		sb.WriteByte('{')
		for i, k := range v.Keys() {
			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(jsonQuote(k))
			sb.WriteByte(':')
			member, _ := v.Get(k)
			writeJSON(sb, member, sep)
		}
		sb.WriteByte('}')
	case *values.List:
		sb.WriteByte('[')
		for i := range v.Len() {
			if i > 0 {
				sb.WriteString(sep)
			}
			writeJSON(sb, v.Get(i), sep)
		}
		sb.WriteByte(']')
	default:
		sb.WriteString(values.String(v, nil))
	}
}

// jsonQuote reproduces jBallerina's JSON string quoting, where '/' is escaped only if some
// other character needs escaping.
func jsonQuote(s string) string {
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
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
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

// lookupCharset resolves a charset name (e.g. "utf-8", "iso-8859-1") to its x/text
// encoding. Names missing from the IANA index fall back to the WHATWG index, which
// covers aliases Java also accepts, such as "utf8".
func lookupCharset(charset string) (encoding.Encoding, error) {
	enc, err := ianaindex.IANA.Encoding(charset)
	if err != nil || enc == nil {
		enc, err = htmlindex.Get(charset)
	}
	if err != nil || enc == nil {
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}
	return enc, nil
}

// mimeDecode drops every character outside the base64 alphabet and accepts missing `=`
// padding before decoding, as RFC 2045 and Java's Base64.getMimeDecoder() do.
func mimeDecode(s string) ([]byte, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '=' || r == '+' || r == '/' || ('0' <= r && r <= '9') || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') {
			return r
		}
		return -1
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
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetByteChannel", setByteChannelExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetJson", setJSONExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetJson", getJSONExtern(t))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetXml", setXMLExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetXml", getXMLExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetText", setTextExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetText", getTextExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetByteArray", setByteArrayExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetByteArray", getByteArrayExtern(t))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetBodyParts", setBodyPartsExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetBodyParts", getBodyPartsExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseMediaType", parseMediaTypeExtern(t))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externParseContentDisposition", parseContentDispositionExtern(t))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "convertContentDispositionToString", contentDispositionToStringExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externBase64Encode", base64EncodeExtern(t))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externBase64Decode", base64DecodeExtern(t))
}

func setByteChannelExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		channel, _ := args[1].(*values.Object)
		setEntityBody(obj, &entityBody{kind: bodyChannel, channel: channel})
		return nil, nil
	}
}

func setJSONExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		setEntityBody(obj, &entityBody{kind: bodyJSON, json: args[1]})
		return nil, nil
	}
}

func getJSONExtern(t *mimeTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		body := getEntityBody(obj)
		if body != nil && body.kind == bodyJSON {
			return body.json, nil
		}
		text, err := stringForBody(ctx, obj)
		if err != nil {
			return mimeErrorWithCause("ParserError", "Error occurred while extracting json data from entity", err), nil
		}
		v, err := t.parseJSON(text)
		if err != nil {
			return mimeErrorWithCause("ParserError", "Error occurred while extracting json data from entity", err), nil
		}
		return v, nil
	}
}

func setXMLExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		xmlContent, _ := args[1].(values.XMLValue)
		setEntityBody(obj, &entityBody{kind: bodyXML, xml: xmlContent})
		return nil, nil
	}
}

func getXMLExtern() extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		xmlVal, err := xmlForBody(ctx, obj)
		if err != nil {
			return mimeErrorWithCause("ParserError", "Error occurred while extracting xml data from entity", err), nil
		}
		return xmlVal, nil
	}
}

func setTextExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		text, _ := args[1].(string)
		setEntityBody(obj, &entityBody{kind: bodyText, text: text})
		return nil, nil
	}
}

func getTextExtern() extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		text, err := stringForBody(ctx, obj)
		if err != nil {
			return mimeErrorWithCause("ParserError", "Error occurred while extracting text data from entity", err), nil
		}
		return text, nil
	}
}

func setByteArrayExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		list, _ := args[1].(*values.List)
		setEntityBody(obj, &entityBody{kind: bodyBytes, bytes: listToBytes(list)})
		return nil, nil
	}
}

func getByteArrayExtern(t *mimeTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		data, err := bytesForBody(ctx, obj)
		if err != nil {
			return mimeError("ParserError", "Error occurred while extracting blob data from entity: "+err.Error()), nil
		}
		return t.byteList(data), nil
	}
}

// parseMediaTypeExtern fills a MediaType object that the .bal side constructs with `new`,
// so it carries its real class type for `is` checks.
func parseMediaTypeExtern(t *mimeTypes) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		contentType, _ := args[1].(string)
		mt, err := parseMediaType(contentType)
		if err != nil {
			return mimeError("InvalidContentTypeError", err.Error()), nil
		}
		obj.Put("primaryType", mt.primaryType)
		obj.Put("subType", mt.subType)
		obj.Put("suffix", mt.suffix)
		obj.Put("parameters", t.stringMap(mt.params))
		return nil, nil
	}
}

func parseContentDispositionExtern(t *mimeTypes) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		value, _ := args[1].(string)
		cd := parseContentDisposition(value)
		obj.Put("disposition", cd.disposition)
		obj.Put("name", cd.name)
		obj.Put("fileName", cd.fileName)
		obj.Put("parameters", t.stringMap(cd.params))
		return nil, nil
	}
}

func contentDispositionToStringExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		return contentDispositionOf(obj).String(), nil
	}
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

func base64EncodeExtern(t *mimeTypes) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		charset, _ := args[1].(string)
		if v, ok := args[0].(*values.List); ok {
			return t.byteList([]byte(mimeEncode(listToBytes(v)))), nil
		}
		v, _ := args[0].(string)
		data, err := encodeCharset(v, charset)
		if err != nil {
			return mimeError("EncodeError", "unsupported charset: "+charset), nil
		}
		return mimeEncode(data), nil
	}
}

func base64DecodeExtern(t *mimeTypes) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		charset, _ := args[1].(string)
		if v, ok := args[0].(*values.List); ok {
			decoded, err := mimeDecode(string(listToBytes(v)))
			if err != nil {
				return mimeError("DecodeError", "base64 decoding failed: "+err.Error()), nil
			}
			return t.byteList(decoded), nil
		}
		v, _ := args[0].(string)
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
	}
}

func init() {
	runtime.RegisterModuleInitializer(initMimeModule)
}
