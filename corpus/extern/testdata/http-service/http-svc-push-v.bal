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

const PORT = 19234;

listener http:Listener ep = new (PORT);

isolated map<string> results = {};

isolated function saveResult(string name, error? result) {
    lock {
        results[name] = result is () ? "ok" : result.message();
    }
}

isolated function result(string name) returns string {
    foreach int _ in 0 ..< 100 {
        lock {
            string? value = results[name];
            if value is string {
                return value;
            }
        }
        runtime:sleep(0.05);
    }
    return "missing";
}

function h2Get(int port, string path, boolean enablePush) returns string|error = external;

function textResponse(string text) returns http:Response {
    http:Response response = new;
    response.setTextPayload(text);
    return response;
}

service / on ep {
    resource function get main(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/styles.css");
        promise.addHeader("x-promise", "one");
        check caller->promise(promise);
        check caller->respond("main");
        http:Response pushed = textResponse("pushed styles");
        pushed.setHeader("x-pushed", "yes");
        check caller->pushPromisedResponse(promise, pushed);
    }

    resource function get early(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/data?page=1");
        check caller->promise(promise);
        http:Response pushed = new;
        pushed.setJsonPayload({page: 1});
        check caller->pushPromisedResponse(promise, pushed);
        check caller->respond("early");
    }

    resource function get unfulfilled(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/never");
        check caller->promise(promise);
        check caller->respond("unfulfilled");
    }

    resource function get head(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/missing", "HEAD");
        check caller->promise(promise);
        http:Response pushed = textResponse("not sent for HEAD");
        pushed.statusCode = 404;
        check caller->pushPromisedResponse(promise, pushed);
        check caller->respond("head");
    }

    resource function get twice(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/twice");
        check caller->promise(promise);
        check caller->promise(promise);
        check caller->pushPromisedResponse(promise, textResponse("first"));
        check caller->pushPromisedResponse(promise, textResponse("second"));
        error? third = caller->pushPromisedResponse(promise, textResponse("third"));
        check caller->respond(third is error ? third.message() : "sent");
    }

    resource function get invalid(http:Caller caller) returns error? {
        error? relative = caller->promise(new http:PushPromise("relative"));
        error? post = caller->promise(new http:PushPromise("/post", "POST"));
        error? malformed = caller->promise(new http:PushPromise("/%zz"));
        http:PushPromise withLength = new ("/length");
        withLength.addHeader("Content-Length", "5");
        error? length = caller->promise(withLength);
        string messages = "";
        foreach error? e in [relative, post, malformed, length] {
            messages += (messages == "" ? "" : "; ") + (e is error ? e.message() : "sent");
        }
        check caller->respond(messages);
    }

    resource function get disabled(http:Caller caller) returns error? {
        error? promised = caller->promise(new http:PushPromise("/disabled"));
        check caller->respond(promised is error ? promised.message() : "sent");
    }

    resource function get headersOnly(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/headers");
        promise.addHeader("x-promise", "unsent");
        error? pushed = caller->pushPromisedResponse(promise, textResponse("headers"));
        check caller->respond(pushed is error ? pushed.message() : "sent");
    }

    resource function get unpromised(http:Caller caller) returns error? {
        check caller->pushPromisedResponse(new http:PushPromise("/unpromised"), textResponse("unpromised"));
    }

    resource function get late(http:Caller caller) returns error? {
        check caller->respond("late");
        http:PushPromise promise = new ("/late");
        error? promised = caller->promise(promise);
        saveResult("late promise", promised);
        error? pushed = caller->pushPromisedResponse(promise, textResponse("late"));
        saveResult("late push", pushed);
    }

    resource function get http1(http:Caller caller) returns error? {
        error? promised = caller->promise(new http:PushPromise("/http1"));
        check caller->respond(promised is error ? promised.message() : "sent");
    }
}

service class CallerGate {
    *http:RequestInterceptor;

    resource function 'default [string... path](http:RequestContext ctx, http:Caller caller) returns http:NextService|error? {
        var _ = caller;
        return ctx.next();
    }
}

service http:InterceptableService /gated on ep {
    public function createInterceptors() returns CallerGate => new;

    resource function get page(http:Caller caller) returns error? {
        http:PushPromise promise = new ("/gated/styles.css");
        check caller->promise(promise);
        check caller->respond("gated");
        check caller->pushPromisedResponse(promise, textResponse("gated styles"));
    }
}

service class PromisingInterceptor {
    *http:ResponseInterceptor;

    remote function interceptResponse(http:RequestContext ctx, http:Caller caller) returns http:NextService|error? {
        check caller->promise(new http:PushPromise("/promising/never"));
        return ctx.next();
    }
}

service http:InterceptableService /promising on ep {
    public function createInterceptors() returns PromisingInterceptor => new;

    resource function get page() returns string => "promising";
}

public function testMain() returns error? {
    io:println(check h2Get(PORT, "/main", true));
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/styles.css :scheme=http x-promise=one
    // @output response :status=200 content-type=text/plain body=main
    // @output pushed :status=200 content-type=text/plain x-pushed=yes body=pushed styles
    io:println(check h2Get(PORT, "/early", true));
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/data?page=1 :scheme=http
    // @output response :status=200 content-type=text/plain body=early
    // @output pushed :status=200 content-type=application/json body={"page":1}
    io:println(check h2Get(PORT, "/unfulfilled", true));
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/never :scheme=http
    // @output response :status=200 content-type=text/plain body=unfulfilled
    // @output pushed reset
    io:println(check h2Get(PORT, "/head", true));
    // @output promise :authority=127.0.0.1:19234 :method=HEAD :path=/missing :scheme=http
    // @output response :status=200 content-type=text/plain body=head
    // @output pushed :status=404 content-type=text/plain body=
    io:println(check h2Get(PORT, "/twice", true));
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/twice :scheme=http
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/twice :scheme=http
    // @output response :status=200 content-type=text/plain body=Promised stream is already rejected or stream is no longer valid
    // @output pushed :status=200 content-type=text/plain body=first
    // @output pushed :status=200 content-type=text/plain body=second
    io:println(check h2Get(PORT, "/invalid", true));
    // @output response :status=200 content-type=text/plain body=invalid push promise path: 'relative': expected an absolute path; invalid push promise method: 'POST': expected 'GET' or 'HEAD'; invalid push promise path: '/%zz': invalid URL escape "%zz"; promised request headers cannot include "Content-Length"
    io:println(check h2Get(PORT, "/disabled", false));
    // @output response :status=200 content-type=text/plain body=Server push not allowed to opposite endpoint
    io:println(check h2Get(PORT, "/headersOnly", true));
    // @output response :status=200 content-type=text/plain body=Promised stream is already rejected or stream is no longer valid
    io:println(check h2Get(PORT, "/gated/page", true));
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/gated/styles.css :scheme=http
    // @output response :status=200 content-type=text/plain body=gated
    // @output pushed :status=200 content-type=text/plain body=gated styles
    io:println(check h2Get(PORT, "/promising/page", true));
    // @output promise :authority=127.0.0.1:19234 :method=GET :path=/promising/never :scheme=http
    // @output response :status=200 content-type=text/plain body=promising
    // @output pushed reset
    io:println(check h2Get(PORT, "/unpromised", true));
    // @output response :status=500 content-type=text/plain body=invalid push promise
    io:println(check h2Get(PORT, "/late", true));
    // @output response :status=200 content-type=text/plain body=late
    io:println("late promise: ", result("late promise")); // @output late promise: ok
    io:println("late push: ", result("late push")); // @output late push: Promised stream is already rejected or stream is no longer valid

    http:Client http1 = check new (string `http://localhost:${PORT}`, {httpVersion: http:HTTP_1_1});
    string message = check http1->/http1;
    io:println("http/1.1: ", message); // @output http/1.1: Sending a PUSH_PROMISE is not supported for HTTP/1.x connections
}
