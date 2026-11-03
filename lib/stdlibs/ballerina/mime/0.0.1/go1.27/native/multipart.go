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
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

//nolint:staticcheck // error text mirrors jBallerina's mimepull message verbatim
var errMissingStartBoundary = errors.New("Missing start boundary")

type decodedPart struct {
	header textproto.MIMEHeader
	body   []byte
}

// entityHeaderValue reads the first value of a header from an Entity's own headerMap
// field (the private `.bal`-declared state `Entity.setHeader`/`getHeader` operate on).
func entityHeaderValue(obj *values.Object, headerName string) (string, bool) {
	hmVal, ok := obj.Get("headerMap")
	if !ok {
		return "", false
	}
	hm, ok := hmVal.(*values.Map)
	if !ok {
		return "", false
	}
	v, ok := hm.Get(strings.ToLower(headerName))
	if !ok {
		return "", false
	}
	list, ok := v.(*values.List)
	if !ok || list.Len() == 0 {
		return "", false
	}
	s, ok := list.Get(0).(string)
	return s, ok
}

// MultipartBoundary parses a Content-Type header value and reports whether it names a
// composite (multipart/* or message/*, per RFC 2046) media type, along with its
// boundary parameter. message/* always reports an empty boundary regardless of the
// Content-Type params: it isn't boundary-delimited, so a "boundary" param present on
// one must never be handed to multipart.NewReader as if it were.
func MultipartBoundary(contentType string) (baseType, boundary string, isComposite bool) {
	if contentType == "" {
		return "", "", false
	}
	mt, err := parseMediaType(contentType)
	if err != nil {
		return contentType, "", false
	}
	baseType = mt.primaryType + "/" + mt.subType
	switch mt.primaryType {
	case "multipart":
		return baseType, mt.param("boundary"), true
	case "message":
		return baseType, "", true
	default:
		return baseType, "", false
	}
}

// DecodeMultipart splits a raw multipart body into per-part Entity values, copying
// every part header verbatim and transfer-decoding each part body. A part without a
// Content-Type header is left without one, as in jBallerina.
//
// A missing boundary is an error, and so is non-empty data holding no part, which
// jBallerina reports as a missing start boundary. Empty data decodes to no parts; callers
// decide whether an empty body is an error.
func DecodeMultipart(ctx *extern.Context, data []byte, boundary string) (*values.List, error) {
	if boundary == "" {
		return nil, fmt.Errorf("no boundary parameter found in Content-Type")
	}
	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	var decoded []decodedPart
	for {
		part, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(transferDecoder(part, part.Header.Get("Content-Transfer-Encoding")))
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, decodedPart{header: part.Header, body: body})
	}
	if len(decoded) == 0 && len(data) > 0 {
		return nil, errMissingStartBoundary
	}
	parts, err := newBodyParts(ctx, len(decoded))
	if err != nil {
		return nil, err
	}
	for i, d := range decoded {
		partObj := parts.Get(i).(*values.Object)
		if err := addEntityHeaders(ctx, partObj, d.header); err != nil {
			return nil, err
		}
		setEntityBody(partObj, &entityBody{kind: bodyBytes, bytes: d.body, fromChannel: true})
	}
	return parts, nil
}

// transferDecoder decodes a part body per its Content-Transfer-Encoding, as jBallerina
// does, while the header itself stays on the part. Unknown encodings pass through.
func transferDecoder(r io.Reader, encoding string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, r)
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	default:
		return r
	}
}

// newBodyParts calls the private .bal newBodyParts helper so the parts and the list
// holding them carry their real Entity/Entity[] types.
func newBodyParts(ctx *extern.Context, count int) (*values.List, error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, "newBodyParts")
	if !ok {
		return nil, fmt.Errorf("mime: internal helper function newBodyParts not found")
	}
	result, err := ctx.InvokeFunction(handle, []values.BalValue{int64(count)})
	if err != nil {
		return nil, err
	}
	parts, ok := result.(*values.List)
	if !ok {
		return nil, fmt.Errorf("mime: newBodyParts returned an unexpected value")
	}
	return parts, nil
}

// addEntityHeaders adds each part header through Entity.addHeader, in sorted name
// order since textproto.MIMEHeader does not keep the wire order.
func addEntityHeaders(ctx *extern.Context, obj *values.Object, header textproto.MIMEHeader) error {
	handle, ok := ctx.LookupObjectMethod(obj, "addHeader")
	if !ok {
		return fmt.Errorf("mime: Entity.addHeader not found")
	}
	for _, name := range slices.Sorted(maps.Keys(header)) {
		for _, value := range header[name] {
			if _, err := ctx.InvokeMethod(handle, []values.BalValue{obj, name, value}); err != nil {
				return err
			}
		}
	}
	return nil
}

// EncodeMultipart serializes body parts into multipart wire bytes, generating a boundary when
// none is given, and returns the boundary actually used. jBallerina does this in its HTTP
// transport rather than through mime's public API.
func EncodeMultipart(ctx *extern.Context, parts []*values.Object, boundary string) (data []byte, usedBoundary string, err error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if boundary != "" {
		if err := w.SetBoundary(boundary); err != nil {
			return nil, "", err
		}
	}
	for _, part := range parts {
		header, partData, err := encodePart(ctx, part)
		if err != nil {
			return nil, "", err
		}
		pw, err := w.CreatePart(header)
		if err != nil {
			return nil, "", err
		}
		if _, err := pw.Write(partData); err != nil {
			return nil, "", err
		}
	}
	usedBoundary = w.Boundary()
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), usedBoundary, nil
}

// encodePart returns a part's wire header and body. A part whose body is itself body parts
// is encoded recursively, and a generated boundary is added to the header written for it.
func encodePart(ctx *extern.Context, part *values.Object) (textproto.MIMEHeader, []byte, error) {
	header := entityHeaders(part)
	body := getEntityBody(part)
	if body == nil || body.kind != bodyParts {
		data, err := bytesForBody(ctx, part)
		return header, data, err
	}
	nested := make([]*values.Object, body.parts.Len())
	for i := range nested {
		nested[i], _ = body.parts.Get(i).(*values.Object)
	}
	contentType, found := entityHeaderValue(part, "content-type")
	if !found {
		contentType = "multipart/form-data"
	}
	_, boundary, _ := MultipartBoundary(contentType)
	data, usedBoundary, err := EncodeMultipart(ctx, nested, boundary)
	if err != nil {
		return nil, nil, err
	}
	if boundary == "" {
		setHeaderValue(header, "Content-Type", contentType+"; boundary="+usedBoundary)
	}
	return header, data, nil
}

// setHeaderValue replaces name's value in header, keeping the casing of an existing key.
func setHeaderValue(header textproto.MIMEHeader, name, value string) {
	for key := range header {
		if strings.EqualFold(key, name) {
			header[key] = []string{value}
			return
		}
	}
	header[name] = []string{value}
}

// entityHeaders reads an Entity's headerNames/headerMap fields back into a MIME header,
// keeping each header name's original casing.
func entityHeaders(obj *values.Object) textproto.MIMEHeader {
	result := textproto.MIMEHeader{}
	namesVal, _ := obj.Get("headerNames")
	names, ok := namesVal.(*values.List)
	if !ok {
		return result
	}
	hmVal, _ := obj.Get("headerMap")
	hm, ok := hmVal.(*values.Map)
	if !ok {
		return result
	}
	for i := range names.Len() {
		name, _ := names.Get(i).(string)
		v, ok := hm.Get(strings.ToLower(name))
		if !ok {
			continue
		}
		list, ok := v.(*values.List)
		if !ok {
			continue
		}
		vals := make([]string, list.Len())
		for j := range list.Len() {
			vals[j], _ = list.Get(j).(string)
		}
		result[name] = vals
	}
	return result
}

func setBodyPartsExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		parts, _ := args[1].(*values.List)
		setEntityBody(obj, &entityBody{kind: bodyParts, parts: parts})
		return nil, nil
	}
}

func getBodyPartsExtern() extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		obj, _ := args[0].(*values.Object)
		body := getEntityBody(obj)
		if body != nil && body.kind == bodyParts {
			return body.parts, nil
		}
		contentType, found := entityHeaderValue(obj, "content-type")
		baseType, boundary, isComposite := MultipartBoundary(contentType)
		if !isComposite {
			if !found {
				// jBallerina concatenates the absent Java header value as "null".
				baseType = "null"
			}
			return mimeError("ParserError", "Entity body is not a type of composite media type. "+
				"Received content-type : "+baseType), nil
		}
		if strings.HasPrefix(strings.ToLower(baseType), "message/") {
			return mimeError("ParserError", "message/* body part decoding is not yet supported. "+
				"Received content-type : "+baseType), nil
		}
		body, err := materializeBody(ctx, obj)
		if err != nil {
			return mimeError("ParserError", "Error occurred while extracting body parts from entity: "+err.Error()), nil
		}
		if body == nil || body.kind != bodyBytes {
			return newBodyParts(ctx, 0)
		}
		parts, err := DecodeMultipart(ctx, body.bytes, boundary)
		if err == nil && body.fromChannel && parts.Len() == 0 {
			err = errMissingStartBoundary
		}
		if err != nil {
			return mimeError("ParserError", "Error occurred while extracting body parts from entity: "+err.Error()), nil
		}
		setEntityBody(obj, &entityBody{kind: bodyParts, parts: parts})
		return parts, nil
	}
}
