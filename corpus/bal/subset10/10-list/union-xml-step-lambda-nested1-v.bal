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

function show(any v) {
    io:println(v is [function (xml) returns function (xml) returns xml[], int]); // @output true
    io:println(v is [function (string) returns function (string) returns string[], int]); // @output false
}

public function main() {
    xml x = xml `<root><a>1</a><b>2</b></root>`;
    // The duplicate generated name this test pins can only arise when the `function (string)` candidate's trial runs
    // first: it lowers `(s + t)/<a>`, fails on the string operand and never reaches `t/<b>`, so the `xml` candidate's trial
    // lowers `t/<b>` through a recreated lambda resolver. When the `xml` candidate runs first the test is inert for that
    // purpose. Candidate order comes from the BDD paths of the union, not from source order.
    [function (string) returns function (string) returns string[], int]|[function (xml) returns function (xml) returns xml[], int] l = [s => t => [(s + t)/<a>, t/<b>], 1];
    show(l);
    if l is [function (xml) returns function (xml) returns xml[], int] {
        var g = l[0];
        var h = g(x);
        io:println(h(x)); // @output [<a>1</a><a>1</a>,<b>2</b>]
    }
}
