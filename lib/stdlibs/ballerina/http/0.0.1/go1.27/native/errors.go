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
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// errorType names the http error type a native error is created as, mirroring jBallerina's
// HttpErrorType. The value is matched by createError in http_errors.bal.
type errorType string

const (
	genericClientError   errorType = "GenericClientError"
	noContentError       errorType = "NoContentError"
	headerNotFoundError  errorType = "HeaderNotFoundError"
	genericListenerError errorType = "GenericListenerError"
	// internalInterceptorReturnError is a 500 that error handling does not log.
	internalInterceptorReturnError errorType = "InternalInterceptorReturnError"
	// The internal errors dispatching and binding raise, which error handling does not log.
	internalQueryParameterBindingError     errorType = "InternalQueryParameterBindingError"
	internalHeaderBindingListenerError     errorType = "InternalHeaderBindingListenerError"
	internalPayloadBindingListenerError    errorType = "InternalPayloadBindingListenerError"
	internalResourceNotFoundError          errorType = "InternalResourceNotFoundError"
	internalResourceMethodNotAllowedError  errorType = "InternalResourceMethodNotAllowedError"
	internalResourceDispatchingServerError errorType = "InternalResourceDispatchingServerError"
)

// httpError returns a distinct http error of type t with message and an optional cause.
func httpError(ctx *extern.Context, t errorType, message string, cause *values.Error) *values.Error {
	var causeVal values.BalValue
	if cause != nil {
		causeVal = cause
	}
	result, _ := invokeModuleFunction(ctx, "createError", string(t), message, causeVal)
	return result.(*values.Error)
}

// invokeModuleFunction calls the private http function name with args.
func invokeModuleFunction(ctx *extern.Context, name string, args ...values.BalValue) (values.BalValue, error) {
	handle, _ := ctx.LookupFunction(orgName, moduleName, name)
	return ctx.InvokeFunction(handle, args)
}

// payloadError is the error a Request or Response payload accessor returns when the body cannot
// be read or decoded as kind (json, xml, text or binary); source is "request" or "response".
func payloadError(ctx *extern.Context, kind, source string, cause *values.Error) *values.Error {
	return httpError(ctx, genericClientError, "Error occurred while retrieving the "+kind+" payload from the "+source, cause)
}

// jsonPayload decodes body for getJsonPayload, which reports an empty body as no content.
func jsonPayload(ctx *extern.Context, types *httpTypes, body []byte, source string) values.BalValue {
	if len(body) == 0 {
		return noContent(ctx, "json")
	}
	payload, jsonErr := decodeJSONBody(ctx, types, body)
	if jsonErr != nil {
		return payloadError(ctx, "json", source, jsonErr)
	}
	return payload
}

// errorCause wraps a Go error as a plain Ballerina error to keep it as a cause.
func errorCause(err error) *values.Error {
	return values.NewErrorWithMessage(err.Error())
}

// noContent is the error a payload accessor returns for a message without a kind payload.
func noContent(ctx *extern.Context, kind string) *values.Error {
	return httpError(ctx, noContentError, "No content", values.NewErrorWithMessage("Empty "+kind+" payload"))
}
