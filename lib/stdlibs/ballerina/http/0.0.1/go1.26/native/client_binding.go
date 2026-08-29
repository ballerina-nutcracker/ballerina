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

// Client-side response data binding, mirroring jBallerina's processResponse /
// performDataBinding chain.

package native

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// Copied verbatim from jBallerina's AbstractPayloadBuilder, so the same media type picks
// the same builder.
var (
	jsonContentType        = regexp.MustCompile(`^(application|text)/(.*[.+-]|)json$`)
	xmlContentType         = regexp.MustCompile(`^(application|text)/(.*[.+-]|)xml$`)
	textContentType        = regexp.MustCompile(`^(text)/(.*[.+-]|)plain$`)
	octetStreamContentType = regexp.MustCompile(`^(application)/(.*[.+-]|)octet-stream$`)
	urlEncodedContentType  = regexp.MustCompile(`^(application)/(.*[.+-]|)x-www-form-urlencoded$`)
)

// bindResponse turns a received response into the value the call site asked for: a target
// admitting http:Response yields the response untouched, a 4xx/5xx status becomes an error
// carrying the status detail, and anything else is deserialised into the target type.
func bindResponse(ctx *extern.Context, types *httpTypes, resp *values.Object, targetArg values.BalValue) values.BalValue {
	tc := ctx.TypeCtx()
	target := targetArg.(*values.TypeDesc).Type
	if admitsResponse(tc, target) {
		return resp
	}
	statusCode := responseStatusCode(resp)
	if statusCode >= 400 && statusCode <= 599 {
		return statusCodeError(ctx, types, resp, statusCode)
	}
	result := performDataBinding(ctx, types, resp, target)
	if _, failed := result.(*values.Error); failed {
		// A mismatch error (e.g. incompatibleTargetError) can return
		// before the body is ever read, leaving the underlying stream/connection open.
		_, _ = responseBody(resp)
	}
	return result
}

// Responses are stamped with the top object type, so any object member counts — matching
// jBallerina's hasHttpResponseType, true for an OBJECT tag anywhere in the union.
func admitsResponse(tc semtypes.Context, target semtypes.SemType) bool {
	return !semtypes.IsEmpty(tc, semtypes.Intersect(target, semtypes.Object))
}

func responseStatusCode(resp *values.Object) int {
	v, _ := resp.Get("statusCode")
	code, _ := v.(int64)
	return int(code)
}

// Mirrors jBallerina's createResponseError. ClientRequestError and RemoteServerError are not
// declared here, so the distinction survives only in the error's type name.
func statusCodeError(ctx *extern.Context, types *httpTypes, resp *values.Object, statusCode int) *values.Error {
	body, extractErr := statusErrorPayload(ctx, types, resp)
	if extractErr != nil {
		return values.NewError(semtypes.Error, "http:ApplicationResponseError creation failed: "+
			strconv.Itoa(statusCode)+" response payload extraction failed", extractErr,
			"PayloadBindingClientError", nil)
	}
	tc := ctx.TypeCtx()
	detail := newMappingValue()
	detail.Put(tc, "statusCode", int64(statusCode))
	detail.Put(tc, "headers", copyResponseHeaders(tc, resp))
	detail.Put(tc, "body", body)
	typeName := "ClientRequestError"
	if statusCode >= 500 {
		typeName = "RemoteServerError"
	}
	return values.NewError(semtypes.Error, reasonPhrase(statusCode), nil, typeName, detail)
}

// Extracts the error body the way jBallerina's getPayload does, matching the media type with
// the regexes rather than an exact primary/sub-type match.
func statusErrorPayload(ctx *extern.Context, types *httpTypes, resp *values.Object) (values.BalValue, *values.Error) {
	body, err := responseBody(resp)
	if err != nil {
		return nil, values.NewErrorWithMessage(err.Error())
	}
	// Short-circuited so a 4xx/5xx sent with a JSON content type and no body keeps its reason
	// phrase instead of failing on the empty document.
	if len(body) == 0 {
		return nil, nil
	}
	contentType := baseContentType(resp)
	switch {
	case jsonContentType.MatchString(contentType):
		return decodeJSONBody(ctx, types, body)
	case xmlContentType.MatchString(contentType):
		return decodeXMLBody(ctx, body, "response")
	case octetStreamContentType.MatchString(contentType):
		return byteArrayValue(ctx, types, body), nil
	default:
		return string(body), nil
	}
}

// The PAL transport reports only the status code, so the phrase sent on the wire is not
// available. Codes outside the IANA registry (nginx's 499, say) have none registered.
func reasonPhrase(statusCode int) string {
	if phrase := http.StatusText(statusCode); phrase != "" {
		return phrase
	}
	return "status code " + strconv.Itoa(statusCode)
}

func copyResponseHeaders(tc semtypes.Context, resp *values.Object) *values.Map {
	src := responseHeaders(resp)
	out := newMappingValue()
	for _, k := range src.Keys() {
		v, _ := src.Get(k)
		out.Put(tc, k, v)
	}
	return out
}

// Picks a builder from the Content-Type as jBallerina does, falling back to the target type
// when the media type is absent or unknown.
func performDataBinding(ctx *extern.Context, types *httpTypes, resp *values.Object, target semtypes.SemType) values.BalValue {
	tc := ctx.TypeCtx()
	// Read before selecting a builder so a read failure (for example exceeding
	// responseLimits.maxEntityBodySize) surfaces as an error instead of being silently
	// discarded, and so the underlying stream is drained and closed.
	body, err := responseBody(resp)
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	// jBallerina hands back a non-empty body here even though () was asked for; returning ()
	// keeps the value within the requested type.
	if semtypes.IsSubtype(tc, target, semtypes.Nil) {
		return nil
	}
	// The spec's "absent payload binds to ()" rule, judged from the raw bytes and applied
	// before the Content-Type picks a builder. Deciding it here rather than inside a builder
	// is what makes it hold for every nilable target, not only the ones whose builder the
	// media type happens to select — `int?` against an empty text/plain body binds to ()
	// instead of failing as an incompatible target.
	if len(body) == 0 && admits(tc, target, semtypes.Nil) {
		return nil
	}
	contentType := baseContentType(resp)
	switch {
	case contentType == "":
		return builderFromType(ctx, types, resp, target)
	case xmlContentType.MatchString(contentType):
		return xmlPayloadBuilder(ctx, resp, target, contentType)
	case textContentType.MatchString(contentType):
		return textPayloadBuilder(ctx, types, resp, target, contentType)
	case urlEncodedContentType.MatchString(contentType):
		return formPayloadBuilder(ctx, types, resp, target, contentType)
	case octetStreamContentType.MatchString(contentType):
		return blobPayloadBuilder(ctx, types, resp, target, contentType)
	case jsonContentType.MatchString(contentType):
		return jsonPayloadBuilder(ctx, types, resp, target)
	default:
		return builderFromType(ctx, types, resp, target)
	}
}

// Used when the response carries no usable Content-Type.
func builderFromType(ctx *extern.Context, types *httpTypes, resp *values.Object, target semtypes.SemType) values.BalValue {
	tc := ctx.TypeCtx()
	switch {
	case narrowsTo(tc, target, semtypes.String):
		return bindAtTarget(tc, textValue(resp), semtypes.String, target)
	case narrowsTo(tc, target, semtypes.XML):
		return bindAtTarget(tc, xmlValue(ctx, resp), semtypes.XML, target)
	case narrowsTo(tc, target, types.byteArrTy):
		return bindAtTarget(tc, binaryValue(ctx, types, resp), types.byteArrTy, target)
	default:
		return jsonPayloadBuilder(ctx, types, resp, target)
	}
}

func xmlPayloadBuilder(ctx *extern.Context, resp *values.Object,
	target semtypes.SemType, contentType string) values.BalValue {
	tc := ctx.TypeCtx()
	// A union member narrower than xml (e.g. xml:Element) only intersects XML rather than
	// admitting or narrowing to it, so the target is accepted whenever some member could
	// hold an xml value; bindAtTarget then converts to (or rejects) the exact member.
	if semtypes.IsEmpty(tc, semtypes.Intersect(target, semtypes.XML)) {
		return incompatibleTargetError(tc, target, contentType)
	}
	return bindAtTarget(tc, xmlValue(ctx, resp), semtypes.XML, target)
}

func textPayloadBuilder(ctx *extern.Context, types *httpTypes, resp *values.Object,
	target semtypes.SemType, contentType string) values.BalValue {
	tc := ctx.TypeCtx()
	switch {
	case narrowsTo(tc, target, semtypes.String), admits(tc, target, semtypes.String):
		return bindAtTarget(tc, textValue(resp), semtypes.String, target)
	case narrowsTo(tc, target, types.byteArrTy), admits(tc, target, types.byteArrTy):
		return bindAtTarget(tc, binaryValue(ctx, types, resp), types.byteArrTy, target)
	default:
		return incompatibleTargetError(tc, target, contentType)
	}
}

func formPayloadBuilder(ctx *extern.Context, types *httpTypes, resp *values.Object,
	target semtypes.SemType, contentType string) values.BalValue {
	tc := ctx.TypeCtx()
	switch {
	case narrowsTo(tc, target, types.mapStringTy), admits(tc, target, types.mapStringTy):
		return bindAtTarget(tc, formDataValue(ctx, types, resp), types.mapStringTy, target)
	case narrowsTo(tc, target, semtypes.String), admits(tc, target, semtypes.String):
		return bindAtTarget(tc, textValue(resp), semtypes.String, target)
	default:
		return incompatibleTargetError(tc, target, contentType)
	}
}

func blobPayloadBuilder(ctx *extern.Context, types *httpTypes, resp *values.Object,
	target semtypes.SemType, contentType string) values.BalValue {
	tc := ctx.TypeCtx()
	switch {
	case narrowsTo(tc, target, types.byteArrTy), admits(tc, target, types.byteArrTy):
		return bindAtTarget(tc, binaryValue(ctx, types, resp), types.byteArrTy, target)
	default:
		return incompatibleTargetError(tc, target, contentType)
	}
}

// Reports whether target fits the type a builder produces, ignoring a nil member so that
// `Colour?` reaches the same builder as `Colour`. A bare `()` is handled before any builder.
func narrowsTo(tc semtypes.Context, target, builderTy semtypes.SemType) bool {
	bare := semtypes.Diff(target, semtypes.Nil)
	return !semtypes.IsEmpty(tc, bare) && semtypes.IsSubtype(tc, bare, builderTy)
}

// Turns a payload built at builderTy into the value target asks for. Targets narrower than
// builderTy (an enum, a closed record, a tuple) must be converted, or the call site ends up
// holding a value outside its declared type; a target builderTy already fits skips the clone.
func bindAtTarget(tc semtypes.Context, payload values.BalValue, builderTy, target semtypes.SemType) values.BalValue {
	if _, failed := payload.(*values.Error); failed {
		return payload
	}
	if admits(tc, target, builderTy) {
		return payload
	}
	bound, convErr := values.CloneWithType(tc, payload, target)
	if convErr != nil {
		return payloadBindingError(convErr.Message, convErr)
	}
	return bound
}

// Converts with the same routine as lang.value:fromJsonWithType. jBallerina also rejects a
// target that is neither http:Response nor anydata; TargetType admits nothing else, and a
// Response target has already returned, so that check has no counterpart here.
func jsonPayloadBuilder(ctx *extern.Context, types *httpTypes, resp *values.Object,
	target semtypes.SemType) values.BalValue {
	tc := ctx.TypeCtx()
	body, err := responseBody(resp)
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	payload, jsonErr := decodeJSONBody(ctx, types, body)
	if jsonErr != nil {
		return jsonErr
	}
	bound, convErr := values.CloneWithType(tc, payload, target)
	if convErr != nil {
		return payloadBindingError(convErr.Message, convErr)
	}
	return bound
}

// The prefix and type name match jBallerina's PayloadBindingClientError, though the distinct
// type itself is not declared here.
func payloadBindingError(message string, cause *values.Error) *values.Error {
	return values.NewError(semtypes.Error, "Payload binding failed: "+message, cause,
		"PayloadBindingClientError", nil)
}

// incompatibleTargetError is only reached from a builder the Content-Type selected, so the
// media type is always known.
func incompatibleTargetError(tc semtypes.Context, target semtypes.SemType, contentType string) *values.Error {
	message := "incompatible '" + semtypes.ToString(tc, target) + "' found for '" + contentType + "' mime type"
	return values.NewError(semtypes.Error, message, nil, "PayloadBindingClientError", nil)
}

// The check jBallerina spells as matchingType.
func admits(tc semtypes.Context, target, member semtypes.SemType) bool {
	return semtypes.IsSubtype(tc, member, target)
}

// responseContentType returns the raw Content-Type header, or "" when it is absent or
// malformed.
func responseContentType(resp *values.Object) string {
	v, ok := responseHeaders(resp).Get("content-type")
	if !ok {
		return ""
	}
	list, ok := v.(*values.List)
	if !ok || list.Len() == 0 {
		return ""
	}
	s, ok := list.Get(0).(string)
	if !ok {
		return ""
	}
	return s
}

// jBallerina matches its patterns against the base type only.
func baseContentType(resp *values.Object) string {
	raw := responseContentType(resp)
	if idx := strings.IndexByte(raw, ';'); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

func responseBody(resp *values.Object) ([]byte, error) {
	bodyVal, _ := resp.Get("body")
	return bodyVal.(*responseBodyHolder).materialize()
}

func textValue(resp *values.Object) values.BalValue {
	body, err := responseBody(resp)
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	return string(body)
}

func binaryValue(ctx *extern.Context, types *httpTypes, resp *values.Object) values.BalValue {
	body, err := responseBody(resp)
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	return byteArrayValue(ctx, types, body)
}

func byteArrayValue(ctx *extern.Context, types *httpTypes, body []byte) *values.List {
	items := make([]values.BalValue, len(body))
	for i, b := range body {
		items[i] = int64(b)
	}
	return newTypedListValue(ctx.TypeEnv(), types.byteArrTy, items)
}

// Repeated keys keep the last value, matching jBallerina's getFormDataMap.
func formDataValue(ctx *extern.Context, types *httpTypes, resp *values.Object) values.BalValue {
	body, err := responseBody(resp)
	if err != nil {
		return values.NewErrorWithMessage(err.Error())
	}
	parsed, parseErr := url.ParseQuery(string(body))
	if parseErr != nil {
		return payloadBindingError(parseErr.Error(), nil)
	}
	tc := ctx.TypeCtx()
	out := values.NewMap(types.mapStringTy, semtypes.ToMappingAtomicType(tc, types.mapStringTy), false, nil)
	for key, vals := range parsed {
		out.Put(tc, key, vals[len(vals)-1])
	}
	return out
}

// Both callers are reached only after performDataBinding has already called responseBody
// once (and returned its error, if any), so the cached, always-successful re-read here never
// needs its own error check.
func xmlValue(ctx *extern.Context, resp *values.Object) values.BalValue {
	body, _ := responseBody(resp)
	payload, xmlErr := decodeXMLBody(ctx, body, "response")
	if xmlErr != nil {
		return xmlErr
	}
	return payload
}

// Error names and messages come from jBallerina (http_commons.bal:367), though neither
// distinct type is declared here. The Go parser's wording differs from Java's, so it goes in
// the cause rather than the message callers assert on.
func decodeXMLBody(ctx *extern.Context, body []byte, source string) (values.BalValue, *values.Error) {
	if len(body) == 0 {
		return nil, values.NewError(semtypes.Error, "No content",
			values.NewErrorWithMessage("Empty xml payload"), "NoContentError", nil)
	}
	parsed, err := values.ParseAsXMLValue(ctx.TypeCtx(), values.FromBytes(body), values.XMLLenientMode)
	if err != nil {
		return nil, values.NewError(semtypes.Error,
			"Error occurred while retrieving the xml payload from the "+source,
			values.NewErrorWithMessage(err.Error()), "GenericClientError", nil)
	}
	return parsed, nil
}

func decodeJSONBody(ctx *extern.Context, types *httpTypes, body []byte) (values.BalValue, *values.Error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, values.NewErrorWithMessage("failed to parse JSON payload: " + err.Error())
	}
	// Decode only consumes the first JSON value; without this check, trailing data after a
	// well-formed value (a second document, or garbage) would be silently ignored.
	if err := dec.Decode(new(json.RawMessage)); err != io.EOF {
		message := "unexpected trailing data after JSON value"
		if err != nil {
			message = err.Error()
		}
		return nil, values.NewErrorWithMessage("failed to parse JSON payload: " + message)
	}
	return values.GoToBalValue(ctx.TypeCtx(), v, types.jsonListTy, types.jsonMapTy), nil
}
