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

service /sc on new http:Listener(19195) {
    // Returns 201 Created via direct field assignment on statusCode.
    resource function post item() returns http:Response {
        http:Response resp = new;
        resp.statusCode = 201;
        resp.setTextPayload("created");
        return resp;
    }

    // Returns 404 Not Found via direct field assignment on statusCode.
    resource function get missing() returns http:Response {
        http:Response resp = new;
        resp.statusCode = 404;
        resp.setTextPayload("not found");
        return resp;
    }

    // Returns 200 OK with no explicit statusCode assignment (default).
    resource function get ping() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("pong");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19195", {});

    // Direct assignment: 201 Created.
    http:Response created = check c->post("/sc/item", "");
    io:println(created.statusCode); // @output 201
    io:println(created.getTextPayload()); // @output created

    // Direct assignment: 404 Not Found.
    http:Response notFound = check c->get("/sc/missing");
    io:println(notFound.statusCode); // @output 404
    io:println(notFound.getTextPayload()); // @output not found

    // Default statusCode is 200.
    http:Response ok = check c->get("/sc/ping");
    io:println(ok.statusCode); // @output 200
    io:println(ok.getTextPayload()); // @output pong
}
