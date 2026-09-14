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

// Skipped: https://github.com/ballerina-nutcracker/ballerina/issues/915
//
// The receiver `letters` has no resolved type inside a default value
// expression, so the langlib method lookup fails with "method not found:
// length" / "method not found: indexOf". The same calls work with a parameter
// or call-result receiver; see langlib-method-default-param-receiver-v.bal.
final readonly & int[] letters = [10, 20, 30, 20];

function lengthOf(int n = letters.length()) returns int => n;

function firstOf(int? pos = letters.indexOf(30)) returns int? => pos;

public function main() {
    io:println(lengthOf()); // @output 4
    io:println(firstOf()); // @output 2
}
