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

service class FooService {
    resource function get bar() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("matched foo/bar");
        return resp;
    }
}

public function testMain() returns error? {
    http:Listener l = check new http:Listener(19211);

    // A programmatically supplied base path with a trailing slash must still
    // match sub-paths at a segment boundary, the same as "/foo" would.
    check l.attach(new FooService(), "/foo/");
    check l.'start();

    http:Client c = check new http:Client("http://localhost:19211", {});
    http:Response ok = check c->get("/foo/bar");
    io:println(ok.statusCode); // @output 200
    io:println(check ok.getTextPayload()); // @output matched foo/bar

    check l.gracefulStop();
}
