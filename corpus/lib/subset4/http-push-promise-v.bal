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

public function main() {
    http:PushPromise defaults = new;
    io:println(defaults.path, " ", defaults.method); // @output / GET

    http:PushPromise promise = new ("/styles.css", "HEAD");
    io:println(promise.path, " ", promise.method); // @output /styles.css HEAD
    io:println(promise.hasHeader("Accept"), " ", promise.getHeaderNames()); // @output false []

    promise.addHeader("Accept", "text/css");
    promise.addHeader("accept", "*/*");
    promise.addHeader("X-Trace", "a");
    io:println(promise.hasHeader("ACCEPT"), " ", promise.getHeader("accept")); // @output true text/css
    io:println(promise.getHeaders("Accept")); // @output ["text/css","*/*"]
    io:println(promise.getHeaderNames()); // @output ["Accept","X-Trace"]

    http:PushPromise ordered = new ("/a");
    ordered.addHeader("b-h", "1");
    ordered.addHeader("A-h", "2");
    ordered.addHeader("b-H", "3");
    io:println(ordered.getHeaderNames(), " ", ordered.getHeader("B-H")); // @output ["A-h","b-h"] 1
    ordered.setHeader("B-h", "4");
    io:println(ordered.getHeaderNames(), " ", ordered.getHeader("b-h")); // @output ["A-h","B-h"] 4

    promise.addHeader("Content-Type", "text/plain");
    promise.addHeader("content-type", "text/css");
    io:println(promise.getHeaders("Content-Type")); // @output ["text/css"]

    promise.setHeader("ACCEPT", "text/html");
    io:println(promise.getHeaders("accept"), " ", promise.getHeaderNames()); // @output ["text/html"] ["ACCEPT","content-type","X-Trace"]

    promise.removeHeader("x-trace");
    io:println(promise.getHeaderNames()); // @output ["ACCEPT","content-type"]
    io:println(promise.getHeaders("missing")); // @output []

    promise.removeAllHeaders();
    io:println(promise.getHeaderNames()); // @output []

    promise.path = "/script.js";
    promise.method = "GET";
    io:println(promise.path, " ", promise.method); // @output /script.js GET
}
