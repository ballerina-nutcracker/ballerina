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

function describe(int|string|boolean x) returns string {
    if x is int|string {
        if x == "a" {
            "a" s = x;
            return s;
        }
    }
    return "other";
}

function loopUntilA(int|string x0) returns string {
    int|string x = x0;
    string seen = "";
    while x is string {
        if x == "a" {
            "a" s = x;
            seen += s;
            x = 1;
        } else {
            x = "a";
        }
    }
    return seen;
}

public function main() {
    io:println(describe("a"), " ", describe(1), " ", describe(true)); // @output a other other
    io:println(loopUntilA("b"), " ", loopUntilA("a"), " ", loopUntilA(1), "|"); // @output a a |
}
