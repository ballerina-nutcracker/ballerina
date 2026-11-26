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

import ballerina/mime;
import ballerina/url;

// Returns the response a resource's returned status code record is sent as, or () for any
// other value. Called by the native listener dispatcher.
isolated function toStatusCodeResponse(any message) returns Response? {
    if message is StatusCodeResponse {
        return createStatusCodeResponse(message);
    }
    return ();
}

isolated function createStatusCodeResponse(StatusCodeResponse message) returns Response {
    Response response = new;
    response.statusCode = message.status.code;

    var headers = message?.headers;
    if headers !is () {
        setStatusCodeHeaders(headers, response);
    }

    string? mediaType = message?.mediaType;
    if mediaType is string {
        mediaType = mediaType.trim();
    }
    anydata body = message?.body;
    if body !is () {
        setStatusCodePayload(body, response, mediaType);
    }
    // The mediaType field overrides both a Content-Type in headers and the payload's default.
    if mediaType is string {
        response.setHeader(mime:CONTENT_TYPE, mediaType);
    }
    return response;
}

isolated function setStatusCodeHeaders(map<string|int|boolean|string[]|int[]|boolean[]> headers, Response response) {
    foreach string key in headers.keys() {
        var value = headers.get(key);
        if value is anydata[] {
            foreach anydata item in value {
                response.addHeader(key, item.toString());
            }
            continue;
        }
        response.setHeader(key, value.toString());
    }
}

isolated function setStatusCodePayload(anydata payload, Response response, string? mediaType) {
    if payload is xml {
        response.setXmlPayload(payload);
    } else if payload is string {
        response.setTextPayload(payload);
    } else if payload is byte[] {
        response.setBinaryPayload(payload);
    } else if isFormMediaType(mediaType) {
        setFormPayload(payload, response);
    } else {
        error? result = response.setAnydataPayload(payload);
        if result is error {
            panic error InitializingOutboundResponseError(string `anydata to json conversion error: ${result.message()}`, result);
        }
    }
}

isolated function isFormMediaType(string? mediaType) returns boolean {
    if mediaType is () {
        return false;
    }
    mime:MediaType|error parsed = mime:getMediaType(mediaType);
    return parsed is mime:MediaType && parsed.getBaseType().toLowerAscii() == mime:APPLICATION_FORM_URLENCODED;
}

isolated function setFormPayload(anydata payload, Response response) {
    map<string>|error pairs = payload.cloneWithType();
    if pairs is error {
        panic error InitializingOutboundResponseError("unsupported content for application/x-www-form-urlencoded media type");
    }
    string formData = "";
    foreach string key in pairs.keys() {
        string|error encodedKey = url:encode(key, "UTF-8");
        string|error encodedValue = url:encode(pairs.get(key), "UTF-8");
        if encodedKey is error {
            panic error InitializingOutboundResponseError("content encoding error: " + encodedKey.message(), encodedKey);
        }
        if encodedValue is error {
            panic error InitializingOutboundResponseError("content encoding error: " + encodedValue.message(), encodedValue);
        }
        string separator = formData == "" ? "" : "&";
        formData += separator + encodedKey + "=" + encodedValue;
    }
    response.setTextPayload(formData, mime:APPLICATION_FORM_URLENCODED);
}
