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

import testorg/function_assign10_v.functions;

type F function (int, int) returns int;

type FooFn function (int) returns int;

public function main() {
    F x = functions:sum;
    io:println(x(1, 2)); // @output 3
    functions:BinaryFn y = functions:sum;
    io:println(y(2, 2)); // @output 4

    FooFn f = functions:foo;
    io:println(f(1)); // @output 2
    f = foo;
    io:println(f(1)); // @output 3
}

function foo(int i) returns int {
    return i + 2;
}
