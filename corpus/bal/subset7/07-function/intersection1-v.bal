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
type F1 function(int) returns int;
type F2 function(byte) returns int;
type F3 function(byte) returns int|string;

type Fx F1&F2;
type Fy F1&F3;

public function main() {
    Fx fx = foo;
    int r1 = fx(1);
    io:println(r1); // @output 2
    fx = bar;
    int r2 = fx(1);
    io:println(r2); // @output 5

    Fy fy = foo;
    int|string r3 = fy(2);
    io:println(r3); // @output 3
}

function foo(int a) returns int {
    return a + 1;
}

function bar(int a) returns byte {
    int _ = a;
    return 5;
}

function baz(byte a) returns int|string {
    if (a > 0) {
        return a - 1;
    } else {
        return "zero";
    }
}
