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

import ballerina/io;

type A record {|
    xml v;
    int m;
|};

type B record {|
    xml v;
    string m;
|};

type Holder record {|
    A|B inner = {v: xml `<root><a>1</a><a>2</a></root>`/<a>, m: 1};
    [xml, int]|[xml, string] list = [xml `<root><a>1</a><a>2</a></root>`/<a>[1], "s"];
    [function (string) returns string[], int]|[function (xml) returns xml[], int] lambda = [s => [s/<a>, s/<b>], 1];
|};

function kind(any v) returns string {
    if v is A {
        return "A";
    }
    if v is B {
        return "B";
    }
    if v is [xml, int] {
        return "[xml, int]";
    }
    if v is [xml, string] {
        return "[xml, string]";
    }
    if v is [function (xml) returns xml[], int] {
        return "[function (xml) returns xml[], int]";
    }
    return "other";
}

// Record field defaults and parameter defaults are resolved by the package resolver, so their steps are owned by
// `$module` even though the parameter defaults sit on a function.
public function main() {
    Holder h = {};
    io:println(kind(h.inner)); // @output A
    io:println(h.inner); // @output {"v":<a>1</a><a>2</a>,"m":1}
    io:println(kind(h.list)); // @output [xml, string]
    io:println(h.list); // @output [<a>2</a>,"s"]
    io:println(kind(h.lambda)); // @output [function (xml) returns xml[], int]
    defaults();
}

function defaults(A|B m = {v: xml `<root><a>1</a></root>`/<a>.get(0), m: 1},
        [xml, int]|[xml, string] l = [xml `<root><a>1</a></root>`/*, 7],
        [function (string) returns string[], int]|[function (xml) returns xml[], int] lambda = [s => [s/<a>, s/<b>], 1]) {
    io:println(kind(m)); // @output A
    io:println(m); // @output {"v":<a>1</a>,"m":1}
    io:println(kind(l)); // @output [xml, int]
    io:println(l); // @output [<a>1</a>,7]
    io:println(kind(lambda)); // @output [function (xml) returns xml[], int]
    if lambda is [function (xml) returns xml[], int] {
        var f = lambda[0];
        io:println(f(xml `<root><a>1</a><b>2</b></root>`)); // @output [<a>1</a>,<b>2</b>]
    }
}
