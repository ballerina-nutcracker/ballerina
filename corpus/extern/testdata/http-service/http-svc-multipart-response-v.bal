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

// An inbound Response (the one the runtime builds for a value returned by
// http:Client, as opposed to one created via `new http:Response()`) must
// support the exact same method set. getBodyParts() previously wasn't wired
// into the inbound object's dispatch table and panicked with "function not
// found: getBodyParts".
service /multiparts on new http:Listener(19502) {
    resource function get encode() returns http:Response|error {
        mime:Entity textPart = new;
        textPart.setText("server part");

        mime:Entity jsonPart = new;
        jsonPart.setJson({ok: true});

        http:Response resp = new;
        resp.setBodyParts([textPart, jsonPart]);
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19502", {});
    http:Response r = check c->get("/multiparts/encode");

    mime:Entity[] parts = check r.getBodyParts();
    io:println(parts.length()); // @output 2
    io:println(check parts[0].getText()); // @output server part
    map<json> j = <map<json>>check parts[1].getJson();
    io:println(j["ok"]); // @output true
}
