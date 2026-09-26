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

type A [int];

type C [int|string];

function pick(C x, [boolean] b, boolean flag) returns string {
    var v = x is A ? x : (flag ? b : x);
    v = b;
    return v is [boolean] ? "boolean" : "other";
}

function pick2(C x, [boolean] b, boolean flag) returns string {
    var v = flag ? (x is A ? x : b) : (x is A ? b : x);
    v = b;
    return v is [boolean] ? "boolean" : "other";
}

public function main() {
    io:println(pick([1], [true], true), " ", pick2(["s"], [true], false)); // @output boolean boolean
}
