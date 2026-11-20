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

import ballerina/http;
import ballerina/io;

function describe(any|error result) returns string {
    if result is http:NoContentError {
        return "NoContentError: " + result.message();
    }
    if result is http:GenericClientError {
        return "GenericClientError: " + result.message();
    }
    if result is http:HeaderNotFoundError {
        return "HeaderNotFoundError: " + result.message();
    }
    if result is http:ClientError {
        return "ClientError: " + result.message();
    }
    if result is error {
        return "error: " + result.message();
    }
    if result is xml {
        return "ok: xml of length " + result.length().toString();
    }
    return "ok: " + result.toBalString();
}

public function main() {
    http:Request none = new;
    io:println(describe(none.getTextPayload())); // @output NoContentError: No content
    io:println(describe(none.getJsonPayload())); // @output NoContentError: No content
    io:println(describe(none.getXmlPayload())); // @output NoContentError: No content
    io:println(describe(none.getBinaryPayload())); // @output ok: []
    io:println(describe(none.getFormParams())); // @output GenericClientError: Error occurred while retrieving form parameters from the request
    io:println(describe(none.getBodyParts())); // @output GenericClientError: Error occurred while retrieving body parts from the request
    io:println(describe(none.getHeader("x-a"))); // @output HeaderNotFoundError: Http header does not exist
    io:println(describe(none.getHeaders("x-a"))); // @output HeaderNotFoundError: Http header does not exist

    http:Request empty = new;
    empty.setTextPayload("");
    io:println(describe(empty.getTextPayload())); // @output ok: ""
    io:println(describe(empty.getXmlPayload())); // @output ok: xml of length 0
    io:println(describe(empty.getJsonPayload())); // @output NoContentError: No content

    http:Request invalidJson = new;
    invalidJson.setTextPayload("{bad", "application/json");
    io:println(describe(invalidJson.getJsonPayload())); // @output GenericClientError: Error occurred while retrieving the json payload from the request
    io:println(describe(invalidJson.getFormParams())); // @output GenericClientError: Error occurred while retrieving form parameters from the request
    io:println(describe(invalidJson.getBodyParts())); // @output GenericClientError: Error occurred while retrieving body parts from the request

    http:Request noEquals = new;
    noEquals.setTextPayload("justastring", "application/x-www-form-urlencoded");
    io:println(describe(noEquals.getFormParams())); // @output ClientError: Datasource does not contain form data

    http:Request multipart = new;
    multipart.setBodyParts([new]);
    io:println(describe(multipart.getTextPayload())); // @output NoContentError: No content
    io:println(describe(multipart.getXmlPayload())); // @output NoContentError: No content

    http:Response multipartResponse = new;
    multipartResponse.setBodyParts([new]);
    io:println(describe(multipartResponse.getTextPayload())); // @output NoContentError: No content
    io:println(describe(multipartResponse.getXmlPayload())); // @output NoContentError: No content

    http:Response response = new;
    io:println(describe(response.getTextPayload())); // @output NoContentError: No content
    io:println(describe(response.getJsonPayload())); // @output NoContentError: No content
    io:println(describe(response.getXmlPayload())); // @output NoContentError: No content
    io:println(describe(response.getHeader("x-a"))); // @output HeaderNotFoundError: Http header does not exist
    io:println(describe(response.getHeaders("x-a"))); // @output HeaderNotFoundError: Http header does not exist

    http:Response invalidXml = new;
    invalidXml.setTextPayload("<a", "application/xml");
    io:println(describe(invalidXml.getXmlPayload())); // @output GenericClientError: Error occurred while retrieving the xml payload from the response
    io:println(describe(invalidXml.getJsonPayload())); // @output GenericClientError: Error occurred while retrieving the json payload from the response
    io:println(describe(invalidXml.getBodyParts())); // @output GenericClientError: Error occurred while retrieving body parts from the response: Entity body is not a type of composite media type. Received content-type : application/xml

    io:println(describe(http:parseHeader("a;b=c"))); // @output ok: [{"value":"a","params":{"b":"c"}}]
    io:println(describe(http:parseHeader(" ;x"))); // @output GenericClientError: failed to parse: error InvalidHeaderValueError ("invalid header value: ;x")
}
