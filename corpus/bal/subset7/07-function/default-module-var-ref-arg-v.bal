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

// The module-level variables are declared after the functions that reference
// them from parameter defaults, so their types have to be resolved on demand
// while the function signatures are being resolved.
function tag(string s, string p = prefix) returns string {
    return p + s;
}

function shift(int n, int amount = offset + 1) returns int {
    return n + amount;
}

function bare(int n = offset) returns int {
    return n;
}

string prefix = "id-";
int offset = 10;

public function main() {
    io:println(tag("a")); // @output id-a
    io:println(tag("a", "x-")); // @output x-a
    io:println(shift(1)); // @output 12
    io:println(shift(1, 0)); // @output 1
    io:println(bare()); // @output 10
}
