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
import ballerina/mime;
import ballerina/io;

public function main() returns error? {
    http:Client c = check new http:Client("http://testserver", {});

    mime:Entity textPart = new;
    textPart.setText("hello world");

    mime:Entity jsonPart = new;
    jsonPart.setJson({x: 1});

    // POST a multipart body (mime:Entity[] via the widened RequestMessage) — exercises
    // msgToBody's Entity[] branch, which serializes via mime's EncodeMultipart and
    // auto-generates a boundary. The server parses the multipart body itself and echoes
    // a deterministic, boundary-independent summary back.
    http:Response r = check c->post("/echo-multipart", [textPart, jsonPart]);
    io:println(check r.getTextPayload()); // @output count=2 1:hello world 2:{"x":1}
    return;
}
