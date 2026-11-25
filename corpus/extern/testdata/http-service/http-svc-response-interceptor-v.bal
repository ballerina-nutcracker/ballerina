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
import ballerina/lang.runtime;

listener http:Listener ep = new (19243);

function mode(http:Request req) returns string {
    string|error value = req.getHeader("x-mode");
    return value is string ? value : "";
}

// The modes of the response error interceptor test it on a failing resource.
function fails(string mode) returns boolean => mode == "fail" || mode.indexOf("reserr") is int;

// Marks the responses it sees.
service class Marker {
    *http:ResponseInterceptor;
    private final string name;

    function init(string name) {
        self.name = name;
    }

    remote function interceptResponse(http:RequestContext ctx, http:Response res) returns http:NextService|error? {
        res.addHeader("x-trail", self.name);
        return ctx.next();
    }
}

// Acts on the response as the x-mode header asks.
service class Mid {
    *http:ResponseInterceptor;

    remote function interceptResponse(http:Request req, http:Response res, http:Caller caller, http:RequestContext ctx)
            returns http:NextService|anydata|http:Response|http:StatusCodeResponse|error {
        res.addHeader("x-trail", "mid");
        match mode(req) {
            "mid-nil" => {
                return;
            }
            "mid-replace" => {
                return "replaced";
            }
            "mid-status" => {
                http:Accepted accepted = {body: "accepted"};
                return accepted;
            }
            "mid-error" => {
                return error("mid failed");
            }
            "mid-skip" => {
                _ = check ctx.next();
                return ctx.next();
            }
            "mid-exhaust" => {
                http:NextService|error? next = ctx.next();
                _ = check ctx.next();
                return next;
            }
            "mid-caller" => {
                check caller->respond("via mid caller");
                return ctx.next();
            }
            "mid-panic" => {
                int[] values = [];
                return values[1];
            }
            "mid-tag" => {
                string tag = check ctx.getWithType("tag");
                res.addHeader("x-trail", tag);
            }
        }
        return ctx.next();
    }
}

// Handles the errors of the response path as the x-mode header asks.
service class ResErr {
    *http:ResponseErrorInterceptor;

    remote function interceptResponseError(error err, http:Response res, http:Request req, http:RequestContext ctx)
            returns http:NextService|http:Response|error? {
        res.addHeader("x-trail", "reserr");
        match mode(req) {
            "reserr-next"|"conflict" => {
                return ctx.next();
            }
            "reserr-fail" => {
                return error("reserr failed: " + err.message());
            }
            "reserr-nil" => {
                return;
            }
        }
        res.setTextPayload("reserr: " + err.message());
        return res;
    }
}

// Tags the request, or fails or answers it, as the x-mode header asks.
service class ReqTag {
    *http:RequestInterceptor;

    resource function 'default [string... path](http:RequestContext ctx, http:Request req, @http:Query int? n)
            returns http:NextService|string|error? {
        var _ = path;
        var _ = n;
        ctx.set("tag", "tagged");
        match mode(req) {
            "req-fail" => {
                return error("req failed");
            }
            "req-respond" => {
                return "from reqtag";
            }
        }
        return ctx.next();
    }
}

// Handles the errors of the request path: logs them and resumes the request, or answers it.
service class ReqErr {
    *http:RequestErrorInterceptor;

    resource function 'default [string... path](error err, http:Request req, http:RequestContext ctx)
            returns http:NextService|string|error? {
        var _ = path;
        io:println("reqerr ", err.message(), " ", err is http:QueryParameterBindingError);
        if mode(req) == "reqerr-respond" {
            return "from reqerr";
        }
        return ctx.next();
    }
}

// The pipeline, with the default error interceptor before it at position 0:
// 1 outer, 2 reserr, 3 reqtag, 4 mid, 5 reqerr, 6 last.
service http:InterceptableService /r on ep {
    public function createInterceptors() returns http:Interceptor[] {
        return [new Marker("outer"), new ResErr(), new ReqTag(), new Mid(), new ReqErr(), new Marker("last")];
    }

    resource function get hello(http:Request req) returns string|error {
        if fails(mode(req)) {
            return error("resource failed");
        }
        match mode(req) {
            "conflict" => {
                // The detail is bound first because an error constructor needs it readonly here (#1374).
                map<string> & readonly body = {code: "C1"};
                return error httpscerr:ConflictError("conflict", body = body);
            }
        }
        return "hello";
    }

    resource function get caller(http:Caller caller) returns error? {
        check caller->respond("via caller");
    }

    resource function get 'continue(http:Caller caller) returns error? {
        check caller->'continue();
        check caller->respond("after continue");
    }

    resource function post echo(@http:Payload int x) returns int => x;
}

// Without error interceptors, an error skips the response interceptors for the default error
// response.
service http:InterceptableService /d on ep {
    public function createInterceptors() returns http:Interceptor[] => [new Marker("outer"), new ReqTag()];

    resource function get a() returns string => "a";
}

// Events recorded after a response was sent, which the test waits for.
isolated string[] events = [];

isolated function note(string event) {
    lock {
        events.push(event);
    }
}

isolated function takeEvents(int count) returns string[]? {
    lock {
        if events.length() < count {
            return ();
        }
        string[] taken = events.clone();
        events.removeAll();
        return taken.clone();
    }
}

function awaitEvents(int count) returns error? {
    foreach int _ in 0 ..< 500 {
        string[]? taken = takeEvents(count);
        if taken is string[] {
            io:println(taken);
            return;
        }
        runtime:sleep(0.01);
    }
    return error("timed out waiting for events");
}

// Responds through its http:Caller as the x-mode header asks.
service class CallerReq {
    *http:RequestInterceptor;

    resource function 'default [string... path](http:Caller caller, http:RequestContext ctx, http:Request req)
            returns http:NextService|error? {
        var _ = path;
        match mode(req) {
            "respond-next" => {
                check caller->respond("early");
                http:NextService|error? next = ctx.next();
                note("next " + (next is error ? next.message() : "ok"));
                return next;
            }
            "respond-error" => {
                check caller->respond("early");
                return error("late");
            }
            "fail" => {
                return error("req failed");
            }
        }
        return ctx.next();
    }
}

// Handles an error through its http:Caller for the caller-fail mode.
service class CallerResErr {
    *http:ResponseErrorInterceptor;

    remote function interceptResponseError(error err, http:Caller caller, http:Response res, http:Request req,
            http:RequestContext ctx) returns http:NextService|error? {
        res.addHeader("x-trail", "callerreserr");
        if mode(req) == "caller-fail" {
            return caller->respond("handled by caller: " + err.message());
        }
        return ctx.next();
    }
}

// Runs ctx.next() past the first stage and then fails, for the exhaust-error mode.
service class Exhaust {
    *http:ResponseInterceptor;

    remote function interceptResponse(http:RequestContext ctx, http:Response res, http:Request req)
            returns http:NextService|error? {
        res.addHeader("x-trail", "exhaust");
        if mode(req) == "exhaust-error" {
            _ = check ctx.next();
            _ = check ctx.next();
            return error("after exhaust");
        }
        return ctx.next();
    }
}

service class ReqLog {
    *http:RequestErrorInterceptor;

    resource function 'default [string... path](error err, http:RequestContext ctx) returns http:NextService|error? {
        var _ = path;
        io:println("reqlog ", err.message());
        note("reqlog " + err.message());
        return ctx.next();
    }
}

// The pipeline: 1 outer, 2 callerreserr, 3 callerreq, 4 exhaust, 5 reqlog.
service http:InterceptableService /c on ep {
    public function createInterceptors() returns http:Interceptor[] =>
        [new Marker("outer"), new CallerResErr(), new CallerReq(), new Exhaust(), new ReqLog()];

    resource function get a(http:Caller caller, http:Request req) returns error? {
        if mode(req) == "caller-fail" {
            return error("caller boom");
        }
        error? sent = caller->respond("a");
        note("service " + (sent is error ? sent.message() : "sent"));
    }

    resource function get b() returns string => "b";

    resource function get err(error e) returns string => "resource error: " + e.message();
}

// A request interceptor binds http:Caller; the resource does not.
service http:InterceptableService /q on ep {
    public function createInterceptors() returns http:Interceptor[] => [new Marker("outer"), new CallerReq(), new Mid()];

    resource function get hello() returns string => "hello";
}

function show(http:Client c, string method, string path, string mode = "", string? body = ()) returns error? {
    map<string> headers = mode == "" ? {} : {"x-mode": mode};
    if body is string {
        headers["content-type"] = "application/json";
    }
    http:Response response = check c->execute(method, path, body, headers);
    string line = response.statusCode.toString();
    string[]|error trail = response.getHeaders("x-trail");
    // Not `trail is string[]`, which the list getHeaders returns fails (#1779).
    if trail !is error {
        line += " [";
        foreach int i in 0 ..< trail.length() {
            line += (i == 0 ? "" : ",") + trail[i];
        }
        line += "]";
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
    http:Client c = check new ("http://127.0.0.1:19243");

    // Response interceptors run from the last towards the first.
    check show(c, "GET", "/r/hello");
    // @output 200 [last,mid,outer] text/plain hello
    check show(c, "GET", "/r/caller");
    // @output 200 [last,mid,outer] text/plain via caller
    check show(c, "POST", "/r/echo", body = "5");
    // @output 201 [last,mid,outer] application/json 5
    check show(c, "GET", "/r/hello", "mid-tag");
    // @output 200 [last,mid,tagged,outer] text/plain hello

    // A value a response interceptor returns replaces the response, which continues towards the first.
    check show(c, "GET", "/r/hello", "mid-nil");
    // @output 202 [outer]
    check show(c, "GET", "/r/hello", "mid-replace");
    // @output 200 [outer] text/plain replaced
    check show(c, "GET", "/r/hello", "mid-status");
    // @output 202 [outer] text/plain accepted
    // jBallerina sends 500 when the resource, or a request interceptor before it, has an http:Caller
    // parameter (ballerina-library#9902).
    check show(c, "GET", "/r/caller", "mid-nil");
    // @output 202 [outer]
    check show(c, "GET", "/q/hello", "mid-nil");
    // @output 202 [outer]

    // Once ctx.next() passed the first response interceptor, () sends the response as it is.
    check show(c, "GET", "/r/hello", "mid-skip");
    // @output 200 [last,mid] text/plain hello
    // jBallerina panics and exits for a service returned after that (ballerina-library#9588).
    check show(c, "GET", "/r/hello", "mid-exhaust");
    // @output 500 [reserr,outer] text/plain reserr: next interceptor service did not match with the configuration

    // A response sent through the http:Caller of a response interceptor goes on from it.
    check show(c, "GET", "/r/hello", "mid-caller");
    // @output 200 [outer] text/plain via mid caller
    check show(c, "GET", "/r/hello", "mid-panic");
    // @output 500 text/plain invalid array index: 1

    // jBallerina runs the response interceptors on the 100 Continue rather than on the final
    // response (ballerina-library#9903).
    check show(c, "GET", "/r/continue");
    // @output 200 [last,mid,outer] text/plain after continue

    // Errors skip the response interceptors for the nearest response error interceptor.
    check show(c, "GET", "/r/hello", "fail");
    // @output 500 [reserr,outer] text/plain reserr: resource failed
    check show(c, "GET", "/r/hello", "mid-error");
    // @output 500 [reserr,outer] text/plain reserr: mid failed
    check show(c, "GET", "/r/missing");
    // @output 404 [reserr,outer] text/plain reserr: no matching resource found for path : /r/missing , method : GET
    check show(c, "PUT", "/r/hello");
    // @output 405 [reserr,outer] text/plain reserr: Method not allowed
    check show(c, "POST", "/r/echo", body = "x");
    // @output 400 [reserr,outer] text/plain reserr: data binding failed: failed to parse JSON payload: invalid character 'x' looking for beginning of value

    // The response an error interceptor receives carries the status and body of the error.
    check show(c, "GET", "/r/hello", "reserr-next");
    // @output 500 [reserr,outer] text/plain resource failed
    check show(c, "GET", "/r/hello", "conflict");
    // @output 409 [reserr,outer] application/json {"code":"C1"}
    check show(c, "GET", "/r/hello", "reserr-nil");
    // @output 202 [outer]
    check show(c, "GET", "/r/hello", "reserr-fail");
    // @output 500 application/json {"status":500,"reason":"Internal Server Error","message":"reserr failed: resource failed","path":"/r/hello","method":"GET"}

    // A request error interceptor handles the errors of the request interceptors before it.
    check show(c, "GET", "/r/hello", "req-fail");
    // @output reqerr req failed false
    // @output 200 [last,mid,outer] text/plain hello
    check show(c, "GET", "/r/hello?n=x", "");
    // @output reqerr error in casting query param : 'n' true
    // @output 200 [last,mid,outer] text/plain hello
    check show(c, "GET", "/r/hello?n=x", "reqerr-respond");
    // @output reqerr error in casting query param : 'n' true
    // @output 200 [mid,outer] text/plain from reqerr

    // A response from a request interceptor goes to the response interceptors before it.
    check show(c, "GET", "/r/hello", "req-respond");
    // @output 200 [outer] text/plain from reqtag

    check show(c, "GET", "/d/a");
    // @output 200 [outer] text/plain a
    check show(c, "GET", "/d/a", "req-fail");
    // @output 500 application/json {"status":500,"reason":"Internal Server Error","message":"req failed","path":"/d/a","method":"GET"}
    check show(c, "GET", "/d/missing");
    // @output 404 application/json {"status":404,"reason":"Not Found","message":"no matching resource found for path : /d/missing , method : GET","path":"/d/missing","method":"GET"}

    // A response interceptor's error still reaches the error interceptors after ctx.next() passed
    // the first stage.
    check show(c, "GET", "/c/b", "exhaust-error");
    // @output 500 [callerreserr,outer] text/plain after exhaust
    // A response error interceptor can respond through its http:Caller after a Caller resource returns.
    check show(c, "GET", "/c/a", "caller-fail");
    // @output 200 [outer] text/plain handled by caller: caller boom
    // An error parameter of the service gets no error a request error interceptor handled.
    check show(c, "GET", "/c/err", "fail");
    // @output reqlog req failed
    // @output 200 [exhaust,outer] text/plain resource error: new error
    check awaitEvents(1);
    // @output ["reqlog req failed"]

    // Responding through the Caller of a request interceptor keeps its ctx.next().
    check show(c, "GET", "/c/a", "respond-next");
    // @output 200 [outer] text/plain early
    check awaitEvents(2);
    // @output ["next ok","service illegal respond: response has already been sent"]
    // An error returned after responding is rejected rather than handed to the request error
    // interceptors, which would record it.
    check show(c, "GET", "/c/a", "respond-error");
    // @output 200 [outer] text/plain early
    check show(c, "GET", "/c/a");
    // @output 200 [exhaust,outer] text/plain a
    check awaitEvents(1);
    // @output ["service sent"]
}
