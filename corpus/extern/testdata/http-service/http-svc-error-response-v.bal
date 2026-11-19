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
import ballerina/http.httpscerr;
import ballerina/io;

service /errors on new http:Listener(19235) {
    resource function get plain() returns string|error => error("plain failure");

    resource function get notFound() returns string|error => error httpscerr:NotFoundError("no album");

    // The detail values are bound first because an error constructor needs them readonly here (#1374).
    resource function get withBody() returns string|error {
        map<string> & readonly body = {code: "C1", detail: "taken"};
        map<string|string[]> & readonly headers = {"x-trace": "t1", "x-multi": ["a", "b"]};
        return error httpscerr:ConflictError("conflict", body = body, headers = headers);
    }

    resource function get textBody() returns string|error => error httpscerr:BadRequestError("bad", body = "bad input");

    resource function get headersOnly() returns string|error {
        map<string|string[]> & readonly headers = {"x-reason": "policy"};
        return error httpscerr:ForbiddenError("forbidden", headers = headers);
    }

    resource function get teapot() returns string|error => error httpscerr:DefaultStatusCodeError("teapot", statusCode = 418);

    resource function get viaCause() returns string|error => error httpscerr:DefaultStatusCodeError("wrapped", error httpscerr:GoneError("gone"));

    resource function get noCode() returns string|error => error httpscerr:DefaultStatusCodeError("no code");

    resource function get legal() returns string|error => error httpscerr:UnavailableDueToLegalReasonsError("legal");

    resource function get tooLong() returns string|error => error httpscerr:URITooLongError("long");

    resource function get httpError() returns string|error => error http:GenericClientError("client side");

    resource function get appError() returns string|error {
        map<string[]> & readonly headers = {"x-upstream": ["u1"]};
        return error http:ClientRequestError("gone upstream", statusCode = 410, headers = headers, body = "bye");
    }

    resource function get appErrorNoBody() returns string|error {
        map<string[]> & readonly headers = {};
        return error http:RemoteServerError("upstream down", statusCode = 503, headers = headers, body = ());
    }

    resource function get tooEarly() returns string|error => error httpscerr:DefaultStatusCodeError("early", statusCode = 425);

    resource function get server599() returns string|error => error httpscerr:DefaultStatusCodeError("odd", statusCode = 599);

    resource function get code650() returns string|error => error httpscerr:DefaultStatusCodeError("big", statusCode = 650);

    resource function get problem() returns string|error {
        map<json> & readonly body = {title: "taken"};
        map<string[]> & readonly headers = {"Content-Type": ["application/problem+json"]};
        return error httpscerr:ConflictError("conflict", body = body, headers = headers);
    }

    resource function get problemNoBody() returns string|error {
        map<string[]> & readonly headers = {"Content-Type": ["application/problem+json"]};
        return error httpscerr:ConflictError("conflict", headers = headers);
    }

    resource function get appErrorCause() returns string|error {
        map<string[]> & readonly headers = {};
        error cause = error http:ClientRequestError("rejected", statusCode = 422, headers = headers, body = ());
        return error httpscerr:DefaultStatusCodeError("wrapped", cause);
    }

    resource function get onlyGet() returns string => "x";
}

// Reports whether the ErrorPayload members appear in jBallerina's order in the raw body.
function inOrder(string text) returns boolean {
    int previous = -1;
    foreach string key in ["timestamp", "status", "reason", "message", "path", "method"] {
        int? index = text.indexOf("\"" + key + "\":");
        if index is () || index <= previous {
            return false;
        }
        previous = index;
    }
    return true;
}

function show(http:Client c, string path, string method = "GET") returns error? {
    http:Response response = check c->execute(method, path, "");
    string|error contentType = response.getHeader("content-type");
    io:println(response.statusCode, " ", contentType is string ? contentType : "-");
    if contentType != "application/json" {
        io:println(check response.getTextPayload());
        return;
    }
    string text = check response.getTextPayload();
    json payload = check response.getJsonPayload();
    if payload is map<json> && payload.hasKey("timestamp") {
        string timestamp = (check payload.timestamp).toString();
        io:println(inOrder(text), " ", timestamp.length() >= 20 && timestamp.substring(timestamp.length() - 1) == "Z");
        io:println(payload.status, " | ", payload.reason, " | ", payload.message, " | ", payload.path, " | ", payload.method);
        return;
    }
    io:println(text);
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19235");

    check show(c, "/errors/plain?x=1");
    // @output 500 application/json
    // @output true true
    // @output 500 | Internal Server Error | plain failure | /errors/plain?x=1 | GET
    check show(c, "/errors/notFound");
    // @output 404 application/json
    // @output true true
    // @output 404 | Not Found | no album | /errors/notFound | GET
    check show(c, "/errors/withBody");
    // @output 409 application/json
    // @output {"code":"C1","detail":"taken"}
    http:Response withBody = check c->get("/errors/withBody");
    io:println(check withBody.getHeader("x-trace"), " ", check withBody.getHeaders("x-multi")); // @output t1 ["a","b"]
    check show(c, "/errors/textBody");
    // @output 400 text/plain
    // @output bad input
    check show(c, "/errors/headersOnly");
    // @output 403 application/json
    // @output true true
    // @output 403 | Forbidden | forbidden | /errors/headersOnly | GET
    http:Response headersOnly = check c->get("/errors/headersOnly");
    io:println(check headersOnly.getHeader("x-reason")); // @output policy
    check show(c, "/errors/teapot");
    // @output 418 application/json
    // @output true true
    // @output 418 | Client Error (418) | teapot | /errors/teapot | GET
    check show(c, "/errors/viaCause");
    // @output 410 application/json
    // @output true true
    // @output 410 | Gone | wrapped | /errors/viaCause | GET
    check show(c, "/errors/noCode");
    // @output 500 application/json
    // @output true true
    // @output 500 | Internal Server Error | no code | /errors/noCode | GET
    check show(c, "/errors/legal");
    // @output 451 application/json
    // @output true true
    // @output 451 | Unavailable For Legal Reasons | legal | /errors/legal | GET
    check show(c, "/errors/tooLong");
    // @output 414 application/json
    // @output true true
    // @output 414 | Request-URI Too Long | long | /errors/tooLong | GET
    check show(c, "/errors/httpError");
    // @output 500 application/json
    // @output true true
    // @output 500 | Internal Server Error | client side | /errors/httpError | GET
    check show(c, "/errors/appError");
    // @output 410 text/plain
    // @output bye
    http:Response appError = check c->get("/errors/appError");
    io:println(check appError.getHeader("x-upstream")); // @output u1
    check show(c, "/errors/appErrorNoBody");
    // @output 503 application/json
    // @output true true
    // @output 503 | Service Unavailable | upstream down | /errors/appErrorNoBody | GET
    http:Response problem = check c->get("/errors/problem");
    io:println(problem.statusCode, " ", check problem.getHeaders("content-type"), " ", check problem.getTextPayload());
    // @output 409 ["application/problem+json"] {"title":"taken"}
    http:Response problemNoBody = check c->get("/errors/problemNoBody");
    json noBodyPayload = check problemNoBody.getJsonPayload();
    io:println(problemNoBody.statusCode, " ", check problemNoBody.getHeaders("content-type"), " ", check noBodyPayload.message);
    // @output 409 ["application/problem+json"] conflict
    check show(c, "/errors/appErrorCause");
    // @output 500 application/json
    // @output true true
    // @output 500 | Internal Server Error | wrapped | /errors/appErrorCause | GET
    check show(c, "/errors/tooEarly");
    // @output 425 application/json
    // @output true true
    // @output 425 | Unordered Collection | early | /errors/tooEarly | GET
    check show(c, "/errors/server599");
    // @output 599 application/json
    // @output true true
    // @output 599 | Server Error (599) | odd | /errors/server599 | GET
    check show(c, "/errors/code650");
    // @output 650 application/json
    // @output true true
    // @output 650 | Internal Server Error | big | /errors/code650 | GET
    check show(c, "/errors/nope?x=1");
    // @output 404 application/json
    // @output true true
    // @output 404 | Not Found | no matching resource found for path : /errors/nope?x=1 , method : GET | /errors/nope?x=1 | GET
    check show(c, "/other");
    // @output 404 application/json
    // @output true true
    // @output 404 | Not Found | no matching service found for path: /other | /other | GET
    check show(c, "/errors/onlyGet", "POST");
    // @output 405 application/json
    // @output true true
    // @output 405 | Method Not Allowed | Method not allowed | /errors/onlyGet | POST
}
