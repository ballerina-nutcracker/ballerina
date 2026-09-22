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

record {| int value = 42; |} moduleValue = {};

type OuterFn function(record {| int a = 7; |}) returns int;
type InnerFn function(record {| int b = 8; |}) returns int;
type NestedFn function(record {| int c = 9; |}) returns int;

function parameterValue(record {| int value = 43; |} input = {}) returns int {
    return input.value;
}

function foo(record { int a = 5; int b = 10; } m) returns int {
    return m.a + m.b;
}

public function main() {
    io:println(moduleValue.value); // @output 42
    io:println(parameterValue()); // @output 43
    io:println(foo({})); // @output 15
    io:println(foo({a: 6})); // @output 16

    OuterFn outerFn = function (record {| int a = 7; |} x) returns int {
        InnerFn innerFn = function (record {| int b = 8; |} y) returns int {
            NestedFn nestedFn = function (record {| int c = 9; |} z) returns int {
                return x.a + y.b + z.c;
            };
            return nestedFn({});
        };
        return innerFn({});
    };
    io:println(outerFn({})); // @output 24
}
