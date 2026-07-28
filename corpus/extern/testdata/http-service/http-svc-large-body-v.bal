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

service /big on new http:Listener(19206) {
    resource function post echo(http:Request req) returns http:Response|error {
        string payload = check req.getTextPayload();
        http:Response resp = new;
        resp.setTextPayload(payload);
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19206", {});

    // Build a >8192-byte body so the server buffers it lazily as a stream
    // rather than eagerly (eagerBufferThreshold = 8192). 32 * 2^9 = 16384.
    string body = "0123456789abcdef0123456789abcdef";
    int i = 0;
    while i < 9 {
        body = body + body;
        i += 1;
    }

    http:Response r = check c->post("/big/echo", body);
    io:println(r.statusCode); // @output 200
    io:println(r.getTextPayload().length() == 16384); // @output true
}
