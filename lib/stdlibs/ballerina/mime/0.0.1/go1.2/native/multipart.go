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
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

// entityMethodKeys lists every public Entity method, keyed by name, pointing at its
// qualified BIR lookup key. Shared by every native path that constructs a fresh
// Entity value (this file's multipart decoder, and sibling stdlibs via NewEntity)
// so a natively-built Entity dispatches identically to one built via `new Entity()`.
func entityMethodKeys() map[string]string {
	names := []string{
		"setContentType", "getContentType", "setContentId", "getContentId",
		"setContentLength", "getContentLength", "setContentDisposition", "getContentDisposition",
		"setBody", "setJson", "getJson", "setText", "getText", "setByteArray", "getByteArray",
		"getHeader", "getHeaders", "getHeaderNames", "addHeader", "setHeader",
		"removeHeader", "removeAllHeaders", "hasHeader", "setBodyParts", "getBodyParts",
	}
	keys := make(map[string]string, len(names))
	for _, n := range names {
		keys[n] = "ballerina/mime:Entity." + n
	}
	return keys
}

func stringListSemType(ctx *extern.Context) semtypes.SemType {
	bld := semtypes.NewListDefinition()
	return bld.DefineListTypeWrappedWithEnvSemType(ctx.Env.TypeEnv, semtypes.STRING)
}

func stringListMapSemType(ctx *extern.Context, elemType semtypes.SemType) semtypes.SemType {
	mmd := semtypes.NewMappingDefinition()
	return mmd.DefineMappingTypeWrapped(ctx.Env.TypeEnv, nil, elemType)
}

// NewEntity constructs a fresh mime:Entity-shaped native object with the same
// zero-state as `.bal`-level `new Entity()`. Exported so sibling stdlibs (e.g.
// http) that need to construct Entity values natively — e.g. multipart body
// parts decoded from a raw request/response body — can reuse this shape
// instead of duplicating the method-key table.
func NewEntity(ctx *extern.Context) *values.Object {
	stringListType := stringListSemType(ctx)
	headerMapType := stringListMapSemType(ctx, stringListType)
	emptyHeaderMap := values.NewMap(headerMapType, semtypes.ToMappingAtomicType(ctx.TypeCtx, headerMapType), false, nil)
	emptyHeaderNames := values.NewList(stringListType, semtypes.ToListAtomicType(ctx.TypeCtx, stringListType), false, nil, 0, nil)
	return values.NewObject(
		semtypes.OBJECT,
		map[string]values.BalValue{
			"cType":        nil,
			"cId":          "",
			"cLength":      int64(0),
			"cDisposition": nil,
			"headerMap":    emptyHeaderMap,
			"headerNames":  emptyHeaderNames,
		},
		entityMethodKeys(),
		nil,
	)
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

// setEntityHeaders replaces an Entity's headerMap/headerNames fields from a parsed
// MIME header set, preserving multi-value headers and original header-name casing.
func setEntityHeaders(ctx *extern.Context, obj *values.Object, header textproto.MIMEHeader) {
	stringListType := stringListSemType(ctx)
	headerMapType := stringListMapSemType(ctx, stringListType)
	entries := make([]values.MapEntry, 0, len(header))
	names := make([]values.BalValue, 0, len(header))
	for key, vals := range header {
		items := make([]values.BalValue, len(vals))
		for i, v := range vals {
			items[i] = v
		}
		valueList := values.NewList(stringListType, semtypes.ToListAtomicType(ctx.TypeCtx, stringListType), false, nil, 0, items)
		entries = append(entries, values.MapEntry{Key: strings.ToLower(key), Value: valueList})
		names = append(names, key)
	}
	obj.Put("headerMap", values.NewMap(headerMapType, semtypes.ToMappingAtomicType(ctx.TypeCtx, headerMapType), false, entries))
	obj.Put("headerNames", values.NewList(stringListType, semtypes.ToListAtomicType(ctx.TypeCtx, stringListType), false, nil, 0, names))
}

// EntityListFromParts builds a Ballerina Entity[] list value from native part objects.
func EntityListFromParts(ctx *extern.Context, parts []*values.Object) *values.List {
	items := make([]values.BalValue, len(parts))
	for i, p := range parts {
		items[i] = p
	}
	bld := semtypes.NewListDefinition()
	ty := bld.DefineListTypeWrappedWithEnvSemType(ctx.Env.TypeEnv, semtypes.OBJECT)
	return values.NewList(ty, semtypes.ToListAtomicType(ctx.TypeCtx, ty), false, nil, 0, items)
}

// MultipartBoundary parses a Content-Type header value and reports whether it names a
// composite (multipart/* or message/*) media type, along with its boundary parameter.
func MultipartBoundary(contentType string) (baseType, boundary string, isComposite bool) {
	if contentType == "" {
		return "", "", false
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return contentType, "", false
	}
	primaryType := strings.ToLower(strings.SplitN(mediaType, "/", 2)[0])
	if primaryType != "multipart" && primaryType != "message" {
		return mediaType, "", false
	}
	return mediaType, params["boundary"], true
}

// DecodeMultipart splits a raw multipart body into per-part Entity values, defaulting
// an absent per-part Content-Type to "text/plain" (matching jBallerina's underlying
// MIME library default) and copying every part header verbatim.
func DecodeMultipart(ctx *extern.Context, data []byte, boundary string) ([]*values.Object, error) {
	if boundary == "" {
		return nil, fmt.Errorf("no boundary parameter found in Content-Type")
	}
	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	var parts []*values.Object
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.Header.Get("Content-Type") == "" {
			part.Header.Set("Content-Type", "text/plain")
		}
		bodyBytes, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		partObj := NewEntity(ctx)
		setEntityHeaders(ctx, partObj, part.Header)
		SetEntityBody(partObj, &EntityBody{Kind: BodyBytes, Bytes: bodyBytes})
		parts = append(parts, partObj)
	}
	return parts, nil
}

// entityHeaderMap reads an Entity's headerMap/headerNames fields back into a Go header map,
// preserving original header-name casing (the reverse of setEntityHeaders).
func entityHeaderMap(obj *values.Object) map[string][]string {
	result := make(map[string][]string)
	namesVal, ok := obj.Get("headerNames")
	if !ok {
		return result
	}
	names, ok := namesVal.(*values.List)
	if !ok {
		return result
	}
	hmVal, ok := obj.Get("headerMap")
	if !ok {
		return result
	}
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
			s, _ := list.Get(j).(string)
			vals[j] = s
		}
		result[name] = vals
	}
	return result
}

// EncodeMultipart serializes body parts into multipart-encoded wire bytes. If boundary is
// empty, one is generated; the boundary actually used is always returned so the caller can
// record the final Content-Type. Exported for sibling stdlibs (http) that need to serialize a
// multipart body for wire transmission — mime's own Entity.getByteArray() intentionally does
// not do this, matching jBallerina, where serializing a multipart entity to bytes is not
// exposed through mime's public API either (jBallerina's HTTP transport layer does it
// internally instead).
func EncodeMultipart(parts []*values.Object, boundary string) (data []byte, usedBoundary string, err error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if boundary != "" {
		if err := w.SetBoundary(boundary); err != nil {
			return nil, "", err
		}
	}
	for _, part := range parts {
		header := textproto.MIMEHeader(entityHeaderMap(part))
		pw, err := w.CreatePart(header)
		if err != nil {
			return nil, "", err
		}
		partData, err := BytesForBody(GetEntityBody(part))
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

func initMultipartModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSetBodyParts",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			list, ok := args[1].(*values.List)
			if !ok {
				return nil, fmt.Errorf("second argument must be an Entity array")
			}
			parts := make([]*values.Object, list.Len())
			for i := range list.Len() {
				part, ok := list.Get(i).(*values.Object)
				if !ok {
					return nil, fmt.Errorf("body part at index %d is not an Entity object", i)
				}
				parts[i] = part
			}
			SetEntityBody(obj, &EntityBody{Kind: BodyParts, Parts: parts})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGetBodyParts",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			obj, ok := args[0].(*values.Object)
			if !ok {
				return nil, fmt.Errorf("first argument must be an Entity object")
			}
			body := GetEntityBody(obj)
			if body != nil && body.Kind == BodyParts {
				return EntityListFromParts(ctx, body.Parts), nil
			}
			contentType, _ := entityHeaderValue(obj, "content-type")
			baseType, boundary, isComposite := MultipartBoundary(contentType)
			if !isComposite {
				return mimeError("ParserError", "Entity body is not a type of composite media type. "+
					"Received content-type : "+baseType), nil
			}
			if body == nil || body.Kind != BodyBytes {
				return mimeError("ParserError", "Entity body is not a type of composite media type. "+
					"Received content-type : "+baseType), nil
			}
			parts, err := DecodeMultipart(ctx, body.Bytes, boundary)
			if err != nil {
				return mimeError("ParserError", "Error occurred while extracting body parts from entity: "+err.Error()), nil
			}
			SetEntityBody(obj, &EntityBody{Kind: BodyParts, Parts: parts})
			return EntityListFromParts(ctx, parts), nil
		})
}

func init() {
	runtime.RegisterModuleInitializer(initMultipartModule)
}
