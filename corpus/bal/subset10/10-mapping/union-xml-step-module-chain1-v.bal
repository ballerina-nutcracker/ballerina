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

// Each module-level constructor's trials reach steps whose operands depend on module-level symbols declared later,
// which are resolved lazily from inside the trial.
A|B first = {v: second.v/<b>, m: 1};
A|B second = {v: y/<a>, m: "s"};
[xml, int]|[xml, string] list = [second.v/<b>[index], 1];
int index = second is B ? 0 : 1;
xml y = xml `<root><a><b>1</b></a><a><b>2</b></a></root>`;
[function (string) returns string[], int]|[function (xml) returns xml[], int] lambda = [s => [s/<a>, (s + y)/**/<b>, first.v/*], 1];

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
    if v is [function (xml) returns xml[], int] {
        return "[function (xml) returns xml[], int]";
    }
    return "other";
}

public function main() {
    io:println(kind(first)); // @output A
    io:println(first); // @output {"v":<b>1</b><b>2</b>,"m":1}
    io:println(kind(second)); // @output B
    io:println(second); // @output {"v":<a><b>1</b></a><a><b>2</b></a>,"m":"s"}
    io:println(kind(list)); // @output [xml, int]
    io:println(list); // @output [<b>1</b><b>2</b>,1]
    var l = lambda;
    io:println(kind(l)); // @output [function (xml) returns xml[], int]
    if l is [function (xml) returns xml[], int] {
        var f = l[0];
        io:println(f(y)); // @output [<a><b>1</b></a><a><b>2</b></a>,<b>1</b><b>2</b><b>1</b><b>2</b>,12]
    }
}
