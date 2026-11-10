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

type U record {|
    int|string a?;
|};

type Fin record {|
    "x" a?;
|};

type Nev record {|
    never a?;
|};

type J record {|
    json a?;
|};

type A1 record {|
    int a?;
|};

type A2 record {|
    string a?;
|};

type Nest record {|
    R inner?;
|};

function take(R r) returns R {
    return r;
}

public function main() {
    int? none = ();

    readonly & R ro = {a: none, b: 1};
    io:println(ro, " ", ro.hasKey("a")); // @output {"b":1} false

    R[] arr = [{a: none, b: 1}, {a: 2, b: 3}];
    io:println(arr); // @output [{"b":1},{"a":2,"b":3}]
    map<R> m = {k: {a: none, b: 4}};
    io:println(m); // @output {"k":{"b":4}}
    io:println(take({a: none, b: 9})); // @output {"b":9}

    Nest n1 = {inner: {a: none, b: 5}};
    io:println(n1); // @output {"inner":{"b":5}}
    R? noInner = ();
    Nest n2 = {inner: noInner};
    io:println(n2); // @output {}

    int|string? us = ();
    U u1 = {a: us};
    io:println(u1); // @output {}
    int|string? us2 = "s";
    U u2 = {a: us2};
    io:println(u2); // @output {"a":"s"}

    "x"? fx = ();
    Fin f = {a: fx};
    io:println(f); // @output {}

    Nev nv = {a: ()};
    io:println(nv); // @output {}

    J j = {a: ()};
    io:println(j); // @output {"a":null}

    A1|A2 sel = {a: none};
    io:println(sel, " ", sel is A1); // @output {} true
}
