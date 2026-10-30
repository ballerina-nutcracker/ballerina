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

final int|string finalVar = 2;

// A final module variable cannot be narrowed by a conditional expression, a
// logical operator, a negated type test or a while condition.
public function main(boolean flag) {
    int a = finalVar is int ? finalVar : 0; // @error a final module variable cannot be narrowed
    io:println(a);
    if finalVar is int && flag {
        int b = finalVar; // @error a final module variable cannot be narrowed
        io:println(b);
    }
    if finalVar !is int {
    } else {
        int c = finalVar; // @error a final module variable cannot be narrowed
        io:println(c);
    }
    while finalVar is int {
        int d = finalVar; // @error a final module variable cannot be narrowed
        io:println(d);
        break;
    }
}
