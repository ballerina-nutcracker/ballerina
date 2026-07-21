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

service /api on new http:Listener(19193) {
    resource function get ping() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("pong");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19193", {});
    // Matching resource → 200.
    http:Response ok = check c->get("/api/ping");
    io:println(ok.statusCode); // @output 200
    // Unknown path under the service → 404.
    http:Response notFound = check c->get("/api/missing");
    io:println(notFound.statusCode); // @output 404
    // Known path, wrong method → 405.
    http:Response wrongMethod = check c->post("/api/ping", "");
    io:println(wrongMethod.statusCode); // @output 405
}
