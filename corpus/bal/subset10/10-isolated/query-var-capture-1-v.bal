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

type F isolated function () returns int;

function q1(int[] xs) returns int[] {
    return from int x in xs
        let F f = isolated function () returns int { return x; }
        select f();
}

function q2(int[] xs) returns int[] {
    return from int x in xs
        let int y = x + 1
        let F f = isolated function () returns int { return y; }
        select f();
}

public function main() {
    io:println(q1([1, 2])); // @output [1,2]
    io:println(q2([1, 2])); // @output [2,3]
}
