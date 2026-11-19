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
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var nettyReasonPhrases = map[int]string{
	100: "Continue", 101: "Switching Protocols", 102: "Processing", 103: "Early Hints",
	200: "OK", 201: "Created", 202: "Accepted", 203: "Non-Authoritative Information", 204: "No Content",
	205: "Reset Content", 206: "Partial Content", 207: "Multi-Status",
	300: "Multiple Choices", 301: "Moved Permanently", 302: "Found", 303: "See Other", 304: "Not Modified",
	305: "Use Proxy", 307: "Temporary Redirect", 308: "Permanent Redirect",
	400: "Bad Request", 401: "Unauthorized", 402: "Payment Required", 403: "Forbidden", 404: "Not Found",
	405: "Method Not Allowed", 406: "Not Acceptable", 407: "Proxy Authentication Required",
	408: "Request Timeout", 409: "Conflict", 410: "Gone", 411: "Length Required", 412: "Precondition Failed",
	413: "Request Entity Too Large", 414: "Request-URI Too Long", 415: "Unsupported Media Type",
	416: "Requested Range Not Satisfiable", 417: "Expectation Failed", 421: "Misdirected Request",
	422: "Unprocessable Entity", 423: "Locked", 424: "Failed Dependency", 425: "Unordered Collection",
	426: "Upgrade Required", 428: "Precondition Required", 429: "Too Many Requests",
	431: "Request Header Fields Too Large", 451: "Unavailable For Legal Reasons",
	500: "Internal Server Error", 501: "Not Implemented", 502: "Bad Gateway", 503: "Service Unavailable",
	504: "Gateway Timeout", 505: "HTTP Version Not Supported", 506: "Variant Also Negotiates",
	507: "Insufficient Storage", 508: "Loop Detected", 510: "Not Extended", 511: "Network Authentication Required",
}

var nettyStatusClasses = []string{"Informational", "Success", "Redirection", "Client Error", "Server Error"}

// errorReasonPhrase returns the listed phrase for code, "<class> (<code>)" for an unlisted code in
// 100-599, and "Internal Server Error" otherwise.
func errorReasonPhrase(code int) string {
	if phrase, ok := nettyReasonPhrases[code]; ok {
		return phrase
	}
	if code >= 100 && code < 600 {
		return nettyStatusClasses[code/100-1] + " (" + strconv.Itoa(code) + ")"
	}
	return nettyReasonPhrases[500]
}

// formatInstant formats t in UTC as yyyy-MM-ddTHH:mm:ss, followed by 0, 3, 6 or 9 fraction digits
// (the fewest that keep the nanoseconds exact) and "Z".
func formatInstant(t time.Time) string {
	t = t.UTC()
	nanos := t.Nanosecond()
	fraction := ""
	if nanos != 0 {
		digits := fmt.Sprintf("%09d", nanos)
		switch {
		case nanos%1_000_000 == 0:
			fraction = "." + digits[:3]
		case nanos%1_000 == 0:
			fraction = "." + digits[:6]
		default:
			fraction = "." + digits
		}
	}
	return t.Format("2006-01-02T15:04:05") + fraction + "Z"
}

// errorPayload returns the ErrorPayload JSON body for a status, error message and request.
func errorPayload(now time.Time, status int, message, path, method string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"timestamp":`)
	writeJSONString(&buf, formatInstant(now))
	buf.WriteString(`,"status":`)
	buf.WriteString(strconv.Itoa(status))
	buf.WriteString(`,"reason":`)
	writeJSONString(&buf, errorReasonPhrase(status))
	buf.WriteString(`,"message":`)
	writeJSONString(&buf, message)
	buf.WriteString(`,"path":`)
	writeJSONString(&buf, path)
	buf.WriteString(`,"method":`)
	writeJSONString(&buf, method)
	buf.WriteByte('}')
	return buf.Bytes()
}

func registerErrorResponseExterns(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setErrorPayload", setErrorPayloadExtern(rt))
	// errorDetail stands in for error:detail(), which the langlib does not provide yet (#1443).
	runtime.RegisterExternFunction(rt, orgName, moduleName, "errorDetail", errorDetailExtern)
}

func setErrorPayloadExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		statusCode, _ := self.Get("statusCode")
		body := errorPayload(rt.Platform().Time.Now(), int(statusCode.(int64)), args[1].(string), args[2].(string), args[3].(string))
		self.Put("body", &responseBodyHolder{payloadSet: true, buf: body})
		headers := responseHeaders(self)
		if _, ok := headers.Get("content-type"); !ok {
			headers.Put(ctx.TypeCtx(), "content-type", newListValue([]values.BalValue{"application/json"}))
		}
		return nil, nil
	}
}

func errorDetailExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*values.Error).Detail, nil
}
