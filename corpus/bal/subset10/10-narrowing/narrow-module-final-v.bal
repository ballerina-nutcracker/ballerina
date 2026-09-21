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

const int|string CONSTANT = 1;
final int|string finalVar = 2;

// Constants and final module bindings cannot be reassigned, so they stay
// eligible for narrowing.
public function main() {
    if CONSTANT is int {
        io:println(CONSTANT + 1); // @output 2
    }
    if finalVar is int {
        io:println(finalVar + 1); // @output 3
    }
}
