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

service /foo on new http:Listener(19209) {
    resource function get bar() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("matched foo/bar");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19209", {});

    http:Response ok = check c->get("/foo/bar");
    io:println(ok.statusCode); // @output 200
    io:println(ok.getTextPayload()); // @output matched foo/bar

    // These are not under the /foo attach point; matching must stop at a
    // path boundary rather than matching on a plain string prefix.
    http:Response noBoundary1 = check c->get("/foobar");
    io:println(noBoundary1.statusCode); // @output 404

    http:Response noBoundary2 = check c->get("/foobaz");
    io:println(noBoundary2.statusCode); // @output 404
}
