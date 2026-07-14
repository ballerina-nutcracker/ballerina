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
    // no proxy (default)
    http:Client _ = check new ("https://example.com");
    // proxy without auth
    http:Client _ = check new ("https://example.com", {
        proxy: {host: "proxy.example.com", port: 3128}
    });
    // proxy with auth
    http:Client _ = check new ("https://example.com", {
        proxy: {host: "proxy.example.com", port: 3128, userName: "user", password: "secret"}
    });
    io:println("ok");
}
// @output ok
