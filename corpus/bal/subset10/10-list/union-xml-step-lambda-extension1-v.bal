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

type C record {|
    xml v;
    float m;
|};

function kind(any v) returns string {
    if v is A {
        return "A";
    }
    if v is B {
        return "B";
    }
    if v is C {
        return "C";
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
    if v is [function (xml, int) returns xml, int] {
        return "[function (xml, int) returns xml, int]";
    }
    return "other";
}

// Inferred lambdas as method-call extension arguments hold their own steps, inside and outside a two-step lambda.
public function main() {
    xml x = xml `<root><a><b>1</b></a><a><b>2</b></a></root>`;
    A|B|C m = {v: x/<a>, m: 1.5};
    io:println(kind(m)); // @output C
    io:println(m); // @output {"v":<a><b>1</b></a><a><b>2</b></a>,"m":1.5}
    [xml, int]|[xml, string]|[xml, float] l = [x/<a>[1], "s"];
    io:println(kind(l)); // @output [xml, string]
    io:println(l); // @output [<a><b>2</b></a>,"s"]
    [xml, int]|[xml, string] mapped = [x/<a>.map(e => e/<b>), 1];
    io:println(kind(mapped)); // @output [xml, int]
    io:println(mapped); // @output [<b>1</b><b>2</b>,1]
    [function (string) returns string[], int]|[function (xml) returns xml[], int] nested =
        [s => [s/<a>.map(e => e/<b>), s/<a>.filter(e => (e/<b>).length() > 0)], 1];
    io:println(kind(nested)); // @output [function (xml) returns xml[], int]
    if nested is [function (xml) returns xml[], int] {
        var f = nested[0];
        io:println(f(x)); // @output [<b>1</b><b>2</b>,<a><b>1</b></a><a><b>2</b></a>]
    }
    [function (string, int) returns string, int]|[function (xml, int) returns xml, int] pair = [(s, i) => s/<a>[i], 1];
    io:println(kind(pair)); // @output [function (xml, int) returns xml, int]
    if pair is [function (xml, int) returns xml, int] {
        var g = pair[0];
        io:println(g(x, 1)); // @output <a><b>2</b></a>
    }
}
