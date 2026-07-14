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
    http:Client c1 = check new ("https://example.com", {});
    var r1 = check c1->get("/path");
    io:println(r1.statusCode);        // @output 200

    http:Client c2 = check new ("https://example.com", {timeout: 30d, followRedirects: {enabled: false}});
    var r2 = check c2->get("/path");
    io:println(r2.statusCode);        // @output 200
    io:println(r2.getTextPayload());  // @output test body

    // Default config — no second arg needed
    http:Client c3 = check new ("https://example.com");
    var r3 = check c3->get("/path");
    io:println(r3.statusCode);        // @output 200

    // Pass request headers
    http:Client c4 = check new ("https://example.com");
    var r4 = check c4->get("/path", {"X-Custom": "value"});
    io:println(r4.statusCode);        // @output 200

    // httpVersion: "1.1" and "2.0" are valid HttpVersion values
    http:Client c5 = check new ("https://example.com", {httpVersion: "1.1"});
    var r5 = check c5->get("/path");
    io:println(r5.statusCode);        // @output 200

    http:Client c6 = check new ("https://example.com", {httpVersion: "2.0"});
    var r6 = check c6->get("/path");
    io:println(r6.statusCode);        // @output 200
    return;
}
