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

enum Color {
    RED,
    GREEN
}

type EnumHeaders record {|
    Color x\-c;
|};

type BadHeaders record {|
    map<string> m;
|};

type RateHeaders record {|
    string x\-id;
    int x\-num;
    string[] x\-multi;
    string? x\-opt;
|};

service /h on new http:Listener(19231) {
    resource function get basic(@http:Header string x\-id, @http:Header {name: "X-Num"} int num) returns string {
        return string `id=${x\-id} num=${num}`;
    }

    resource function get multi(@http:Header string[] x\-multi) returns string {
        return x\-multi.toString();
    }

    resource function get first(@http:Header string x\-multi) returns string {
        return x\-multi;
    }

    resource function get opt(@http:Header string? x\-opt) returns string {
        return x\-opt is () ? "nil" : string `[${x\-opt}]`;
    }

    resource function get rec(@http:Header RateHeaders h) returns string {
        return string `${h.x\-id} ${h.x\-num} ${h.x\-multi.toString()} ${h.x\-opt ?: "nil"}`;
    }

    resource function get recopt(@http:Header RateHeaders? h) returns string {
        return h is () ? "nil" : h.x\-id;
    }

    resource function get bool(@http:Header boolean x\-b) returns string {
        return x\-b.toString();
    }

    resource function get dec(@http:Header decimal[] x\-d) returns string {
        return x\-d.toString();
    }

    resource function get mixed(@http:Header string x\-id, string q, http:Request req) returns string|error {
        return string `${x\-id} ${q} ${check req.getHeader("x-id")}`;
    }

    resource function get badtype(@http:Header map<string> m) returns string {
        return m.toString();
    }

    resource function get erec(@http:Header EnumHeaders h) returns string {
        return h.x\-c;
    }

    resource function get badrec(@http:Header BadHeaders h) returns string {
        return h.m.toString();
    }

    resource function get all(http:Headers headers) returns string|error {
        string[] names = headers.getHeaderNames();
        boolean listed = names.indexOf("x-id") != () && names.indexOf("x-multi") != ();
        string|error missing = headers.getHeader("x-missing");
        return string `${check headers.getHeader("x-id")} ${(check headers.getHeaders("x-multi")).toString()} ` +
            string `${headers.hasHeader("X-ID")} ${headers.hasHeader("x-missing")} ${listed} ` +
            string `${missing is error ? "missing" : missing}`;
    }

    resource function get withreq(http:Headers headers, http:Request req) returns string|error {
        req.setHeader("x-id", "changed");
        return check headers.getHeader("x-id");
    }

    resource function post withbody(@http:Payload string body, http:Headers headers) returns string|error {
        return string `${body} ${check headers.getHeader("x-id")}`;
    }

    resource function get twoheaders(http:Headers a, http:Headers b) returns string {
        return a.hasHeader("x").toString() + b.hasHeader("x").toString();
    }

    resource function get twoann(@http:Header @http:Query string a) returns string {
        return a;
    }
}

function show(http:Client c, string path, map<string|string[]> headers = {}) returns string|error {
    http:Response r = check c->get(path, headers);
    if r.statusCode >= 400 {
        json body = check r.getJsonPayload();
        return string `${r.statusCode} ${(check body.message).toString()}`;
    }
    return string `${r.statusCode} ${check r.getTextPayload()}`;
}

public function testMain() returns error? {
    http:Client c = check new ("http://localhost:19231/h");
    http:Headers detached = new;
    string|error unbound = trap detached.getHeader("x-id");
    io:println(unbound is error ? unbound.message() : unbound); // @output Request is not initialized to retrieve headers
    io:println(show(c, "/all", {"x-id": "abc", "x-multi": ["a", "b"]})); // @output 200 abc ["a","b"] true false true missing
    io:println(show(c, "/withreq", {"x-id": "abc"})); // @output 200 changed
    http:Response r = check c->post("/withbody", "text", {"x-id": "abc"});
    io:println(r.statusCode, " ", check r.getTextPayload()); // @output 201 text abc
    io:println(show(c, "/twoheaders")); // @output 500 invalid multiple http:Headers parameter: b
    io:println(show(c, "/basic", {"x-id": "abc", "x-num": "7"})); // @output 200 id=abc num=7
    io:println(show(c, "/basic", {"x-num": "7"})); // @output 400 no header value found for 'x-id'
    io:println(show(c, "/basic", {"x-id": "abc", "x-num": "seven"})); // @output 400 header binding failed for parameter: 'X-Num'
    io:println(show(c, "/basic", {"x-id": "", "x-num": "7"})); // @output 400 no header value found for 'x-id'
    io:println(show(c, "/multi", {"x-multi": ["a", "b"]})); // @output 200 ["a","b"]
    io:println(show(c, "/multi", {"x-multi": "a, b"})); // @output 200 ["a, b"]
    io:println(show(c, "/first", {"x-multi": ["a", "b"]})); // @output 200 a
    io:println(show(c, "/opt")); // @output 200 nil
    io:println(show(c, "/opt", {"x-opt": ""})); // @output 200 nil
    io:println(show(c, "/opt", {"X-OPT": "v"})); // @output 200 [v]
    io:println(show(c, "/rec", {"x-id": "i", "x-num": "3", "x-multi": ["m1", "m2"]})); // @output 200 i 3 ["m1","m2"] nil
    io:println(show(c, "/rec", {"x-id": "i", "x-num": "3", "x-multi": "m", "x-opt": "o"})); // @output 200 i 3 ["m"] o
    io:println(show(c, "/rec", {"x-id": "i", "x-multi": "m"})); // @output 400 no header value found for 'x-num'
    io:println(show(c, "/rec", {"x-id": "i", "x-num": "n", "x-multi": "m"})); // @output 400 header binding failed for parameter: 'x-num'
    io:println(show(c, "/recopt", {"x-id": "i"})); // @output 200 nil
    io:println(show(c, "/bool", {"x-b": "nope"})); // @output 200 false
    io:println(show(c, "/dec", {"x-d": ["1.5", "2"]})); // @output 200 [1.5,2]
    io:println(show(c, "/dec", {"x-d": "abc"})); // @output 400 header binding failed for parameter: 'x-d'
    io:println(show(c, "/erec", {"x-c": "RED"})); // @output 200 RED
    io:println(show(c, "/erec", {"x-c": "BLUE"})); // @output 400 header binding failed for parameter: 'h'
    io:println(show(c, "/mixed?q=z", {"x-id": "abc"})); // @output 200 abc z abc
    io:println(show(c, "/badtype", {"m": "v"})); // @output 500 invalid type of header param m: expected one of the following types is expected: string,int,float,decimal,boolean, an array of the above types or a record which consists of the above types
    io:println(show(c, "/badrec", {"m": "v"})); // @output 500 invalid type of header param h: expected one of the following types is expected: string,int,float,decimal,boolean, an array of the above types or a record which consists of the above types
    io:println(show(c, "/twoann", {"a": "v"})); // @output 500 invalid multiple resource parameter annotations for a: expected one of the following types: http:Payload, http:CallerInfo, http:Header, http:Query
}
