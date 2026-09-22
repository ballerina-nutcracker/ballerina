// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;

function checkedInt(int value) returns int|error {
    return value;
}

function checkedValues() returns int[]|error {
    return [1, 2];
}

function withDefault(int value, int increment = 1) returns int {
    return value + increment;
}

function named(int value, int increment = 1) returns int {
    return value + increment;
}

class Counter {
    function add(int value, int increment = 1) returns int {
        return value + increment;
    }
}

public function main() returns error? {
    int defaulted = withDefault(check checkedInt(8));
    io:println(defaulted); // @output 9

    int namedResult = named(value = check checkedInt(6));
    io:println(namedResult); // @output 7

    Counter counter = new;
    int methodResult = counter.add(check checkedInt(5));
    io:println(methodResult); // @output 6

    int[] values = [0, ...check checkedValues()];
    io:println(values); // @output [0,1,2]

    int[] queried = from var value in check checkedValues()
        select value;
    io:println(queried); // @output [1,2]
}
