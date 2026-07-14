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
    http:Client c = check new http:Client("http://testserver", {});

    // POST a byte[] body -> exercises listToBytes converting the Ballerina list
    // to raw octets. The server echoes the bytes back.
    byte[] payload = [72, 101, 108, 108, 111]; // "Hello"
    http:Response r = check c->post("/echo-bytes", payload);
    byte[] echoed = check r.getBinaryPayload();
    io:println(echoed.length()); // @output 5
    io:println(echoed[0]);       // @output 72
    io:println(echoed[4]);       // @output 111

    // GET a raw octet-stream response -> getBinaryPayload over a binary body.
    http:Response r2 = check c->get("/bytes");
    byte[] b = check r2.getBinaryPayload();
    io:println(b.length()); // @output 4
    io:println(b[0]);       // @output 1
    return;
}
