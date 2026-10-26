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

    resource function post relay(http:Request req) returns http:Response|error {
        mime:Entity p = new;
        p.setText("forwarded part");
        req.setBodyParts([p], "multipart/mixed");
        http:Response resp = check upstream->forward("/multiparts/echo", req);
        resp.setHeader("x-relayed-content-type", req.getContentType());
        return resp;
    }

    resource function post count(http:Request req) returns string|error {
        mime:Entity[] parts = check req.getBodyParts();
        return parts.length().toString();
    }

    resource function post edit(http:Request req) returns http:Response|error {
        mime:Entity[] parts = check req.getBodyParts();
        parts[0].setText("edited part");
        mime:Entity[] again = check req.getBodyParts();
        http:Response resp = check upstream->forward("/multiparts/echo", req);
        resp.setHeader("x-same-parts", (parts === again).toString());
        return resp;
    }

    resource function post echo(http:Request req) returns string|error {
        mime:Entity[] parts = check req.getBodyParts();
        return string `${req.getContentType()} count=${parts.length()} 1:${check parts[0].getText()}`;
    }
}

final http:Client upstream = check new ("http://localhost:19500");

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
    io:println(check r.getTextPayload()); // @output {"count":2,"disposition":"form-data;name=\"field1\"","text":"hello world","x":42}

    r = check c->post("/multiparts/relay", [textPart]);
    io:println(r.statusCode); // @output 201
    string echoed = check r.getTextPayload();
    io:println(echoed.indexOf("multipart/mixed; boundary=") == 0); // @output true
    io:println(echoed.includes(" count=1 1:forwarded part")); // @output true
    string relayedContentType = check r.getHeader("x-relayed-content-type");
    io:println(echoed.indexOf(relayedContentType + " ") == 0); // @output true

    http:Request noPart = new;
    noPart.setTextPayload("garbage", "multipart/form-data; boundary=zzz");
    r = check c->post("/multiparts/count", noPart);
    io:println(r.statusCode); // @output 500

    http:Request emptyBody = new;
    emptyBody.setTextPayload("", "multipart/form-data; boundary=zzz");
    r = check c->post("/multiparts/count", emptyBody);
    io:println(check r.getTextPayload()); // @output 0

    mime:Entity secondPart = new;
    secondPart.setText("second part");
    r = check c->post("/multiparts/edit", [textPart, secondPart]);
    io:println(check r.getHeader("x-same-parts")); // @output true
    string edited = check r.getTextPayload();
    io:println(edited.includes(" count=2 1:edited part")); // @output true
}
