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
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// The dispatch order mirrors jBallerina's Response.setPayload (http_response.bal:538-557).
// v is never nil: writeResult intercepts a () resource return before reaching here, and
// both msgToBody call sites guard against a nil RequestMessage before this is invoked.
func outboundPayload(tc semtypes.Context, types *httpTypes, v values.BalValue) ([]byte, string, error) {
	switch p := v.(type) {
	case string:
		return []byte(p), "text/plain", nil
	case values.XMLValue:
		return []byte(p.XMLString()), "application/xml", nil
	case *values.List:
		if !semtypes.IsZero(p.Type) && semtypes.IsSubtype(tc, p.Type, types.byteArrTy) {
			return p.ToByteSlice(), "application/octet-stream", nil
		}
	}
	b, err := toJSONBytes(v)
	if err != nil {
		return nil, "", err
	}
	return b, "application/json", nil
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
