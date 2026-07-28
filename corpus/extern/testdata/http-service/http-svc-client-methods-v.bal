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

service /verb on new http:Listener(19201) {
    resource function post post() returns http:Response {
        return new;
    }

    resource function put put() returns http:Response {
        return new;
    }

    resource function delete delete() returns http:Response {
        return new;
    }

    resource function patch patch() returns http:Response {
        return new;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19201", {});

    http:Response r1 = check c->post("/verb/post", "hello");
    io:println(r1.statusCode); // @output 200

    http:Response r2 = check c->put("/verb/put", "world");
    io:println(r2.statusCode); // @output 200

    http:Response r3 = check c->delete("/verb/delete");
    io:println(r3.statusCode); // @output 200

    http:Response r4 = check c->patch("/verb/patch", {"key": "value"});
    io:println(r4.statusCode); // @output 200
}
