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

type IntFn record {|
    function (int) returns xml f;
    int m;
|};

type StringFn record {|
    function (string) returns xml f;
    int m;
|};

function kind(any v) returns string {
    return v is IntFn ? "IntFn" : v is StringFn ? "StringFn" : "neither";
}

// The lambda parameter is the index expression or the method-call argument of the step, so the step operand is the
// same under both candidates and only the extension decides.
public function main() {
    xml x = xml `<root><a>1</a><a>2</a><a>3</a></root>`;
    IntFn|StringFn indexed = {f: i => x/<a>[i], m: 1};
    io:println(kind(indexed)); // @output IntFn
    var f = indexed.f;
    if f is function (int) returns xml {
        io:println(f(2)); // @output <a>3</a>
    }
    IntFn|StringFn called = {f: i => x/<a>.get(i), m: 1};
    io:println(kind(called)); // @output IntFn
    var g = called.f;
    if g is function (int) returns xml {
        io:println(g(1)); // @output <a>2</a>
    }
}
