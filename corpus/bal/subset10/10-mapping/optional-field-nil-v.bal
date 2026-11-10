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

type R record {|
    int a?;
    int b;
|};

type N record {|
    int? a?;
|};

type Inner record {|
    string s;
|};

type Outer record {|
    Inner i?;
    readonly string r?;
|};

type Open record {
    int a?;
};

type Rest record {|
    int b;
|};

type Tagged record {|
    "t" tag;
    int a?;
|};

type Other record {|
    "o" tag;
    string a?;
|};

const R C = {a: (), b: 8};
const map<R> CM = {k: {a: (), b: 9}};

int calls = 0;

function next(int? v) returns int? {
    calls += 1;
    return v;
}

public function main() {
    int? none = ();
    int? some = 5;

    R r1 = {a: none, b: 1};
    io:println(r1); // @output {"b":1}
    io:println(r1.hasKey("a")); // @output false
    R r2 = {a: some, b: 2};
    io:println(r2); // @output {"a":5,"b":2}
    R r3 = {"a": none, b: 3};
    io:println(r3); // @output {"b":3}
    R r4 = {a: (), b: 4};
    io:println(r4); // @output {"b":4}

    int? a = ();
    int b = 5;
    R r5 = {a, b};
    io:println(r5); // @output {"b":5}

    N n = {a: none};
    io:println(n); // @output {"a":null}

    R r6 = {a: next(()), b: 6};
    io:println(r6, " ", calls); // @output {"b":6} 1

    Inner? inner = ();
    Outer o1 = {i: inner};
    io:println(o1); // @output {}
    string? rs = ();
    Outer o2 = {readonly r: rs, i: {s: "x"}};
    io:println(o2); // @output {"i":{"s":"x"}}

    Open op = {a: none};
    io:println(op); // @output {}

    Rest rest = {b: 7};
    R r7 = {a: none, ...rest};
    io:println(r7); // @output {"b":7}
    R r8 = {a: some, ...rest};
    io:println(r8); // @output {"a":5,"b":7}

    Tagged|Other t = {tag: "t", a: none};
    io:println(t); // @output {"tag":"t"}

    io:println(C, " ", C.hasKey("a")); // @output {"b":8} false
    anydata c = C;
    io:println(c is R); // @output true
    io:println(CM); // @output {"k":{"b":9}}
}
