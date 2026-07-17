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

// Every Request mutator (setHeader/addHeader/removeHeader/removeAllHeaders/
// setContentType/setTextPayload/setJsonPayload/setBinaryPayload) previously
// wasn't wired into the inbound Request object's dispatch table (only the
// read-only accessors were), so calling any of them on a service resource's
// injected http:Request panicked with a generic, undiagnosable 500.
service /reqmut on new http:Listener(19501) {
    resource function post headers(http:Request req) returns http:Response|error {
        req.setHeader("x-one", "a");
        req.addHeader("x-one", "b");
        req.setHeader("x-remove-me", "gone");
        req.removeHeader("x-remove-me");
        check req.setContentType("text/plain");
        check req.setTextPayload("mutated");

        string hasRemoveMe = "false";
        if req.hasHeader("x-remove-me") {
            hasRemoveMe = "true";
        }

        http:Response resp = new;
        resp.setHeader("x-echo-one", check req.getHeader("x-one"));
        resp.setHeader("x-echo-content-type", req.getContentType());
        resp.setHeader("x-has-remove-me", hasRemoveMe);
        resp.setTextPayload(check req.getTextPayload());
        return resp;
    }

    resource function post payloads(http:Request req) returns http:Response|error {
        req.removeAllHeaders();
        anydata countAfterRemove = req.getHeaderNames().length();
        check req.setJsonPayload({y: 7});
        map<json> j = <map<json>>check req.getJsonPayload();
        check req.setBinaryPayload("bytes".toBytes());
        byte[] b = check req.getBinaryPayload();

        anydata jsonY = j["y"];
        http:Response resp = new;
        resp.setHeader("x-count-after-remove", countAfterRemove.toString());
        resp.setHeader("x-json-y", jsonY.toString());
        resp.setTextPayload(check string:fromBytes(b));
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19501", {});

    http:Response r1 = check c->post("/reqmut/headers", "original");
    io:println(r1.statusCode); // @output 200
    io:println(check r1.getHeader("x-echo-one")); // @output a
    io:println(check r1.getHeader("x-echo-content-type")); // @output text/plain
    io:println(check r1.getHeader("x-has-remove-me")); // @output false
    io:println(check r1.getTextPayload()); // @output mutated

    http:Response r2 = check c->post("/reqmut/payloads", "ignored");
    io:println(check r2.getHeader("x-count-after-remove")); // @output 0
    io:println(check r2.getHeader("x-json-y")); // @output 7
    io:println(check r2.getTextPayload()); // @output bytes
}
