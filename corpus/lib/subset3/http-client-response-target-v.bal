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

type Echo record {
    string rq;
};

type EchoOrResponse Echo|http:Response;

public function main() returns error? {
    http:Client c = check new ("https://example.com");

    // A target admitting http:Response yields the response itself, not the bound payload.
    Echo|http:Response recordFirst = check c->get("/path");
    io:println(recordFirst is http:Response); // @output true

    http:Response|string responseFirst = check c->post("/path", "req");
    io:println(responseFirst is http:Response); // @output true

    var explicit = check c->execute("GET", "/path", (), targetType = EchoOrResponse);
    io:println(explicit is http:Response); // @output true

    anydata|http:Response anyTarget = check c->options("/path");
    if anyTarget is http:Response {
        io:println(anyTarget.statusCode); // @output 200
        io:println(anyTarget.reasonPhrase); // @output OK
    }

    http:Response|error headResp = c->head("/path");
    io:println(headResp is http:Response); // @output true
}
