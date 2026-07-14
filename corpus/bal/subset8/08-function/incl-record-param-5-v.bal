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

type V1 record {
    int foo = 5;
};

type V2 record {
    int bar = 10;
};

function foo(int base, *V1 v1, *V2 v2) returns int {
    return base * (v1.foo + v2.bar);
}

public function main() {
    io:println(foo(1)); // @output 15
    io:println(foo(1, foo = 10)); // @output 20
    io:println(foo(1, bar = 5)); // @output 10
    io:println(foo(1, foo = 1, bar = 5)); // @output 6
    io:println(foo(1, {foo: 1}, {bar: 5})); // @output 6
}
