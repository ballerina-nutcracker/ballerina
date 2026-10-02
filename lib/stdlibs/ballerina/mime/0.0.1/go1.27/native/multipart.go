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
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/textproto"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

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

// multipartBoundary parses a Content-Type header value and reports whether it names a
// composite (multipart/* or message/*, per RFC 2046) media type, along with its
// boundary parameter. message/* always reports an empty boundary regardless of the
// Content-Type params: it isn't boundary-delimited, so a "boundary" param present on
// one must never be handed to multipart.NewReader as if it were.
func multipartBoundary(contentType string) (baseType, boundary string, isComposite bool) {
	if contentType == "" {
		return "", "", false
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return contentType, "", false
	}
	primaryType := strings.ToLower(strings.SplitN(mediaType, "/", 2)[0])
	switch primaryType {
	case "multipart":
		return mediaType, params["boundary"], true
	case "message":
		return mediaType, "", true
	default:
		return mediaType, "", false
	}
}

// decodeMultipart splits a raw multipart body into per-part Entity values, defaulting
// an absent per-part Content-Type to "text/plain" (matching jBallerina's underlying
// MIME library default) and copying every part header verbatim.
//
// A missing boundary is a ParserError here; jBallerina instead silently returns an
// empty Entity[] in this case (it never attempts to decode a manually-set byte array
// as multipart at all — only an inbound request/response body is eligible there).
func decodeMultipart(ctx *extern.Context, data []byte, boundary string) (*values.List, error) {
	if boundary == "" {
		return nil, fmt.Errorf("no boundary parameter found in Content-Type")
	}
	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	var decoded []decodedPart
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.Header.Get("Content-Type") == "" {
			part.Header.Set("Content-Type", "text/plain")
		}
		body, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, decodedPart{header: part.Header, body: body})
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
		setEntityBody(partObj, &entityBody{kind: bodyBytes, bytes: d.body})
	}
	return parts, nil
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

func registerMultipartExterns(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetBodyParts",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			parts, ok := args[1].(*values.List)
			if !ok {
				return nil, fmt.Errorf("second argument must be an Entity array")
			}
			setEntityBody(obj, &entityBody{kind: bodyParts, parts: parts})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetBodyParts",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			body := getEntityBody(obj)
			if body != nil && body.kind == bodyParts {
				return body.parts, nil
			}
			contentType, _ := entityHeaderValue(obj, "content-type")
			baseType, boundary, isComposite := multipartBoundary(contentType)
			if !isComposite {
				return mimeError("ParserError", "Entity body is not a type of composite media type. "+
					"Received content-type : "+baseType), nil
			}
			if strings.HasPrefix(strings.ToLower(baseType), "message/") {
				return mimeError("ParserError", "message/* body part decoding is not yet supported. "+
					"Received content-type : "+baseType), nil
			}
			if body == nil || body.kind != bodyBytes {
				return mimeError("ParserError", "Entity body is not a type of composite media type. "+
					"Received content-type : "+baseType), nil
			}
			parts, err := decodeMultipart(ctx, body.bytes, boundary)
			if err != nil {
				return mimeError("ParserError", "Error occurred while extracting body parts from entity: "+err.Error()), nil
			}
			setEntityBody(obj, &entityBody{kind: bodyParts, parts: parts})
			return parts, nil
		})
}
