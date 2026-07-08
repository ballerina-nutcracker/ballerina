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
    // Insecure mode — disable TLS verification
    http:Client c1 = check new ("https://example.com", {secureSocket: {enable: false}});
    var r1 = check c1->get("/path");
    io:println(r1.statusCode);      // @output 200

    // shareSession, serverName, ciphers, handshakeTimeout — compile-time shape check
    http:Client c2 = check new ("https://example.com", {
        secureSocket: {
            enable: false,
            verifyHostName: false,
            shareSession: false,
            serverName: "example.com",
            ciphers: ["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"],
            handshakeTimeout: 10.0
        }
    });
    var r2 = check c2->get("/path");
    io:println(r2.statusCode);      // @output 200

    // protocol field — compile-time shape check
    http:Client c3 = check new ("https://example.com", {
        secureSocket: {
            enable: false,
            protocol: {name: "TLS", versions: ["TLSv1.2", "TLSv1.3"]}
        }
    });
    var r3 = check c3->get("/path");
    io:println(r3.statusCode);      // @output 200

    return;
}
