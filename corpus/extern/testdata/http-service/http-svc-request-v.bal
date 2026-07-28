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

service /echo on new http:Listener(19192) {
    resource function post msg(http:Request req) returns http:Response|error {
        json payload = check req.getJsonPayload();
        http:Response resp = new;
        resp.setJsonPayload(payload);
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19192", {});
    json body = {name: "ballerina", version: 1};
    http:Response r = check c->post("/echo/msg", body);
    io:println(r.statusCode); // @output 200
    // Read the echoed JSON back as text (deterministic key order from the
    // server's json.Marshal).
    io:println(r.getTextPayload()); // @output {"name":"ballerina","version":1}
}
