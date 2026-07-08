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

    // String body
    http:Response r = check c->post("/echo", "hello post");
    io:println(r.statusCode);            // @output 200
    io:println(r.getTextPayload());      // @output body: hello post, ct: text/plain

    // Map body — should be serialized to JSON
    http:Response r2 = check c->post("/echo", {"msg": "hello"});
    io:println(r2.statusCode);           // @output 200
    io:println(r2.getTextPayload());     // @output body: {"msg":"hello"}, ct: application/json
    return;
}
