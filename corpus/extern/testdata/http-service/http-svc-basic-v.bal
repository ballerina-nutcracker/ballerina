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

service /hello on new http:Listener(19190) {
    resource function get greeting() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("Hello, World!");
        return resp;
    }
}

// testMain is invoked by the harness while the runtime is parked in the
// listening state. It drives the live service over a real HTTP client.
public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19190", {});
    http:Response r = check c->get("/hello/greeting");
    io:println(r.statusCode); // @output 200
    io:println(r.getTextPayload()); // @output Hello, World!
}
