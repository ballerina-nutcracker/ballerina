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

// Exercises the optional TLS listener settings: protocol version bounds, an
// explicit (RSA-compatible) cipher suite, and disabled session tickets.
http:ListenerConfiguration secureConfig = {
    httpVersion: http:HTTP_1_1,
    secureSocket: {
        key: {
            certFile: "testdata/certs/server.crt",
            keyFile: "testdata/certs/server.key"
        },
        protocol: ["TLSv1.2"],
        ciphers: ["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"],
        shareSession: false
    }
};

service /secure on new http:Listener(19207, secureConfig) {
    resource function get hello() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("secure hello");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("https://localhost:19207", {secureSocket: {enable: false}});
    http:Response r = check c->get("/secure/hello");
    io:println(r.statusCode); // @output 200
    io:println(r.getTextPayload()); // @output secure hello
}
