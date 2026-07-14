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

public function main() returns error? {
    http:Client c = check new http:Client("http://testserver", {});

    // PUT with string body
    http:Response r1 = check c->put("/echo", "put body");
    io:println(r1.statusCode);             // @output 200
    io:println(r1.getTextPayload());       // @output body: put body, ct: text/plain

    // PATCH with map body — serialized to JSON
    http:Response r2 = check c->patch("/echo", {"k": "v"});
    io:println(r2.statusCode);             // @output 200
    io:println(r2.getTextPayload());       // @output body: {"k":"v"}, ct: application/json

    // DELETE with no body
    http:Response r3 = check c->delete("/delete");
    io:println(r3.statusCode);             // @output 200

    // HEAD
    http:Response r4 = check c->head("/head");
    io:println(r4.statusCode);             // @output 200

    // OPTIONS
    http:Response r5 = check c->options("/options");
    io:println(r5.statusCode);             // @output 200

    // execute with explicit verb
    http:Response r6 = check c->execute("PUT", "/echo", "exec body");
    io:println(r6.statusCode);             // @output 200
    io:println(r6.getTextPayload());       // @output body: exec body, ct: text/plain

    return;
}
