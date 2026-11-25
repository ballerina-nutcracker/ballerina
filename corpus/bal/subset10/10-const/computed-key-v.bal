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

const K = "a";

const M1 = {["a"]: 1};
const M2 = {[K]: 1, [K + "b"]: 2, [string `c${K}`]: 3};
const M3 = {["x"]: {["y"]: [1, 2]}};

public function main() {
    io:println(M1); // @output {"a":1}
    io:println(M2); // @output {"a":1,"ab":2,"ca":3}
    io:println(M3); // @output {"x":{"y":[1,2]}}
    // The type of a constant is the singleton type of its value.
    1 a = M1.a;
    io:println(a); // @output 1
    io:println(M3 is readonly); // @output true
}
