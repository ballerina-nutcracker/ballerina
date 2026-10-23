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

// Outbound payload serialisation, shared by the client (request bodies) and the listener
// (resource return values) so both infer the same media type for the same value.

package native

import (
	"cmp"
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// The dispatch order mirrors jBallerina's Response.setPayload (http_response.bal:538-557).
// v is never nil: writeResult intercepts a () resource return before reaching here, and
// both msgToBody call sites guard against a nil RequestMessage before this is invoked.
// contentType, when non-empty, replaces the inferred type, and a multipart body is encoded with
// its boundary.
func outboundPayload(ctx *extern.Context, types *httpTypes, v values.BalValue, contentType string) ([]byte, string, error) {
	switch p := v.(type) {
	case string:
		return []byte(p), cmp.Or(contentType, "text/plain"), nil
	case values.XMLValue:
		return []byte(p.XMLString()), cmp.Or(contentType, "application/xml"), nil
	case *values.List:
		if !semtypes.IsZero(p.Type) && semtypes.IsSubtype(ctx.TypeCtx(), p.Type, types.byteArrTy) {
			return p.ToByteSlice(), cmp.Or(contentType, "application/octet-stream"), nil
		}
		// A list holding objects can only be a mime:Entity[], since anydata excludes objects.
		if !semtypes.IsZero(p.Type) && !semtypes.IsSubtype(ctx.TypeCtx(), p.Type, types.anydataTy) {
			return encodeMultipartBody(ctx, p, cmp.Or(contentType, "multipart/form-data"))
		}
	}
	b, err := toJSONBytes(v)
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize body to JSON: %w", err)
	}
	return b, cmp.Or(contentType, "application/json"), nil
}

// Preserving an already-set Content-Type is jBallerina's rule (http_commons.bal:475-485),
// and differs from setTextPayload/setJsonPayload here, which overwrite unconditionally.
func payloadContentType(existing string, override values.BalValue, defaultType string) string {
	if s, ok := override.(string); ok && s != "" {
		return s
	}
	if existing != "" {
		return existing
	}
	return defaultType
}

// Externs for methods with defaulted trailing parameters can be invoked with a short
// argument slice.
func optionalArg(args []values.BalValue, i int) values.BalValue {
	if len(args) > i {
		return args[i]
	}
	return nil
}
