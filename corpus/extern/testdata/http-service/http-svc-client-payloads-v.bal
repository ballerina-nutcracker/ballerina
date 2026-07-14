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

service /data on new http:Listener(19200) {
    resource function get status() returns http:Response {
        return new;
    }

    resource function get jsondata() returns http:Response {
        http:Response resp = new;
        resp.setJsonPayload({message: "hello", count: 3});
        return resp;
    }

    resource function get html() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("<html><body>hello</body></html>", "text/html");
        return resp;
    }

    resource function get bytes() returns http:Response {
        http:Response resp = new;
        resp.setBinaryPayload([1, 2, 3, 4, 5, 6, 7, 8]);
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19200", {});

    http:Response statusResp = check c->get("/data/status");
    io:println(statusResp.statusCode); // @output 200

    http:Response jsonResp = check c->get("/data/jsondata");
    json payload = check jsonResp.getJsonPayload();
    map<json> obj = <map<json>>payload;
    io:println(obj["message"]); // @output hello
    io:println(obj["count"]); // @output 3

    http:Response textResp = check c->get("/data/html");
    string text = textResp.getTextPayload();
    io:println(text); // @output <html><body>hello</body></html>

    http:Response binaryResp = check c->get("/data/bytes");
    byte[] b = check binaryResp.getBinaryPayload();
    io:println(b); // @output [1,2,3,4,5,6,7,8]
}
