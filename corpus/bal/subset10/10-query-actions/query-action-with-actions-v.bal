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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

client class NumberClient {
    remote function values() returns int[] {
        return [1, 2, 3];
    }

    remote function offset() returns int {
        return 10;
    }

    remote function scale(int value, int factor) returns int {
        return value * factor;
    }

    remote function checkedValues() returns int[]|error {
        return [4, 5];
    }

    remote function checkedOffset() returns int|error {
        return 20;
    }

    remote function fallible(int value) returns int|error {
        if value == 2 {
            return error("value two failed");
        }
        return value;
    }
}

client class NumberResourceClient {
    resource function get values() returns int[] {
        return [1, 2, 3];
    }

    resource function get offset() returns int {
        return 5;
    }

    resource function get scale/[int value]/[int factor]() returns int {
        return value * factor;
    }
}

function runCheckedAction(NumberClient numbers, int[] results) returns error? {
    from var value in check numbers->checkedValues()
        let int offset = check numbers->checkedOffset()
        do {
            results.push(value + offset);
    };
}

function runFailingAction(NumberClient numbers, int[] results) returns error? {
    from var value in numbers->values()
        do {
            int result = check numbers->fallible(value);
            results.push(result);
        };
}

public function main() {
    NumberClient numbers = new;
    int[] results = [];

    from var value in numbers->values()
        let int offset = numbers->offset()
        do {
            int scaled = numbers->scale(value, 2);
            results.push(scaled + offset);
        };
    io:println(results); // @output [12,14,16]

    results = [];
    from var value in (numbers->values())
        let int offset = (numbers->offset())
        do {
            int scaled = (numbers->scale(value, 3));
            results.push(scaled + offset);
        };
    io:println(results); // @output [13,16,19]

    results = [];
    error? checkedResult = runCheckedAction(numbers, results);
    io:println(checkedResult is ()); // @output true
    io:println(results); // @output [24,25]

    NumberResourceClient resources = new;
    results = [];
    from var value in resources->/values
        let int offset = resources->/offset
        do {
            int scaled = resources->/scale/[value]/[4];
            results.push(scaled + offset);
        };
    io:println(results); // @output [9,13,17]

    int nestedTotal = 0;
    from var value in [1, 2]
        let error? nestedResult = from var nestedValue in [value, value + 1]
            do {
                nestedTotal += nestedValue;
            }
        where nestedResult is ()
        do {
            nestedTotal += value * 10;
        };
    io:println(nestedTotal); // @output 38

    results = [];
    error? failedResult = runFailingAction(numbers, results);
    io:println(failedResult is error); // @output true
    io:println(results); // @output [1]
}
