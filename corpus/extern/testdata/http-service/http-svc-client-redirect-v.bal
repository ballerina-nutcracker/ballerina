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

service /r on new http:Listener(19202) {
    resource function get redirect() returns http:Response {
        http:Response resp = new;
        resp.statusCode = 302;
        resp.setHeader("Location", "/r/done");
        return resp;
    }

    resource function get done() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("arrived");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19202", {
        followRedirects: {enabled: true, maxCount: 3}
    });
    http:Response r = check c->get("/r/redirect");
    io:println(r.statusCode); // @output 200
}
