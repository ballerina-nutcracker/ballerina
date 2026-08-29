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

type Person record {|
    string name;
    int age;
|};

public function main() returns error? {
    http:Client c = check new ("https://example.com");

    // http:Response target — the response is handed back untouched.
    http:Response r = check c->get("/path");
    io:println(r.statusCode); // @output 200

    // A union containing http:Response also yields the response, not a bound payload.
    http:Response|string ru = check c->get("/path");
    io:println(ru is string); // @output false

    // string target — the stub body has no Content-Type, so the builder is
    // picked from the target type.
    string text = check c->get("/path");
    io:println(text); // @output test body

    // Nilable string target.
    string? optText = check c->get("/path");
    io:println(optText); // @output test body

    // byte[] target.
    byte[] bytes = check c->get("/path");
    io:println(bytes.length()); // @output 9

    // Nilable byte[] target.
    byte[]? optBytes = check c->get("/path");
    io:println(optBytes is byte[]); // @output true

    // () target discards the payload.
    () nothing = check c->get("/path");
    io:println(nothing); // @output

    // The target type can be passed explicitly when there is no contextual type.
    var explicitText = check c->get("/path", targetType = string);
    io:println(explicitText); // @output test body

    // The stub body is not valid JSON, so binding to a record fails.
    Person|error person = c->get("/path");
    io:println(person is error); // @output true

    // An xml target with no Content-Type falls back to the target type; the stub body
    // parses as an xml text node.
    xml asXml = check c->get("/path");
    io:println(asXml); // @output test body
    io:println(asXml is xml:Text); // @output true

    return;
}
