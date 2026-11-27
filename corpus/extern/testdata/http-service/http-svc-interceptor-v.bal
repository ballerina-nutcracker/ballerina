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
import ballerina/lang.runtime;
import ballerina/mime;
import ballerina/time;

listener http:Listener ep = new (19242);

// One connection, so a backend response body left open blocks the next request until it times out.
final http:Client backend = check new ("http://127.0.0.1:19242/b",
    {httpVersion: http:HTTP_1_1, poolConfig: {maxActiveConnections: 1}, timeout: 5});

isolated service /b on ep {
    resource function get big() returns string {
        string text = "x";
        while text.length() < 65536 {
            text += text;
        }
        return text;
    }
}

// Stopped while it starts, so it never serves.
listener http:Listener stopping = new (19244);

service http:InterceptableService /s on stopping {
    public function createInterceptors() returns http:Interceptor {
        checkpanic stopping.immediateStop();
        return new Tagger("s");
    }

    resource function get a() returns string => "a";
}

// Records the stages a request passes and acts as the x-mode header asks when it names this tag.
service class Tagger {
    *http:RequestInterceptor;
    private final string tag;

    function init(string tag) {
        self.tag = tag;
    }

    resource function 'default [string... path](http:RequestContext ctx, @http:Header string? x\-mode)
            returns http:NextService|error? {
        string joined = "";
        foreach string segment in path {
            joined += "/" + segment;
        }
        io:println("interceptor ", self.tag, " ", joined);
        string[] seen = ctx.hasKey("seen") ? check ctx.getWithType("seen") : [];
        seen.push(self.tag);
        ctx.set("seen", seen);
        match x\-mode {
            "nil" => {
                if self.tag == "b" {
                    return;
                }
            }
            "nextThenNil" => {
                if self.tag == "b" {
                    _ = check ctx.next();
                    return;
                }
            }
            "error" => {
                if self.tag == "b" {
                    return error("boom from " + self.tag);
                }
            }
            "nextTwice" => {
                _ = check ctx.next();
                return ctx.next();
            }
            "self" => {
                if self.tag == "a" {
                    return self;
                }
            }
            "firstNext" => {
                if self.tag == "a" {
                    http:NextService? first = check ctx.next();
                    _ = check ctx.next();
                    return first;
                }
            }
            "panic" => {
                if self.tag == "b" {
                    _ = ctx.get("missing");
                }
            }
        }
        return ctx.next();
    }
}

service http:InterceptableService /x on ep {
    public function createInterceptors() returns http:Interceptor[] => [new Tagger("a"), new Tagger("b")];

    resource function get a(http:RequestContext ctx) returns json|error {
        string[] seen = check ctx.getWithType("seen");
        io:println("service /x/a");
        return {seen};
    }

    resource function post a(http:RequestContext ctx) returns string {
        var _ = ctx;
        return "posted";
    }

    resource function post echo(@http:Payload string payload) returns string => payload;

    resource function get next(http:RequestContext ctx) returns string {
        http:NextService|error? next = ctx.next();
        return next is error ? next.message() : "unexpected";
    }

    resource function get members(http:RequestContext ctx) returns string {
        int|error missing = ctx.getWithType("nope");
        int|error mismatch = ctx.getWithType("seen");
        return string `${missing is error ? missing.message() : ""}; ${mismatch is error ? mismatch.message() : ""}`;
    }
}

service class Answer {
    *http:RequestInterceptor;

    resource function get answer() returns string => "from interceptor";
}

service http:InterceptableService /single on ep {
    public function createInterceptors() returns http:Interceptor => new Answer();

    resource function get answer() returns string => "from service";

    resource function get other() returns string => "other";
}

// Changes the request, reads its body or answers it, as the x-mode header asks.
service class Mutator {
    *http:RequestInterceptor;

    resource function 'default [string... path](http:RequestContext ctx, http:Request req, http:Caller caller)
            returns http:NextService|error|anydata|http:Response|http:StatusCodeResponse {
        var _ = path;
        req.setHeader("x-added", "yes");
        string|error mode = req.getHeader("x-mode");
        if mode is string {
            match mode {
                "read" => {
                    io:println("interceptor read ", check req.getTextPayload());
                }
                "replace" => {
                    req.setTextPayload("replaced");
                }
                "empty" => {
                    req.setTextPayload("");
                }
                "json" => {
                    return {a: 1};
                }
                "response" => {
                    http:Response response = new;
                    response.statusCode = 299;
                    response.setTextPayload("response");
                    return response;
                }
                "created" => {
                    http:Created created = {body: "created"};
                    return created;
                }
                "caller" => {
                    check caller->respond("via caller");
                    return ctx.next();
                }
                "callerOnly" => {
                    check caller->respond("via caller only");
                    return;
                }
                "callerNextNil" => {
                    check caller->respond("via caller, then nil");
                    _ = check ctx.next();
                    return;
                }
                "parts" => {
                    mime:Entity part = new;
                    part.setText("part one");
                    req.setBodyParts([part], "multipart/mixed");
                }
            }
        }
        return ctx.next();
    }
}

service http:InterceptableService /m on ep {
    public function createInterceptors() returns http:Interceptor[] => [new Mutator()];

    resource function post payload(@http:Header string x\-added, @http:Payload string payload) returns string =>
        x\-added + ":" + payload;

    resource function post request(http:Request req) returns string|error =>
        check req.getHeader("x-added") + ":" + check req.getTextPayload() + ":" + check req.getHeader("content-length");

    resource function get header(@http:Header string? x\-added) returns string => x\-added ?: "none";

    resource function post parts(http:Request req) returns string|error {
        mime:Entity[] parts = check req.getBodyParts();
        return parts[0].getText();
    }

    resource function get big() returns http:Response|error {
        return backend->get("/big");
    }

    resource function get viaBackend() returns string|error {
        decimal started = time:monotonicNow();
        http:Response response = check backend->get("/big");
        int length = (check response.getTextPayload()).length();
        return string `${length} ${time:monotonicNow() - started < 2d ? "at once" : "after a wait"}`;
    }

    resource function get log() returns string {
        io:println("service /m/log");
        return "log";
    }

    resource function get crash() returns string {
        int[] values = [];
        return values[1].toString();
    }
}

service class Query {
    *http:RequestInterceptor;

    resource function get [string... path](int q, http:RequestContext ctx) returns http:NextService|error? {
        var _ = path;
        io:println("query ", q);
        return ctx.next();
    }
}

// Unlike jBallerina, a method-specific interceptor leaves OPTIONS and preflight requests to the
// service, and its responses follow the service configuration (ballerina-library#9886, #9887).
@http:ServiceConfig {cors: {allowOrigins: ["http://a.com"]}, mediaTypeSubtypePrefix: "vnd.test"}
service http:InterceptableService /q on ep {
    public function createInterceptors() returns http:Interceptor => new Query();

    resource function get a() returns json => {a: 1};

    resource function put a() returns string => "put";
}

service /plain on ep {
    resource function get a(http:RequestContext ctx) returns string {
        http:NextService|error? next = ctx.next();
        return next is error ? next.message() : "unexpected";
    }
}

service class LateService {
    *http:InterceptableService;

    public function createInterceptors() returns http:Interceptor => new Tagger("late");

    resource function get a(http:RequestContext ctx) returns json|error {
        string[] seen = check ctx.getWithType("seen");
        return {seen};
    }
}

service class Echo {
    *http:ResponseInterceptor;

    remote function interceptResponse(http:RequestContext ctx) returns http:NextService|error? {
        return ctx.next();
    }
}

service class WithResponseInterceptor {
    *http:InterceptableService;

    public function createInterceptors() returns http:Interceptor => new Echo();

    resource function get a() returns string => "a";
}

function show(http:Client c, string method, string path, map<string> headers = {}, string? body = ())
        returns error? {
    http:Response response = check c->execute(method, path, body, headers);
    string line = response.statusCode.toString();
    foreach string name in ["allow", "access-control-allow-origin", "access-control-allow-methods"] {
        string|error value = response.getHeader(name);
        if value is string {
            line += string ` ${name}=[${value}]`;
        }
    }
    string|error contentType = response.getHeader("content-type");
    string|error payload = response.getTextPayload();
    if payload is string {
        int? timestamp = payload.indexOf("\"timestamp\":\"");
        int? end = timestamp is int ? payload.indexOf("\",", timestamp + 13) : ();
        if timestamp is int && end is int {
            payload = payload.substring(0, timestamp) + payload.substring(end + 2);
        }
    }
    if payload is string && payload != "" {
        line += string ` ${contentType is string ? contentType : ""} ${payload}`;
    }
    io:println(line);
}

public function testMain() returns error? {
    http:RequestContext ctx = new;
    ctx.set("k", 1);
    io:println(ctx.get("k"), " ", ctx.hasKey("k"), " ", ctx.keys());
    ctx.remove("k");
    http:NextService|error? next = ctx.next();
    io:println(ctx.hasKey("k"), " ", next is error ? next.message() : "unexpected");
    // @output 1 true ["k"]
    // @output false request context object does not contain the configured interceptors
    int[] initial = [1];
    ctx.set("list", initial);
    int[] copy = check ctx.getWithType("list");
    copy.push(2);
    // jBallerina returns the member itself: https://github.com/ballerina-platform/ballerina-library/issues/9915
    io:println(ctx.get("list")); // @output [1]

    http:Client c = check new ("http://127.0.0.1:19242");
    check show(c, "GET", "/x/a");
    // @output interceptor a /a
    // @output interceptor b /a
    // @output service /x/a
    // @output 200 application/json {"seen":["a","b"]}
    check show(c, "GET", "/x/next");
    // @output interceptor a /next
    // @output interceptor b /next
    // @output 200 text/plain no next service to be returned
    check show(c, "GET", "/x/members");
    // @output interceptor a /members
    // @output interceptor b /members
    // @output 200 text/plain no member found for key: nope; type conversion failed for value of key: seen
    check show(c, "GET", "/x/missing");
    // @output interceptor a /missing
    // @output interceptor b /missing
    // @output 404 application/json {"status":404,"reason":"Not Found","message":"no matching resource found for path : /x/missing , method : GET","path":"/x/missing","method":"GET"}
    check show(c, "PUT", "/x/a");
    // @output interceptor a /a
    // @output interceptor b /a
    // @output 405 application/json {"status":405,"reason":"Method Not Allowed","message":"Method not allowed","path":"/x/a","method":"PUT"}
    check show(c, "POST", "/x/echo", body = "hi");
    // @output interceptor a /echo
    // @output interceptor b /echo
    // @output 201 text/plain hi
    check show(c, "POST", "/x/a", body = "x");
    // @output interceptor a /a
    // @output interceptor b /a
    // @output 201 text/plain posted
    check show(c, "GET", "/x/a", {"x-mode": "nil"});
    // @output interceptor a /a
    // @output interceptor b /a
    // @output 202
    // jBallerina answers 202 and still runs the service (ballerina-library#9884).
    check show(c, "GET", "/x/a", {"x-mode": "nextThenNil"});
    // @output interceptor a /a
    // @output interceptor b /a
    // @output 202
    check show(c, "GET", "/x/a", {"x-mode": "error"});
    // @output interceptor a /a
    // @output interceptor b /a
    // @output 500 application/json {"status":500,"reason":"Internal Server Error","message":"boom from b","path":"/x/a","method":"GET"}
    check show(c, "GET", "/x/a", {"x-mode": "nextTwice"});
    // @output interceptor a /a
    // @output service /x/a
    // @output 200 application/json {"seen":["a"]}
    check show(c, "GET", "/x/a", {"x-mode": "self"});
    // @output interceptor a /a
    // @output 500 application/json {"status":500,"reason":"Internal Server Error","message":"next interceptor service did not match with the configuration","path":"/x/a","method":"GET"}
    check show(c, "GET", "/x/a", {"x-mode": "firstNext"});
    // @output interceptor a /a
    // @output 500 application/json {"status":500,"reason":"Internal Server Error","message":"target service did not match with the configuration","path":"/x/a","method":"GET"}
    check show(c, "GET", "/x/a", {"x-mode": "panic"});
    // @output interceptor a /a
    // @output interceptor b /a
    // @output 500 text/plain missing map key 'missing'

    check show(c, "GET", "/single/answer");
    // @output 200 text/plain from interceptor
    check show(c, "GET", "/single/other");
    // @output 200 text/plain other

    check show(c, "POST", "/m/payload", body = "hello");
    // @output 201 text/plain yes:hello
    check show(c, "POST", "/m/request", body = "hello");
    // @output 201 text/plain yes:hello:5
    check show(c, "GET", "/m/header");
    // @output 200 text/plain yes
    check show(c, "POST", "/m/payload", {"x-mode": "read"}, "hello");
    // @output interceptor read hello
    // @output 201 text/plain yes:hello
    check show(c, "POST", "/m/request", {"x-mode": "read"}, "hello");
    // @output interceptor read hello
    // @output 201 text/plain yes:hello:5
    check show(c, "POST", "/m/payload", {"x-mode": "replace"}, "hello");
    // @output 201 text/plain yes:replaced
    check show(c, "POST", "/m/request", {"x-mode": "replace"}, "hello");
    // @output 201 text/plain yes:replaced:8
    check show(c, "POST", "/m/request", {"x-mode": "empty"}, "hello");
    // @output 201 text/plain yes::0
    check show(c, "POST", "/m/parts", {"x-mode": "parts"}, "hello");
    // @output 201 text/plain part one
    check show(c, "GET", "/m/crash");
    // @output 500 text/plain invalid array index: 1
    check show(c, "POST", "/m/payload", {"x-mode": "json"}, "hello");
    // @output 200 application/json {"a":1}
    check show(c, "GET", "/m/header", {"x-mode": "response"});
    // @output 299 text/plain response
    check show(c, "GET", "/m/header", {"x-mode": "created"});
    // @output 201 text/plain created
    check show(c, "GET", "/m/header", {"x-mode": "caller"});
    // @output 200 text/plain via caller
    check show(c, "GET", "/m/header", {"x-mode": "callerOnly"});
    // @output 200 text/plain via caller only
    check show(c, "GET", "/m/log", {"x-mode": "callerNextNil"});
    // @output 200 text/plain via caller, then nil
    check show(c, "POST", "/m/payload", {"x-mode": "caller"}, "hello");
    // @output 200 text/plain via caller
    // The streamed response the resource returns is not sent, and its body is closed.
    check show(c, "GET", "/m/big", {"x-mode": "caller"});
    // @output 200 text/plain via caller
    // Lets the resource return the backend response after the interceptor's response is sent.
    runtime:sleep(0.5);
    check show(c, "GET", "/m/viaBackend");
    // @output 200 text/plain 65536 at once

    http:Client stopped = check new ("http://127.0.0.1:19244", {timeout: 1});
    http:Response|error refused = stopped->get("/s/a");
    io:println(refused is error ? "not served" : refused.statusCode); // @output not served

    // jBallerina never answers when an interceptor parameter cannot be bound (ballerina-library#9885).
    check show(c, "GET", "/q/a");
    // @output 400 application/vnd.test+json {"status":400,"reason":"Bad Request","message":"no query param value found for 'q'","path":"/q/a","method":"GET"}
    check show(c, "GET", "/q/a?q=1", {Origin: "http://a.com"});
    // @output query 1
    // @output 200 access-control-allow-origin=[http://a.com] application/vnd.test+json {"a":1}
    check show(c, "OPTIONS", "/q/a");
    // @output 204 allow=[GET, PUT, OPTIONS]
    check show(c, "OPTIONS", "/q/a", {Origin: "http://a.com", "Access-Control-Request-Method": "PUT"});
    // @output 204 access-control-allow-origin=[http://a.com] access-control-allow-methods=[PUT]

    check show(c, "GET", "/plain/a");
    // @output 200 text/plain no next service to be returned

    // jBallerina creates no interceptors for a service attached after the listener started
    // (ballerina-library#9585).
    check ep.attach(new LateService(), "/late");
    check show(c, "GET", "/late/a");
    // @output interceptor late /a
    // @output 200 application/json {"seen":["late"]}
    error? attached = ep.attach(new WithResponseInterceptor(), "/response");
    io:println(attached is error ? attached.message() : "attached");
    // @output interceptor service registration failed: response and error interceptors are not yet supported
}
