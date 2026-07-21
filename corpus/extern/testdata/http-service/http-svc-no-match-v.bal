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

service /api on new http:Listener(19204) {
    resource function get ping() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("pong");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19204", {});

    // No service is attached to this base path → 404.
    http:Response noService = check c->get("/other/thing");
    io:println(noService.statusCode); // @output 404

    // The base path matches a service but the empty sub-path resolves no
    // resource → 404.
    http:Response bareBase = check c->get("/api");
    io:println(bareBase.statusCode); // @output 404
}
