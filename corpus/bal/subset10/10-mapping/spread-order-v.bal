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

type Tagged record {|
    int p;
|};

type Inner record {|
    int v;
|};

type Wrapper record {|
    Inner inner;
|};

function trace(string tag, int value) returns int {
    io:println(tag);
    return value;
}

function makeTagged() returns Tagged {
    io:println("operand");
    return {p: trace("p", 1)};
}

function optional(string tag, int value) returns int? {
    io:println(tag);
    return value;
}

public function main() {
    map<int> ordered = {q: trace("q", 2), ...makeTagged(), s: trace("s", 3)};
    // @output q
    // @output operand
    // @output p
    // @output s
    io:println(ordered); // @output {"q":2,"p":1,"s":3}

    Tagged src = {p: 7};
    map<int> destination = {...src};
    destination["p"] = 5;
    io:println(src); // @output {"p":7}
    io:println(destination); // @output {"p":5}

    // Fields whose evaluation needs setup statements keep their place in source order.
    map<int?> lifted = {...makeTagged(), r: optional("r", 4) + 1};
    // @output operand
    // @output p
    // @output r
    io:println(lifted); // @output {"p":1,"r":5}

    Inner shared = {v: 1};
    Wrapper wrapper = {inner: shared};
    var copied = {...wrapper};
    shared.v = 2;
    io:println(copied.inner.v); // @output 2
}
