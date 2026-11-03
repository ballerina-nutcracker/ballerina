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
    function (string) returns string[] f;
    int m;
|};

type B record {|
    function (xml) returns xml[] f;
    int m;
|};

// The duplicate generated name this test pins can only arise when the `function (string)` candidate's trial runs
// first: it lowers `s/<a>`, fails on the string operand and never reaches `s/<b>`, so the `xml` candidate's trial
// lowers `s/<b>` through a recreated lambda resolver. When the `xml` candidate runs first the test is inert for that
// purpose. Candidate order comes from the BDD paths of the union, not from source order.
A|B m = {f: s => [s/<a>, s/<b>], m: 1};

function show(any v) {
    io:println(v is B); // @output true
    io:println(v is A); // @output false
}

public function main() {
    xml x = xml `<root><a>1</a><b>2</b></root>`;
    A|B v = m;
    show(v);
    if v is B {
        var f = v.f;
        io:println(f(x)); // @output [<a>1</a>,<b>2</b>]
    }
}
