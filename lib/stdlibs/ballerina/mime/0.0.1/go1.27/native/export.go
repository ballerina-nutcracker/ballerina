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

package native

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// EntityHeader is one mime:Entity header name, in its original casing, with every value set for it.
type EntityHeader struct {
	Name   string
	Values []string
}

// EntityHeaders returns a mime:Entity's headers in the order they were first set, skipping headers
// whose value list is empty.
func EntityHeaders(obj *values.Object) []EntityHeader {
	namesVal, _ := obj.Get("headerNames")
	names, ok := namesVal.(*values.List)
	if !ok {
		return nil
	}
	hmVal, _ := obj.Get("headerMap")
	hm, ok := hmVal.(*values.Map)
	if !ok {
		return nil
	}
	result := make([]EntityHeader, 0, names.Len())
	for i := range names.Len() {
		name, _ := names.Get(i).(string)
		v, _ := hm.Get(strings.ToLower(name))
		list, ok := v.(*values.List)
		if !ok || list.Len() == 0 {
			continue
		}
		vals := make([]string, list.Len())
		for j := range list.Len() {
			vals[j], _ = list.Get(j).(string)
		}
		result = append(result, EntityHeader{Name: name, Values: vals})
	}
	return result
}

// EntityBodyParts returns the parts of a mime:Entity whose body was set with setBodyParts, and
// false for any other body.
func EntityBodyParts(obj *values.Object) ([]*values.Object, bool) {
	body := getEntityBody(obj)
	if body == nil || body.kind != bodyParts {
		return nil, false
	}
	parts := make([]*values.Object, body.parts.Len())
	for i := range parts {
		parts[i], _ = body.parts.Get(i).(*values.Object)
	}
	return parts, true
}

// EntityBytes returns a mime:Entity's body bytes, reading a byte-channel body and encoding a text,
// JSON or XML body in the Content-Type charset.
func EntityBytes(ctx *extern.Context, obj *values.Object) ([]byte, error) {
	return bytesForBody(ctx, obj)
}

// EncodeCharset encodes s in charset as Java's String.getBytes does, using UTF-8 when charset is
// empty and '?' for characters the charset cannot represent.
func EncodeCharset(s, charset string) ([]byte, error) {
	return encodeCharset(s, charset)
}

// MimeBase64 encodes data as MIME base64: 76-character lines separated by CRLF, with no trailing
// line break.
func MimeBase64(data []byte) string {
	return mimeEncode(data)
}
