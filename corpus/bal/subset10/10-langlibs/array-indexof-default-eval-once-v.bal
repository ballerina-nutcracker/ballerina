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

// An argument expression is evaluated exactly once even when the compiler
// supplies the omitted `startIndex` from the declared default, and the default
// itself is evaluated per invocation only when the argument is omitted.
int counter = 0;

function next() returns int {
    counter += 1;
    io:println("eval");
    return 2;
}

public function main() {
    int[] arr = [1, 2, 3, 2];

    // @output eval
    io:println(arr.indexOf(next())); // @output 1
    io:println(counter); // @output 1

    // @output eval
    io:println(arr.indexOf(next(), 2)); // @output 3
    io:println(counter); // @output 2
}
