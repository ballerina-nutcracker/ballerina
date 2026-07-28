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

// HTTP_1_0 is accepted at compile time but not supported by the Go HTTP
// runtime; the listener must fall back to HTTP/1.1 and print a warning
// rather than forwarding "1.0" to the platform layer.
http:ListenerConfiguration cfg = {
    httpVersion: http:HTTP_1_0
};

service /http10 on new http:Listener(19213, cfg) {
    resource function get ping() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("pong");
        return resp;
    }
}

public function testMain() returns error? {
    // The listener fell back to HTTP/1.1; the client must not attempt
    // plaintext h2c prior-knowledge against an HTTP/1.1-only server.
    http:Client c = check new http:Client("http://localhost:19213", {httpVersion: http:HTTP_1_1});
    http:Response r = check c->get("/http10/ping");
    io:println(r.statusCode); // @output 200
    io:println(r.getTextPayload()); // @output pong
}
