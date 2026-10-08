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

function isOpenWithIntRest(any v) returns boolean {
    return v is record {| string b; int...; |};
}

function isOpenWithIntOrStringRest(any v) returns boolean {
    return v is record {| string b; int|string...; |};
}

function isClosed(any v) returns boolean {
    return v is record {| string b; |};
}

public function main() {
    string k = "a";
    var m = {[k]: 1, b: "s"};
    io:println(m.b, m[k]); // @output s1
    io:println(isOpenWithIntRest(m)); // @output true
    io:println(isClosed(m)); // @output false
    string j = "c";
    var u = {[k]: 1, b: "s", [j]: "t"};
    io:println(isOpenWithIntOrStringRest(u), isOpenWithIntRest(u)); // @output truefalse
    var n = {[k]: 1};
    string z = "z";
    n[z] = 42;
    io:println(n); // @output {"a":1,"z":42}
    var l = {[k]: [1, 2]};
    int[]? li = l[k];
    io:println(li); // @output [1,2]
}
