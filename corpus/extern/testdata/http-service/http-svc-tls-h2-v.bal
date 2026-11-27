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

listener http:Listener h2 = new (19251, {
    secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/server.key"}}
});
listener http:Listener h11 = new (19252, {
    httpVersion: http:HTTP_1_1,
    secureSocket: {key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/server.key"}}
});
// Without an AES-128-GCM suite, the HTTP/2 listener cannot carry HTTP/2 and serves HTTP/1.1.
listener http:Listener noH2Cipher = new (19253, {
    secureSocket: {
        key: {certFile: "testdata/certs/server.crt", keyFile: "testdata/certs/server.key"},
        protocol: ["TLSv1.2"],
        ciphers: ["TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"]
    }
});

function httpsProtocol(int port) returns string|error = external;

service / on h2, h11, noH2Cipher {
    resource function get hello() returns string => "hello";
}

public function testMain() returns error? {
    io:println(check httpsProtocol(19251)); // @output HTTP/2.0 200
    io:println(check httpsProtocol(19252)); // @output HTTP/1.1 200
    io:println(check httpsProtocol(19253)); // @output HTTP/1.1 200
}
