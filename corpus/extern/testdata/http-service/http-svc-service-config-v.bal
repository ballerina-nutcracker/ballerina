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
import ballerina/mime;

type Inner record {|
    int x;
|};

type Closed record {|
    string name;
    int? age;
    string nick?;
    Inner inner?;
    Inner[] items?;
|};

type ById record {|
    map<Inner> byId;
|};

type RestOnly record {|
    Inner...;
|};

type Headers record {|
    string? a;
|};

listener http:Listener ep = new (19238);

// Matches any port and any letter case of the request's host name (jBallerina compares the whole Host header, ballerina-library#9867).
@http:ServiceConfig {host: "LocalHost"}
service /h on ep {
    resource function get .() returns string => "named host";
}

@http:ServiceConfig {host: "localhost:1"}
service /p on ep {
    resource function get .() returns string => "other port";
}

service /h on ep {
    resource function get .() returns string => "default host";
}

listener http:Listener hostEp = new (19246);

@http:ServiceConfig {host: "localhost"}
service /x on hostEp {
    resource function get .() returns string => "host name";
}

// A service for the exact host and port answers before one for the host name alone.
@http:ServiceConfig {host: "LOCALHOST:19246"}
service /x on hostEp {
    resource function get .() returns string => "host and port";
}

@http:ServiceConfig {treatNilableAsOptional: false, mediaTypeSubtypePrefix: "vnd.test"}
service /t on ep {
    resource function get q(string? a) returns string => a is () ? "nil" : "[" + a + "]";

    resource function get h(@http:Header string? x) returns string => x ?: "nil";

    resource function get hr(@http:Header Headers? hs) returns string => hs is () ? "nil" : hs.a ?: "nil field";

    resource function get hrRequired(@http:Header Headers hs) returns string => hs.a ?: "nil field";

    resource function get j() returns json => {a: 1};

    resource function get e() returns error => error("bad");

    resource function get nf() returns http:NotFound => {body: "nf"};

    resource function get n() returns http:Accepted => {};

    resource function get c(http:Caller caller) returns error? {
        check caller->respond("caller");
    }

    resource function get prefixed() returns http:Ok => {body: "p", mediaType: "text/vnd.test+plain"};

    resource function get charset() returns http:Ok => {body: {a: 1}, mediaType: "application/json; charset=utf-8"};

    resource function get multipart() returns http:Response|error {
        mime:Entity part = new;
        part.setText("part");
        http:Response response = new;
        response.setBodyParts([part]);
        return response;
    }
}

@http:ServiceConfig {laxDataBinding: true}
service /lax on ep {
    resource function post r(Closed c) returns string => describe(c);

    resource function post o(Closed? c) returns string => c is () ? "nil" : describe(c);

    resource function post a(Closed[] c) returns int => c.length();

    resource function post m(ById body) returns int => body.byId.length();

    resource function post mp(map<Inner> body) returns int => body.length();

    resource function post rest(RestOnly body) returns int => body.length();
}

service /strict on ep {
    resource function post r(Closed c) returns string => describe(c);
}

listener http:Listener namedOnly = new (19239);

@http:ServiceConfig {host: "localhost"}
service /n on namedOnly {
    resource function get .() returns string => "any port";
}

// A service for the exact host and port takes the requests for it over one for the host name.
@http:ServiceConfig {host: "localhost:19239"}
service /n on namedOnly {
    resource function get .() returns string => "exact port";
}

function describe(Closed c) returns string {
    int? age = c.age;
    Inner? inner = c.inner;
    Inner[]? items = c.items;
    return string `name=${c.name} age=${age is () ? "nil" : age.toString()} nick=${c.hasKey("nick")} `
        + string `inner=${inner is () ? "-" : inner.x.toString()} items=${items is () ? "-" : items.length().toString()}`;
}

function show(http:Client c, string path, string method = "GET", json body = (), map<string> headers = {},
        boolean statusOnly = false) returns error? {
    http:Response response = check c->execute(method, path, body, headers);
    string|error contentType = response.getHeader("content-type");
    string mediaType = contentType is string ? contentType : "-";
    int? parameters = mediaType.indexOf(";");
    io:println(response.statusCode, " ", parameters is int ? mediaType.substring(0, parameters) : mediaType);
    if statusOnly {
        return;
    }
    string|http:ClientError text = response.getTextPayload();
    if text is http:NoContentError {
        io:println("<empty>");
        return;
    }
    string payload = check text;
    if payload.indexOf("{\"timestamp\"") == 0 {
        json errorPayload = check response.getJsonPayload();
        io:println(check errorPayload.message);
        return;
    }
    io:println(payload);
}

public function testMain() returns error? {
    http:Client byName = check new ("http://localhost:19238");
    http:Client byIp = check new ("http://127.0.0.1:19238");

    check show(byName, "/h");
    // @output 200 text/plain
    // @output named host
    check show(byIp, "/h");
    // @output 200 text/plain
    // @output default host
    check show(byName, "/p");
    // @output 404 application/json
    // @output no matching service found for path: /p
    check show(byIp, "/p");
    // @output 404 application/json
    // @output no matching service found for path: /p
    check show(check new ("http://localhost:19246"), "/x");
    // @output 200 text/plain
    // @output host and port

    check show(byIp, "/t/q");
    // @output 400 application/vnd.test+json
    // @output no query param value found for 'a'
    check show(byIp, "/t/q?a");
    // @output 200 text/vnd.test+plain
    // @output nil
    check show(byIp, "/t/q?a=");
    // @output 200 text/vnd.test+plain
    // @output []
    check show(byIp, "/t/h");
    // @output 400 application/vnd.test+json
    // @output no header value found for 'x'
    check show(byIp, "/t/h", headers = {x: ""});
    // @output 200 text/vnd.test+plain
    // @output nil
    check show(byIp, "/t/hr");
    // @output 200 text/vnd.test+plain
    // @output nil
    check show(byIp, "/t/hr", headers = {a: ""});
    // @output 200 text/vnd.test+plain
    // @output nil field
    check show(byIp, "/t/hrRequired");
    // @output 400 application/vnd.test+json
    // @output no header value found for 'a'
    check show(byIp, "/t/j");
    // @output 200 application/vnd.test+json
    // @output {"a":1}
    check show(byIp, "/t/e");
    // @output 500 application/vnd.test+json
    // @output bad
    check show(byIp, "/t/nf");
    // @output 404 text/vnd.test+plain
    // @output nf
    check show(byIp, "/t/n");
    // @output 202 -
    // @output <empty>
    check show(byIp, "/t/c");
    // @output 200 text/vnd.test+plain
    // @output caller
    check show(byIp, "/t/prefixed");
    // @output 200 text/vnd.test+plain
    // @output p
    http:Response charset = check byIp->get("/t/charset");
    io:println(check charset.getHeader("content-type"));
    // @output application/vnd.test+json; charset=utf-8
    // jBallerina prefixes these again (ballerina-library#9868).
    check show(byIp, "/t/multipart", statusOnly = true);
    // @output 200 multipart/form-data
    check show(byIp, "/t/missing");
    // @output 404 application/json
    // @output no matching resource found for path : /t/missing , method : GET

    json full = {name: "n", age: (), nick: (), extra: 1, inner: {x: 1, y: 2}, items: [{x: 2, y: 3}]};
    check show(byIp, "/lax/r", "POST", full);
    // @output 201 text/plain
    // @output name=n age=nil nick=false inner=1 items=1
    check show(byIp, "/lax/r", "POST", {name: "n"});
    // @output 201 text/plain
    // @output name=n age=nil nick=false inner=- items=-
    check show(byIp, "/lax/m", "POST", {byId: {k: {x: 1, y: 2}}});
    // @output 201 application/json
    // @output 1
    // Lax binding applies to a record parameter only, as in jBallerina, not to a nilable record, a map or an array.
    check show(byIp, "/lax/o", "POST", full, statusOnly = true);
    // @output 400 application/json
    check show(byIp, "/lax/mp", "POST", {k: {x: 1, y: 2}}, statusOnly = true);
    // @output 400 application/json
    // A rest-only record is the same type as map<Inner>; jBallerina projects it and answers 201.
    check show(byIp, "/lax/rest", "POST", {k: {x: 1, y: 2}}, statusOnly = true);
    // @output 400 application/json
    check show(byIp, "/lax/a", "POST", [full], statusOnly = true);
    // @output 400 application/json
    check show(byIp, "/strict/r", "POST", {name: "n"}, statusOnly = true);
    // @output 400 application/json

    http:Client namedOnlyByIp = check new ("http://127.0.0.1:19239");
    check show(namedOnlyByIp, "/n");
    // @output 404 application/json
    // @output no service has registered for listener : /127.0.0.1:19239
    check show(check new ("http://localhost:19239"), "/n");
    // @output 200 text/plain
    // @output exact port
}
