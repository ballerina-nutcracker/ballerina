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

service /resp on new http:Listener(19203) {
    // No return value: dispatch yields (), which the server maps to 202 Accepted.
    resource function post accepted() {
    }

    // Returning an error value is mapped to a 500 error response.
    resource function get failing() returns http:Response|error {
        return error("boom");
    }

    // Two values for the same header exercise the addHeader (Add) path. The
    // hop-by-hop Connection header must be dropped before reaching the client.
    resource function get headers() returns http:Response {
        http:Response resp = new;
        resp.addHeader("X-Multi", "a");
        resp.addHeader("X-Multi", "b");
        resp.setHeader("Connection", "keep-alive");
        resp.setTextPayload("ok");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19203", {});

    http:Response accepted = check c->post("/resp/accepted", "");
    io:println(accepted.statusCode); // @output 202

    http:Response failing = check c->get("/resp/failing");
    io:println(failing.statusCode); // @output 500

    http:Response headers = check c->get("/resp/headers");
    string[] multi = check headers.getHeaders("X-Multi");
    io:println(multi.length()); // @output 2
    io:println(headers.hasHeader("Connection")); // @output false
}
