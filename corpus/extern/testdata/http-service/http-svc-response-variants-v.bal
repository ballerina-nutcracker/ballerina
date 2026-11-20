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

final http:Client backend = check new ("http://localhost:19203/resp");

service /resp on new http:Listener(19203) {
    // No return value: dispatch yields (), which the server maps to 202 Accepted.
    resource function post accepted() {
    }

    // Returning an error value is mapped to a 500 error response.
    resource function get failing() returns http:Response|error {
        return error("boom");
    }

    // Two values for the same header exercise the addHeader (Add) path. The
    // hop-by-hop Connection header must be dropped before reaching the client.
    resource function get headers() returns http:Response {
        http:Response resp = new;
        resp.addHeader("X-Multi", "a");
        resp.addHeader("X-Multi", "b");
        resp.setHeader("Connection", "keep-alive");
        resp.setTextPayload("ok");
        return resp;
    }

    // Each of these sets a Content-Length that its body does not match.
    resource function get noBody() returns http:Response {
        http:Response resp = new;
        resp.setHeader("Content-Length", "100");
        return resp;
    }

    resource function get short() returns http:Response {
        http:Response resp = new;
        resp.setHeader("Content-Length", "100");
        resp.setTextPayload("hello");
        return resp;
    }

    // A HEAD response sends the length a GET would, with no body.
    resource function head short() returns http:Response {
        http:Response resp = new;
        resp.setHeader("Content-Length", "5");
        return resp;
    }

    // Also answers HEAD, which gets the length of the body a GET would send.
    resource function default long() returns http:Response {
        http:Response resp = new;
        resp.setHeader("Content-Length", "3");
        resp.setTextPayload("hello world");
        return resp;
    }

    resource function default noContent() returns http:Response {
        http:Response resp = new;
        resp.statusCode = 204;
        resp.setHeader("Content-Length", "100");
        return resp;
    }

    resource function default notModified() returns http:Response {
        http:Response resp = new;
        resp.statusCode = 304;
        resp.setHeader("Content-Length", "100");
        resp.setTextPayload("hello");
        return resp;
    }

    resource function get hello() returns string => "hello";

    // The backend response carries Content-Length: 5, which the new payload no longer matches.
    resource function get replaced() returns http:Response|error {
        http:Response resp = check backend->get("/hello");
        resp.setTextPayload("replaced body");
        return resp;
    }

    resource function get digits() returns http:Response {
        string digits = "";
        foreach int _ in 0 ..< 2000 {
            digits += "0123456789";
        }
        http:Response resp = new;
        resp.setTextPayload(digits);
        return resp;
    }

    // The backend body is longer than the client buffers, so it is relayed as a stream.
    resource function get relayedDigits() returns http:Response|error {
        http:Response resp = check backend->get("/digits");
        resp.setHeader("Content-Length", "100");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new http:Client("http://localhost:19203", {});

    http:Response accepted = check c->post("/resp/accepted", "");
    io:println(accepted.statusCode); // @output 202

    http:Response failing = check c->get("/resp/failing");
    io:println(failing.statusCode); // @output 500

    http:Response headers = check c->get("/resp/headers");
    string[] multi = check headers.getHeaders("X-Multi");
    io:println(multi.length()); // @output 2
    io:println(headers.hasHeader("Connection")); // @output false

    // The Content-Length sent is the length of the body, whatever the service set, and none is
    // sent with a 204 or a relayed stream. jBallerina does this over HTTP/1.1 only; over HTTP/2
    // it sends the service's value, which the client rejects
    // (ballerina-platform/ballerina-library#9810). A bodyless HEAD response keeps the service's
    // value, which jBallerina sends as 0 over HTTP/1.1 (ballerina-platform/ballerina-library#9819).
    // @output 200 0 | HEAD 200 5 | 200 5 hello | 200 11 hello world | HEAD 200 11 | 204 false | HEAD 204 false | 304 | HEAD 304 false | replaced body | 20000 false
    io:println(check sentLengths(c));
    http:Client h1 = check new http:Client("http://localhost:19203", {httpVersion: http:HTTP_1_1});
    // @output 200 0 | HEAD 200 5 | 200 5 hello | 200 11 hello world | HEAD 200 11 | 204 false | HEAD 204 false | 304 | HEAD 304 false | replaced body | 20000 false
    io:println(check sentLengths(h1));
}

function sentLengths(http:Client cl) returns string|error {
    http:Response noBody = check cl->get("/resp/noBody");
    http:Response headShort = check cl->head("/resp/short");
    http:Response short = check cl->get("/resp/short");
    http:Response long = check cl->get("/resp/long");
    http:Response headLong = check cl->head("/resp/long");
    http:Response noContent = check cl->get("/resp/noContent");
    http:Response headNoContent = check cl->head("/resp/noContent");
    http:Response notModified = check cl->get("/resp/notModified");
    http:Response headNotModified = check cl->head("/resp/notModified");
    string replaced = check cl->get("/resp/replaced");
    http:Response relayed = check cl->get("/resp/relayedDigits");
    string noBodyLength = check noBody.getHeader("content-length");
    string headLength = check headShort.getHeader("content-length");
    string shortLength = check short.getHeader("content-length");
    string longLength = check long.getHeader("content-length");
    string headLongLength = check headLong.getHeader("content-length");
    return string `${noBody.statusCode} ${noBodyLength} | HEAD ${headShort.statusCode} ${headLength} | `
        + string `${short.statusCode} ${shortLength} ${check short.getTextPayload()} | `
        + string `${long.statusCode} ${longLength} ${check long.getTextPayload()} | HEAD ${headLong.statusCode} ${headLongLength} | `
        + string `${noContent.statusCode} ${noContent.hasHeader("content-length")} | `
        + string `HEAD ${headNoContent.statusCode} ${headNoContent.hasHeader("content-length")} | ${notModified.statusCode} | `
        + string `HEAD ${headNotModified.statusCode} ${headNotModified.hasHeader("content-length")} | ${replaced} | `
        + string `${(check relayed.getTextPayload()).length()} ${relayed.hasHeader("content-length")}`;
}
