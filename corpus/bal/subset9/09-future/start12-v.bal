// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

function printAndReturn(int value) returns int {
    io:println(value);
    return value;
}

function crash() returns int {
    panic error("started strand panic");
}

public function main() {
    _ = start printAndReturn(1);
    future<int> second = start printAndReturn(2);
    io:println(0); // @output 0
    int|error secondResult = wait second;
    io:println(secondResult); // @output 1
                                // @output 2
                                // @output 2

    future<int> completed = start printAndReturn(3);
    int|error completedResult = wait completed;
    io:println(completedResult); // @output 3
                                 // @output 3
    _ = start printAndReturn(4);
    completedResult = wait completed;
    io:println(completedResult); // @output 4
                                 // @output error("multiple waits on the same future is not allowed")

    future<int> failed = start crash();
    _ = start printAndReturn(5);
    int|error trapped = trap wait failed;
    io:println(trapped is error); // @output 5
                                  // @output true

    _ = start printAndReturn(6);
    io:println(7); // @output 7
}
