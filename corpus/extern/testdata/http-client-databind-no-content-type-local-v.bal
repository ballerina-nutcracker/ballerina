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

// With no Content-Type on the response, the builder is picked from the target type alone.
// The server here is a Go one: an http:Listener always emits a Content-Type, so a Ballerina
// service cannot serve these responses.

import ballerina/http;
import ballerina/io;

type Person record {|
    string name;
    int age;
|};

enum Colour {
    RED = "red",
    GREEN = "green"
}

public function main() returns error? {
    http:Client c = check new http:Client("http://testserver", {});

    // string and byte[] targets are read directly.
    string untyped = check c->get("/no-type");
    io:println(untyped); // @output untyped body

    byte[] untypedBytes = check c->get("/no-type");
    io:println(untypedBytes.length()); // @output 12

    // A target narrower than the builder's own type is converted on this path too.
    Colour fallbackColour = check c->get("/no-type-colour");
    io:println(fallbackColour, " ", <any>fallbackColour is Colour); // @output red true

    Colour? optFallbackColour = check c->get("/no-type-colour");
    io:println(optFallbackColour, " ", <any>optFallbackColour is Colour); // @output red true

    // Every other target is parsed as JSON, so a text body fails to bind to a record.
    Person|error asRecord = c->get("/no-type");
    if asRecord is error {
        io:println(asRecord.message()); // @output failed to parse JSON payload: invalid character 'u' looking for beginning of value
    }

    // An empty body binds to () for nilable targets, whatever the target's builder would be.
    string? untypedEmpty = check c->get("/no-type-empty");
    io:println(untypedEmpty is ()); // @output true

    byte[]? untypedEmptyBytes = check c->get("/no-type-empty");
    io:println(untypedEmptyBytes is ()); // @output true

    Colour? noFallbackColour = check c->get("/no-type-empty");
    io:println(noFallbackColour is ()); // @output true

    return;
}
