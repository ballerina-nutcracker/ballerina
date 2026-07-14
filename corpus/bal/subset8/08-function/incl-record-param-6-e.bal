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

type Values record {
    int foo = 5;
    int bar = 10;
};

function foo(int base, *Values values) returns int {
    return base * (values.foo + values.bar);
}

public function main() {
    io:println(foo(1, {foo: 1, bar: 5}, bar = 20)); // @error
    io:println(foo(1, {foo: 1}, bar = 20)); // @error
}
