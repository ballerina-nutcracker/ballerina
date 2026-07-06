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
    // Default compression (COMPRESSION_AUTO)
    http:Client c1 = check new ("https://example.com", {});
    var r1 = check c1->get("/path");
    io:println(r1.statusCode);  // @output 200

    // Explicit COMPRESSION_AUTO
    http:Client c2 = check new ("https://example.com", {compression: http:COMPRESSION_AUTO});
    var r2 = check c2->get("/path");
    io:println(r2.statusCode);  // @output 200

    // COMPRESSION_ALWAYS
    http:Client c3 = check new ("https://example.com", {compression: http:COMPRESSION_ALWAYS});
    var r3 = check c3->get("/path");
    io:println(r3.statusCode);  // @output 200

    // COMPRESSION_NEVER — disables Accept-Encoding negotiation
    http:Client c4 = check new ("https://example.com", {compression: http:COMPRESSION_NEVER});
    var r4 = check c4->get("/path");
    io:println(r4.statusCode);  // @output 200

    // Compression combined with other config fields
    http:Client c5 = check new ("https://example.com", {
        timeout: 15,
        httpVersion: http:HTTP_1_1,
        compression: http:COMPRESSION_NEVER
    });
    var r5 = check c5->get("/path");
    io:println(r5.statusCode);  // @output 200

    return;
}
