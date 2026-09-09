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

// An inbound Request (the one the runtime builds for a service resource, as
// opposed to one created via `new http:Request()`) must support the exact
// same method set. getBodyParts() previously wasn't wired into the inbound
// object's dispatch table and panicked with "function not found: getBodyParts".
service /multiparts on new http:Listener(19500) {
    resource function post decode(http:Request req) returns http:Response|error {
        mime:Entity[] parts = check req.getBodyParts();
        string firstText = check parts[0].getText();
        string disposition = check parts[0].getHeader("content-disposition");
        map<json> secondJson = <map<json>>check parts[1].getJson();
        json result = {count: parts.length(), text: firstText, disposition: disposition, x: secondJson["x"]};

        http:Response resp = new;
        resp.setJsonPayload(result);
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19500", {});

    mime:Entity textPart = new;
    textPart.setText("hello world");
    mime:ContentDisposition disposition = check mime:getContentDispositionObject("form-data; name=\"field1\"");
    textPart.setContentDisposition(disposition);

    mime:Entity jsonPart = new;
    jsonPart.setJson({x: 42});

    http:Response r = check c->post("/multiparts/decode", [textPart, jsonPart]);
    io:println(r.statusCode); // @output 200
    io:println(check r.getTextPayload()); // @output {"count":2,"disposition":"form-data; name=field1","text":"hello world","x":42}
}
