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

// Resource signature parameter binding, mirroring jBallerina's ParamHandler and the
// compiler plugin's default payload parameter inference (HttpPayloadParamIdentifier).

package native

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type paramKind uint8

const (
	requestParam paramKind = iota
	headersParam
	queryParam
	headerParam
	payloadParam
)

type resourceParam struct {
	kind paramKind
	// name is the query key or header name the parameter binds from.
	name string
	// label is the declared parameter name, used in signature errors.
	label string
	ty    semtypes.SemType
}

// scalarKind is jBallerina's "effective type tag": how a raw query or header string is parsed
// before being converted to the declared type.
type scalarKind uint8

const (
	invalidScalar scalarKind = iota
	stringScalar
	intScalar
	floatScalar
	booleanScalar
	decimalScalar
	mapScalar
)

var httpPackage = model.PackageIdentifier{Organization: orgName, Package: moduleName, Version: "0.0.1"}

var (
	payloadAnnotation = model.AnnotationKey(httpPackage, "Payload")
	headerAnnotation  = model.AnnotationKey(httpPackage, "Header")
	queryAnnotation   = model.AnnotationKey(httpPackage, "Query")
)

// bindingError is reported to the client as a JSON error response with the given status.
type bindingError struct {
	status  int
	message string
}

func badRequest(message string) *bindingError {
	return &bindingError{status: http.StatusBadRequest, message: message}
}

// jBallerina rejects these signatures at compile time; without a compiler plugin they are
// only detected when a request reaches the resource.
func invalidSignature(message string) *bindingError {
	return &bindingError{status: http.StatusInternalServerError, message: message}
}

// resourceArgs binds the non-path parameters of the resource behind handle from r. The
// request body is consumed or closed before it returns.
func resourceArgs(ctx *extern.Context, state *listenerState, handle extern.MethodHandle,
	accessor string, r *http.Request) ([]values.BalValue, *bindingError) {
	signature, sigOK := ctx.MethodSignature(handle)
	metadata, metaOK := ctx.MethodMetadata(handle)
	if !sigOK || !metaOK {
		closeBody(r)
		return nil, invalidSignature("resource method signature is unavailable")
	}
	params, bindErr := classifyParams(ctx.TypeCtx(), state, signature.Params, metadata.Params, accessor)
	if bindErr != nil {
		closeBody(r)
		return nil, bindErr
	}
	return bindParams(ctx, state, params, r)
}

// A server request's body is never nil.
func closeBody(r *http.Request) {
	_ = r.Body.Close()
}

func classifyParams(tc semtypes.Context, state *listenerState, sigParams []extern.Parameter,
	metaParams []extern.ParameterMetadata, accessor string) ([]resourceParam, *bindingError) {
	params := make([]resourceParam, len(sigParams))
	var unannotated []int
	hasRequest, hasHeaders, hasPayload := false, false, false
	for i, p := range sigParams {
		label := unescapeIdentifier(p.Name)
		annotations := metaParams[i].Annotations
		annotationCount := 0
		for _, key := range []string{payloadAnnotation, headerAnnotation, queryAnnotation} {
			if _, ok := annotations[key]; ok {
				annotationCount++
			}
		}
		if annotationCount > 1 {
			return nil, invalidSignature("invalid multiple resource parameter annotations for " + label +
				": expected one of the following types: http:Payload, http:CallerInfo, http:Header, http:Query")
		}
		params[i] = resourceParam{kind: queryParam, name: encodeParamName(label), label: label, ty: p.Type}
		switch {
		case isExactType(tc, state.requestTy, p.Type):
			if hasRequest {
				return nil, invalidSignature("invalid multiple http:Request parameter: " + label)
			}
			hasRequest = true
			params[i].kind = requestParam
		case isExactType(tc, state.headersTy, p.Type):
			if hasHeaders {
				return nil, invalidSignature("invalid multiple http:Headers parameter: " + label)
			}
			hasHeaders = true
			params[i].kind = headersParam
		case annotations[payloadAnnotation] != nil:
			if !isEntityBodyAccessor(accessor) {
				return nil, invalidSignature("invalid usage of payload annotation for a non entity body resource : " +
					accessor + ". Use an accessor that supports entity body")
			}
			if hasPayload {
				return nil, invalidSignature("invalid multiple 'http:Payload' annotation usage")
			}
			hasPayload = true
			params[i].kind = payloadParam
		case annotations[headerAnnotation] != nil:
			params[i].kind = headerParam
			if headerName, ok := annotationName(annotations[headerAnnotation]); ok {
				params[i].name = headerName
			}
			if !isValidHeaderType(tc, p.Type) {
				return nil, invalidSignature("invalid type of header param " + label + ": expected one of the " +
					"following types is expected: string,int,float,decimal,boolean, an array of the above types " +
					"or a record which consists of the above types")
			}
		case annotations[queryAnnotation] != nil:
			if queryName, ok := annotationName(annotations[queryAnnotation]); ok && strings.TrimSpace(queryName) != "" {
				params[i].name = encodeParamName(unescapeIdentifier(queryName))
			}
		default:
			unannotated = append(unannotated, i)
		}
	}
	if !hasPayload && isEntityBodyAccessor(accessor) {
		if bindErr := inferPayloadParam(tc, state.types, params, unannotated); bindErr != nil {
			return nil, bindErr
		}
	}
	for _, p := range params {
		if p.kind == queryParam && queryScalarKind(tc, state.types, p.ty) == invalidScalar {
			return nil, invalidSignature("invalid type of query param " + p.label + ": expected one of the string, " +
				"int, float, boolean, decimal, map<anydata> types or the array types of them")
		}
	}
	return params, nil
}

func isEntityBodyAccessor(accessor string) bool {
	switch accessor {
	case "get", "head", "options":
		return false
	}
	return true
}

// Request and Headers objects carry the top object type at runtime, so these parameters are
// recognised by their declared type instead.
func isExactType(tc semtypes.Context, want, paramTy semtypes.SemType) bool {
	return semtypes.IsSubtype(tc, paramTy, want) && semtypes.IsSubtype(tc, want, paramTy)
}

func annotationName(value values.AnnotationValue) (string, bool) {
	if m, ok := value.(*values.Map); ok {
		if name, ok := m.Get("name"); ok {
			s, ok := name.(string)
			return s, ok
		}
	}
	return "", false
}

// unescapeIdentifier mirrors jBallerina's HttpUtil.unescapeValue. Parameter and record field
// names reach native code with their quoted-identifier escapes intact; the leading quote of a
// quoted identifier is already gone.
func unescapeIdentifier(name string) string {
	return strings.ReplaceAll(name, "\\", "")
}

// encodeParamName mirrors jBallerina's HttpUtil.encodeString: Java's URLEncoder, with a space
// as %20. Query keys are matched without being decoded, so the name is encoded instead.
func encodeParamName(name string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', strings.IndexByte(".-*_", c) >= 0:
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xf])
		}
	}
	return b.String()
}

// The compiler plugin's rules: the single unannotated structured parameter of an entity-body
// resource is the payload; two such parameters are ambiguous.
func inferPayloadParam(tc semtypes.Context, types *httpTypes, params []resourceParam, candidates []int) *bindingError {
	payload := -1
	for _, i := range candidates {
		switch structuredShape(tc, types, params[i].ty) {
		case structuredType:
			if payload >= 0 {
				return invalidSignature("ambiguous types for parameter " + params[payload].label + " and " +
					params[i].label + ". Use annotations to avoid ambiguity")
			}
			payload = i
		case mixedType:
			if payload < 0 {
				return invalidSignature("invalid union type for default payload param: " + params[i].label +
					". Use basic structured anydata types")
			}
		}
	}
	if payload >= 0 {
		params[payload].kind = payloadParam
	}
	return nil
}

type shape uint8

const (
	basicType shape = iota
	structuredType
	mixedType
)

func structuredShape(tc semtypes.Context, types *httpTypes, ty semtypes.SemType) shape {
	base := semtypes.Diff(ty, semtypes.Nil)
	if semtypes.IsSubtype(tc, base, types.byteArrTy) {
		return structuredType
	}
	if isBasicParamType(tc, base) {
		return basicType
	}
	structured := semtypes.Union(semtypes.Union(semtypes.Mapping, semtypes.Table), semtypes.Union(semtypes.List, semtypes.XML))
	if semtypes.IsSubtype(tc, base, structured) && semtypes.IsSubtype(tc, base, types.anydataTy) {
		return structuredType
	}
	if !semtypes.IsEmpty(tc, semtypes.Intersect(base, structured)) {
		return mixedType
	}
	return basicType
}

func isBasicParamType(tc semtypes.Context, ty semtypes.SemType) bool {
	if semtypes.IsSubtype(tc, ty, semtypes.List) {
		ty = semtypes.ListMemberTypeInnerVal(tc, ty, semtypes.Int)
	}
	kind := scalarKindOf(tc, semtypes.Diff(ty, semtypes.Nil))
	return kind != invalidScalar && kind != mapScalar
}

func scalarKindOf(tc semtypes.Context, ty semtypes.SemType) scalarKind {
	if semtypes.IsEmpty(tc, ty) {
		return invalidScalar
	}
	for _, k := range []struct {
		ty   semtypes.SemType
		kind scalarKind
	}{
		{semtypes.String, stringScalar},
		{semtypes.Int, intScalar},
		{semtypes.Float, floatScalar},
		{semtypes.Boolean, booleanScalar},
		{semtypes.Decimal, decimalScalar},
		{semtypes.Mapping, mapScalar},
	} {
		if semtypes.IsSubtype(tc, ty, k.ty) {
			return k.kind
		}
	}
	return invalidScalar
}

// elementType returns the element type of an array parameter, with nil removed from both the
// parameter and its elements; ok is false when the parameter is not an array.
func elementType(tc semtypes.Context, ty semtypes.SemType) (semtypes.SemType, bool) {
	base := semtypes.Diff(ty, semtypes.Nil)
	if semtypes.IsEmpty(tc, base) || !semtypes.IsSubtype(tc, base, semtypes.List) {
		return semtypes.SemType{}, false
	}
	return semtypes.Diff(semtypes.ListMemberTypeInnerVal(tc, base, semtypes.Int), semtypes.Nil), true
}

func queryScalarKind(tc semtypes.Context, types *httpTypes, ty semtypes.SemType) scalarKind {
	if elem, isArray := elementType(tc, ty); isArray {
		ty = elem
	}
	kind := scalarKindOf(tc, semtypes.Diff(ty, semtypes.Nil))
	if kind == mapScalar && !semtypes.IsSubtype(tc, ty, types.anydataTy) {
		return invalidScalar
	}
	return kind
}

func headerScalarKind(tc semtypes.Context, ty semtypes.SemType) scalarKind {
	if elem, isArray := elementType(tc, ty); isArray {
		ty = elem
	}
	kind := scalarKindOf(tc, semtypes.Diff(ty, semtypes.Nil))
	if kind == mapScalar {
		return invalidScalar
	}
	return kind
}

// headerRecordType returns the record a header parameter binds into, or nil when the
// parameter is not a record.
func headerRecordType(tc semtypes.Context, ty semtypes.SemType) *semtypes.MappingAtomicType {
	base := semtypes.Diff(ty, semtypes.Nil)
	if semtypes.IsEmpty(tc, base) || !semtypes.IsSubtype(tc, base, semtypes.Mapping) {
		return nil
	}
	atomic := semtypes.ToMappingAtomicType(tc, base)
	if atomic == nil || len(atomic.FieldNames()) == 0 {
		return nil
	}
	return atomic
}

func isValidHeaderType(tc semtypes.Context, ty semtypes.SemType) bool {
	if record := headerRecordType(tc, ty); record != nil {
		for _, field := range record.FieldNames() {
			if headerScalarKind(tc, record.FieldInnerVal(field)) == invalidScalar {
				return false
			}
		}
		return true
	}
	return headerScalarKind(tc, ty) != invalidScalar
}

// The body is read up front when a payload parameter needs it, so a Request parameter can
// still read it afterwards; otherwise a Request parameter takes over the body as is. A Headers
// parameter wraps the same Request object, built without a body when nothing else needs one.
func bindParams(ctx *extern.Context, state *listenerState, params []resourceParam, r *http.Request) ([]values.BalValue, *bindingError) {
	types := state.types
	hasPayload, hasRequest, hasHeaders := false, false, false
	for _, p := range params {
		hasPayload = hasPayload || p.kind == payloadParam
		hasRequest = hasRequest || p.kind == requestParam
		hasHeaders = hasHeaders || p.kind == headersParam
	}
	var body []byte
	var req *values.Object
	switch {
	case hasPayload:
		data, err := readBody(r)
		if err != nil {
			return nil, badRequest("failed to read request body: " + err.Error())
		}
		body = data
		if hasRequest || hasHeaders {
			req = buildRequest(r.Method, r.RequestURI, r.Proto, r.Header, nil, 0, r.URL.RawQuery, body)
		}
	case hasRequest:
		built, err := buildRequestFromHTTP(r)
		if err != nil {
			return nil, badRequest("failed to read request body: " + err.Error())
		}
		req = built
	default:
		closeBody(r)
		if hasHeaders {
			req = buildRequest(r.Method, r.RequestURI, r.Proto, r.Header, nil, 0, r.URL.RawQuery, nil)
		}
	}
	args := make([]values.BalValue, len(params))
	var query map[string][]string
	for i, p := range params {
		var bindErr *bindingError
		switch p.kind {
		case requestParam:
			args[i] = req
		case headersParam:
			headers, err := ctx.InvokeFunction(state.createHeaders, []values.BalValue{req})
			if err != nil {
				return nil, invalidSignature(err.Error())
			}
			args[i] = headers
		case queryParam:
			if query == nil {
				parsed, err := parseQueryParams(r.URL.RawQuery)
				if err != nil {
					return nil, &bindingError{status: http.StatusInternalServerError, message: err.Error()}
				}
				query = parsed
			}
			args[i], bindErr = bindQueryParam(ctx, types, p, query)
		case headerParam:
			args[i], bindErr = bindHeaderParam(ctx, types, p, r.Header)
		case payloadParam:
			args[i], bindErr = bindPayloadParam(ctx, types, p.ty, r.Header.Get("Content-Type"), body)
		}
		if bindErr != nil {
			return nil, bindErr
		}
	}
	return args, nil
}

func readBody(r *http.Request) ([]byte, error) {
	defer closeBody(r)
	return io.ReadAll(r.Body)
}

// parseQueryParams follows jBallerina's URIUtil.populateQueryParamMap: values are split on
// commas before being URL-decoded, keys are left encoded, and a key without '=' is present
// with no value (a nil slice). A malformed escape fails the whole request, as Java's
// URLDecoder exception does.
func parseQueryParams(rawQuery string) (map[string][]string, error) {
	params := make(map[string][]string)
	if rawQuery == "" {
		return params, nil
	}
	for _, pair := range strings.Split(rawQuery, "&") {
		name, value, hasValue := strings.Cut(pair, "=")
		if !hasValue {
			if _, seen := params[pair]; !seen {
				params[pair] = nil
			}
			continue
		}
		name = strings.TrimSpace(name)
		var vals []string
		seen := make(map[string]struct{})
		for _, part := range strings.Split(strings.TrimSpace(value), ",") {
			decoded, err := url.QueryUnescape(part)
			if err != nil {
				return nil, err
			}
			if _, dup := seen[decoded]; !dup {
				seen[decoded] = struct{}{}
				vals = append(vals, decoded)
			}
		}
		params[name] = append(params[name], vals...)
	}
	return params, nil
}

// Missing and value-less parameters both bind to () for a nilable type, matching jBallerina's
// default treatNilableAsOptional. Defaultable parameters cannot be honoured: the runtime does
// not expose a parameter's default value to native code.
func bindQueryParam(ctx *extern.Context, types *httpTypes, p resourceParam, query map[string][]string) (values.BalValue, *bindingError) {
	tc := ctx.TypeCtx()
	vals := query[p.name]
	if vals == nil {
		if admits(tc, p.ty, semtypes.Nil) {
			return nil, nil
		}
		return nil, badRequest("no query param value found for '" + p.name + "'")
	}
	castErr := badRequest("error in casting query param : '" + p.name + "'")
	kind := queryScalarKind(tc, types, p.ty)
	parsed, ok := parseParamValues(ctx, types, kind, p.ty, vals)
	if !ok {
		return nil, castErr
	}
	bound, err := values.CloneWithType(tc, parsed, p.ty)
	if err != nil {
		return nil, castErr
	}
	return bound, nil
}

func parseParamValues(ctx *extern.Context, types *httpTypes, kind scalarKind, ty semtypes.SemType, vals []string) (values.BalValue, bool) {
	if _, isArray := elementType(ctx.TypeCtx(), ty); !isArray {
		return parseScalar(ctx, types, kind, vals[0])
	}
	items := make([]values.BalValue, len(vals))
	for i, v := range vals {
		item, ok := parseScalar(ctx, types, kind, v)
		if !ok {
			return nil, false
		}
		items[i] = item
	}
	return newTypedListValue(ctx.TypeEnv(), types.jsonListTy, items), true
}

// parseScalar mirrors jBallerina's ParamUtils.castParam, which parses with Java's number and
// boolean parsers.
func parseScalar(ctx *extern.Context, types *httpTypes, kind scalarKind, s string) (values.BalValue, bool) {
	switch kind {
	case intScalar:
		n, err := strconv.ParseInt(s, 10, 64)
		return n, err == nil
	case floatScalar:
		return parseJavaDouble(s)
	case booleanScalar:
		return strings.EqualFold(s, "true"), true
	case decimalScalar:
		d, err := decimal.FromString(s)
		return d, err == nil
	case mapScalar:
		v, jsonErr := decodeJSONBody(ctx, types, []byte(s))
		return v, jsonErr == nil
	default:
		return s, true
	}
}

// Java's Double.parseDouble trims surrounding whitespace, accepts a trailing float or double
// type suffix, and spells the specials only as "NaN" and "Infinity"; Go's parser also accepts
// "Inf", any letter case for the specials, and underscores.
func parseJavaDouble(s string) (values.BalValue, bool) {
	s = strings.TrimSpace(s)
	switch strings.TrimLeft(s, "+-") {
	case "NaN", "Infinity":
		f, err := strconv.ParseFloat(strings.Replace(s, "Infinity", "Inf", 1), 64)
		return f, err == nil
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "inf") || strings.Contains(lower, "nan") || strings.Contains(s, "_") {
		return nil, false
	}
	if n := len(s); n > 1 && strings.ContainsRune("dDfF", rune(s[n-1])) {
		s = s[:n-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func bindHeaderParam(ctx *extern.Context, types *httpTypes, p resourceParam, header http.Header) (values.BalValue, *bindingError) {
	tc := ctx.TypeCtx()
	if record := headerRecordType(tc, p.ty); record != nil {
		return bindHeaderRecord(ctx, types, p, record, header)
	}
	vals := header.Values(p.name)
	if len(vals) == 0 || (len(vals) == 1 && vals[0] == "") {
		if admits(tc, p.ty, semtypes.Nil) {
			return nil, nil
		}
		return nil, badRequest("no header value found for '" + p.name + "'")
	}
	bindErr := badRequest("header binding failed for parameter: '" + p.name + "'")
	parsed, ok := parseParamValues(ctx, types, headerScalarKind(tc, p.ty), p.ty, vals)
	if !ok {
		return nil, bindErr
	}
	bound, err := values.CloneWithType(tc, parsed, p.ty)
	if err != nil {
		return nil, bindErr
	}
	return bound, nil
}

// A header record's field names are its header names. jBallerina also honours an
// @http:Header {name} on a field, but field annotations are not reachable from a parameter's
// type here.
func bindHeaderRecord(ctx *extern.Context, types *httpTypes, p resourceParam, record *semtypes.MappingAtomicType,
	header http.Header) (values.BalValue, *bindingError) {
	tc := ctx.TypeCtx()
	fields := newMappingValue()
	for _, field := range record.FieldNames() {
		fieldTy := record.FieldInnerVal(field)
		key := unescapeIdentifier(field)
		vals := header.Values(key)
		if len(vals) == 0 || (len(vals) == 1 && vals[0] == "") {
			switch {
			case admits(tc, fieldTy, semtypes.Nil):
				fields.Put(tc, field, nil)
				continue
			case admits(tc, p.ty, semtypes.Nil):
				return nil, nil
			default:
				return nil, badRequest("no header value found for '" + key + "'")
			}
		}
		parsed, ok := parseParamValues(ctx, types, headerScalarKind(tc, fieldTy), fieldTy, vals)
		if !ok {
			return nil, badRequest("header binding failed for parameter: '" + key + "'")
		}
		fields.Put(tc, field, parsed)
	}
	bound, err := values.CloneWithType(tc, fields, p.ty)
	if err != nil {
		return nil, badRequest("header binding failed for parameter: '" + p.name + "'")
	}
	return bound, nil
}

// bindPayloadParam picks a builder from the Content-Type as jBallerina's
// AbstractPayloadBuilder.getBuilder does, falling back to the parameter type when the media
// type is absent or unknown.
func bindPayloadParam(ctx *extern.Context, types *httpTypes, ty semtypes.SemType, rawContentType string,
	body []byte) (values.BalValue, *bindingError) {
	tc := ctx.TypeCtx()
	if len(body) == 0 && admits(tc, ty, semtypes.Nil) {
		return nil, nil
	}
	contentType := strings.ToLower(strings.TrimSpace(rawContentType))
	if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	var (
		value values.BalValue
		msg   string
	)
	switch {
	case contentType == "":
		value, msg = payloadFromType(ctx, types, ty, body)
	case xmlContentType.MatchString(contentType):
		value, msg = xmlPayload(ctx, ty, body)
	case textContentType.MatchString(contentType), urlEncodedContentType.MatchString(contentType):
		value, msg = stringPayload(ctx, types, ty, body)
	case octetStreamContentType.MatchString(contentType):
		value, msg = binaryPayload(ctx, types, ty, body)
	case jsonContentType.MatchString(contentType):
		value, msg = jsonPayload(ctx, types, ty, body)
	default:
		value, msg = payloadFromType(ctx, types, ty, body)
	}
	if msg != "" {
		return nil, badRequest("data binding failed: " + msg)
	}
	return value, nil
}

func payloadFromType(ctx *extern.Context, types *httpTypes, ty semtypes.SemType, body []byte) (values.BalValue, string) {
	tc := ctx.TypeCtx()
	switch {
	case semtypes.IsSubtype(tc, ty, semtypes.String):
		return stringPayload(ctx, types, ty, body)
	case semtypes.IsSubtype(tc, ty, semtypes.XML):
		return xmlPayload(ctx, ty, body)
	case semtypes.IsSubtype(tc, ty, types.byteArrTy):
		return binaryPayload(ctx, types, ty, body)
	default:
		return jsonPayload(ctx, types, ty, body)
	}
}

func incompatiblePayload(tc semtypes.Context, ty semtypes.SemType) string {
	return "incompatible type found: '" + semtypes.ToString(tc, ty) + "'"
}

func xmlPayload(ctx *extern.Context, ty semtypes.SemType, body []byte) (values.BalValue, string) {
	tc := ctx.TypeCtx()
	if semtypes.IsEmpty(tc, semtypes.Intersect(ty, semtypes.XML)) {
		return nil, incompatiblePayload(tc, ty)
	}
	if len(body) == 0 {
		return nil, "Empty xml payload"
	}
	parsed, err := values.ParseAsXMLValue(tc, values.FromBytes(body), values.XMLLenientMode)
	if err != nil {
		return nil, "failed to parse xml: " + err.Error()
	}
	return convertPayload(tc, parsed, semtypes.XML, ty)
}

// jBallerina tries each union member in declaration order; that order is not available
// from a semtype, so string is preferred over byte[] over a form map.
func stringPayload(ctx *extern.Context, types *httpTypes, ty semtypes.SemType, body []byte) (values.BalValue, string) {
	tc := ctx.TypeCtx()
	base := semtypes.Diff(ty, semtypes.Nil)
	intersects := func(t semtypes.SemType) bool { return !semtypes.IsEmpty(tc, semtypes.Intersect(base, t)) }
	asString, asBytes, asForm := intersects(semtypes.String), intersects(types.byteArrTy), intersects(types.mapStringTy)
	if !asString && !asBytes && !asForm {
		return nil, incompatiblePayload(tc, ty)
	}
	if len(body) == 0 {
		return nil, "String payload is null"
	}
	switch {
	case asString:
		return convertPayload(tc, string(body), semtypes.String, ty)
	case asBytes:
		return convertPayload(tc, byteArrayValue(ctx, types, body), types.byteArrTy, ty)
	default:
		form, msg := formPayload(ctx, types, body)
		if msg != "" {
			return nil, msg
		}
		return convertPayload(tc, form, types.mapStringTy, ty)
	}
}

// Repeated keys keep the last value, as jBallerina's UrlEncodedStringToMapConverter does.
func formPayload(ctx *extern.Context, types *httpTypes, body []byte) (values.BalValue, string) {
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err.Error()
	}
	tc := ctx.TypeCtx()
	out := values.NewMap(types.mapStringTy, semtypes.ToMappingAtomicType(tc, types.mapStringTy), false, nil)
	for key, vals := range parsed {
		out.Put(tc, key, vals[len(vals)-1])
	}
	return out, ""
}

func binaryPayload(ctx *extern.Context, types *httpTypes, ty semtypes.SemType, body []byte) (values.BalValue, string) {
	tc := ctx.TypeCtx()
	if semtypes.IsEmpty(tc, semtypes.Intersect(ty, types.byteArrTy)) {
		return nil, incompatiblePayload(tc, ty)
	}
	return convertPayload(tc, byteArrayValue(ctx, types, body), types.byteArrTy, ty)
}

func jsonPayload(ctx *extern.Context, types *httpTypes, ty semtypes.SemType, body []byte) (values.BalValue, string) {
	tc := ctx.TypeCtx()
	if len(body) == 0 {
		return nil, "empty JSON document"
	}
	payload, jsonErr := decodeJSONBody(ctx, types, body)
	if jsonErr != nil {
		return nil, jsonErr.Message
	}
	bound, convErr := values.CloneWithType(tc, payload, ty)
	if convErr != nil {
		return nil, convErr.Message
	}
	return bound, ""
}

// A target the builder's type already fits needs no conversion; a narrower one (an enum, a
// closed record, readonly) must be converted so the resource never holds a value outside its
// declared type.
func convertPayload(tc semtypes.Context, value values.BalValue, builderTy, ty semtypes.SemType) (values.BalValue, string) {
	if admits(tc, ty, builderTy) {
		return value, ""
	}
	bound, err := values.CloneWithType(tc, value, ty)
	if err != nil {
		return nil, err.Message
	}
	return bound, ""
}
