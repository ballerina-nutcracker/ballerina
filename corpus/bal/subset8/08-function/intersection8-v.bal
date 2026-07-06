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

// Venn diagram https://github.com/ballerina-platform/nballerina/issues/907#issuecomment-1588623742
import ballerina/io;

type F1 function (1|2|3|4) returns "a"|"b"|"c";

type F2 function (2|3|4|5) returns "b"|"c"|"d";

type F3 function (3|4|5|6) returns "c"|"d"|"e";

type F F1 & F2 & F3;

public function main() {
    F f = foo;
    "c"|"b" r1 = f(2);
    io:println(r1); // @output c
    3|4 val = 3;
    "c" r2 = f(val);
    io:println(r2); // @output c
    "c"|"d" r3 = f(5);
    io:println(r3); // @output c
}

function foo(1|2|3|4|5|6 a) returns "c" {
    1|2|3|4|5|6 _ = a;
    return "c";
}
