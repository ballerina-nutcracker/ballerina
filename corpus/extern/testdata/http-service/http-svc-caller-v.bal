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

isolated map<string> results = {};
isolated boolean released = false;

isolated function saveResult(string name, http:ListenerError? result) {
    lock {
        results[name] = result is () ? "sent" : result.message();
    }
}

isolated function isReleased() returns boolean {
    lock {
        return released;
    }
}

listener http:Listener ep = new (19237);

// Isolated so its resources run concurrently: /release arrives while respondThenWait is still running.
isolated service /w on ep {
    // Holds the resource open until the client has read the response.
    resource function get respondThenWait(http:Caller caller) returns error? {
        check caller->respond("early");
        int waited = 0;
        while !isReleased() && waited < 40 {
            runtime:sleep(0.05);
            waited += 1;
        }
        string outcome = isReleased() ? "released" : "timed out";
        lock {
            results["wait"] = outcome;
        }
    }

    resource function get release() returns string {
        lock {
            released = true;
        }
        return "released";
    }
}

service /c on ep {
    resource function get twice(http:Caller caller) returns error? {
        check caller->respond("first");
        http:ListenerError? second = caller->respond("second");
        saveResult("twice", second is http:GenericListenerError ? second : ());
    }

    resource function get respondThenPanic(http:Caller caller) returns error? {
        check caller->respond("ok");
        panic error("boom");
    }

    resource function get panicFirst(http:Caller caller) {
        var _ = caller;
        panic error("early");
    }

    resource function get panicReturn() returns string {
        panic error("boom");
    }

    resource function get earlyHints(http:Caller caller) {
        http:Response response = new;
        response.statusCode = 103;
        http:ListenerError? result = caller->respond(response);
        saveResult("earlyHints", result);
    }

    resource function get respondThenRespondError(http:Caller caller) returns error? {
        check caller->respond("a");
        http:ListenerError? result = caller->respond(error("b"));
        saveResult("secondError", result);
    }

    resource function get redirectNowhere(http:Caller caller) returns error? {
        check caller->redirect(new http:Response(), http:REDIRECT_FOUND_302, []);
    }

    resource function get none(http:Caller caller) {
        var _ = caller;
    }

    resource function get err(http:Caller caller) returns error? {
        var _ = caller;
        return error("boom");
    }

    resource function get respondThenErr(http:Caller caller) returns error? {
        check caller->respond("done");
        return error("after");
    }

    resource function post created(http:Caller caller) returns error? {
        check caller->respond({a: 1});
    }

    resource function post response(http:Caller caller) returns error? {
        http:Response response = new;
        response.setTextPayload("as is");
        check caller->respond(response);
    }

    resource function 'default anyMethod(http:Caller caller) returns error? {
        check caller->respond("any");
    }

    resource function get status(http:Caller caller) returns error? {
        http:NotFound notFound = {body: "nf"};
        check caller->respond(notFound);
    }

    resource function get respondError(http:Caller caller) returns error? {
        check caller->respond(error("plain"));
    }

    resource function get respondAppError(http:Caller caller) returns error? {
        map<string[]> & readonly headers = {};
        check caller->respond(error http:ClientRequestError("cre", statusCode = 409, headers = headers, body = "x"));
    }

    resource function get respondConnectorError(http:Caller caller) returns error? {
        check caller->respond(error http:ClientConnectorError("upstream", error("refused")));
    }

    resource function get redirect(http:Caller caller) returns error? {
        check caller->redirect(new http:Response(), http:REDIRECT_FOUND_302, ["/a", "/b"]);
    }

    resource function get info(http:Caller caller) returns error? {
        io:println(caller.remoteAddress.host, " ", caller.remoteAddress.ip, " ", caller.remoteAddress.port > 0, " ",
            caller.localAddress.host, " ", caller.localAddress.port, " ", caller.protocol, " ", caller.getRemoteHostName());
        check caller->respond("info");
    }

    resource function get cont(http:Caller caller) returns error? {
        check caller->'continue();
        check caller->respond("after continue");
    }

    resource function get later(http:Caller caller) {
        _ = start respondLater(caller);
    }

    resource function get result/[string name]() returns string {
        lock {
            return results[name] ?: "pending";
        }
    }

    resource function get withRequest(http:Request request, string name, http:Caller caller) returns error? {
        check caller->respond(string `${request.method} ${name}`);
    }
}

// Isolated because a non-isolated strand started from a resource never runs (#1758).
isolated function respondLater(http:Caller caller) {
    runtime:sleep(0.2);
    http:ListenerError? result = caller->respond("late");
    saveResult("later", result);
}

function show(http:Client c, string path, string method = "GET") returns error? {
    http:Response response = check c->execute(method, path, "");
    string|error contentType = response.getHeader("content-type");
    string|error location = response.getHeader("location");
    io:println(response.statusCode, " ", contentType is string ? contentType : "-", location is string ? " [" + location + "]" : "");
    string|http:ClientError text = response.getTextPayload();
    if text is http:NoContentError {
        io:println("<empty>");
        return;
    }
    string body = check text;
    if contentType == "application/json" && body.indexOf("{\"timestamp\"") == 0 {
        json payload = check response.getJsonPayload();
        io:println(check payload.status, " | ", check payload.message, " | ", check payload.path);
        return;
    }
    io:println(body);
}

// Waits for a resource that responded first to record its result, then shows it.
function showResult(http:Client c, string name) returns error? {
    foreach int _ in 0 ..< 40 {
        string result = check c->get("/result/" + name);
        if result != "pending" {
            break;
        }
        runtime:sleep(0.05);
    }
    check show(c, "/result/" + name);
}

public function testMain() returns error? {
    http:Client c = check new ("http://127.0.0.1:19237/c", {followRedirects: {enabled: false}});

    check show(c, "/twice");
    // @output 200 text/plain
    // @output first
    check showResult(c, "twice");
    // @output 200 text/plain
    // @output illegal respond: response has already been sent
    check show(c, "/respondThenPanic");
    // @output 200 text/plain
    // @output ok
    check show(c, "/panicFirst");
    // @output 500 text/plain
    // @output early
    check show(c, "/panicReturn");
    // @output 500 text/plain
    // @output boom
    http:Client w = check new ("http://127.0.0.1:19237/w");
    // A separate client, so /release does not queue behind respondThenWait on the same connection.
    http:Client releaser = check new ("http://127.0.0.1:19237/w");
    check show(w, "/respondThenWait");
    // @output 200 text/plain
    // @output early
    check show(releaser, "/release");
    // @output 200 text/plain
    // @output released
    runtime:sleep(0.3);
    check show(c, "/result/wait");
    // @output 200 text/plain
    // @output released
    check show(c, "/earlyHints");
    // @output 500 application/json
    // @output 500 | invalid status code for a response: 103 | /c/earlyHints
    check showResult(c, "earlyHints");
    // @output 200 text/plain
    // @output invalid status code for a response: 103
    check show(c, "/respondThenRespondError");
    // @output 200 text/plain
    // @output a
    check showResult(c, "secondError");
    // @output 200 text/plain
    // @output illegal respond: response has already been sent
    // jBallerina panics on an empty locations array (ballerina-library#9853).
    check show(c, "/redirectNowhere");
    // @output 302 - []
    // @output <empty>
    check show(c, "/none");
    // @output 500 -
    // @output <empty>
    check show(c, "/err");
    // @output 500 application/json
    // @output 500 | boom | /c/err
    check show(c, "/respondThenErr");
    // @output 200 text/plain
    // @output done
    check show(c, "/created", "POST");
    // @output 201 application/json
    // @output {"a":1}
    check show(c, "/response", "POST");
    // @output 200 text/plain
    // @output as is
    check show(c, "/anyMethod", "POST");
    // @output 200 text/plain
    // @output any
    check show(c, "/anyMethod", "PUT");
    // @output 200 text/plain
    // @output any
    check show(c, "/status");
    // @output 404 text/plain
    // @output nf
    check show(c, "/respondError");
    // @output 500 application/json
    // @output 500 | plain | /c/respondError
    check show(c, "/respondAppError");
    // @output 409 text/plain
    // @output x
    check show(c, "/respondConnectorError");
    // @output 502 application/json
    // @output 502 | upstream: refused | /c/respondConnectorError
    check show(c, "/redirect");
    // @output 302 - [/a,/b]
    // @output <empty>
    check show(c, "/info");
    // @output 127.0.0.1 127.0.0.1 true 127.0.0.1 19237 http localhost
    // @output 200 text/plain
    // @output info
    // jBallerina never returns from 'continue() without an Expect header (ballerina-library#9852).
    check show(c, "/cont");
    // @output 200 text/plain
    // @output after continue
    check show(c, "/later");
    // @output 500 -
    // @output <empty>
    runtime:sleep(0.5);
    check show(c, "/result/later");
    // @output 200 text/plain
    // @output illegal respond: response has already been sent
    check show(c, "/withRequest?name=n");
    // @output 200 text/plain
    // @output GET n
}
