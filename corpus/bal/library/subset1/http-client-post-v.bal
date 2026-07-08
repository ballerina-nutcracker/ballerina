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
    http:Client c = check new ("https://example.com");

    // POST with string body — headers and mediaType default to ()
    var r = check c->post("/post", "hello world");
    io:println(r.statusCode);           // @output 200

    // POST with explicit Content-Type
    var r2 = check c->post("/post", "{\"key\":\"value\"}", (), "application/json");
    io:println(r2.statusCode);          // @output 200

    // POST with map body — automatically serialized to JSON
    var r3 = check c->post("/post", {"name": "test", "count": 1});
    io:println(r3.statusCode);          // @output 200
    return;
}
