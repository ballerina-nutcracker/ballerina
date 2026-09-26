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

    remote function factor() returns int {
        return 2;
    }

    remote function scale(int value, int factor) returns int {
        return value * factor;
    }

    remote function pair(int value) returns [string, int] {
        string key = "three";
        if value == 1 {
            key = "one";
        } else if value == 2 {
            key = "two";
        }
        return [key, value * 10];
    }

    remote function checkedValues() returns int[]|error {
        return [4, 5];
    }

    remote function checkedFactor() returns int|error {
        return 3;
    }

    remote function fallible(int value) returns int|error {
        if value == 2 {
            return error("value two failed");
        }
        return value * 10;
    }
}

client class NumberResourceClient {
    resource function get values() returns int[] {
        return [1, 2, 3];
    }

    resource function get factor() returns int {
        return 4;
    }

    resource function get scale/[int value]/[int factor]() returns int {
        return value * factor;
    }
}

function checkedQuery(NumberClient numbers) returns int[]|error {
    return from var value in check numbers->checkedValues()
        let int factor = check numbers->checkedFactor()
        select check numbers->scale(value, factor);
}

function failingCheckedQuery(NumberClient numbers) returns int[]|error {
    return from var value in numbers->values()
        select check numbers->fallible(value);
}

public function main() returns error? {
    NumberClient numbers = new;

    int[] selected = from var value in numbers->values()
        let int factor = numbers->factor()
        select numbers->scale(value, factor);
    io:println(selected); // @output [2,4,6]

    selected = from var value in (numbers->values())
        let int factor = (numbers->factor())
        select (numbers->scale(value, factor));
    io:println(selected); // @output [2,4,6]

    map<int> mapped = map from var value in numbers->values()
        select numbers->pair(value);
    io:println(mapped["one"], mapped["two"], mapped["three"]); // @output 102030

    io:println(check checkedQuery(numbers)); // @output [12,15]

    NumberResourceClient resources = new;
    selected = from var value in resources->/values
        let int factor = resources->/factor
        select resources->/scale/[value]/[factor];
    io:println(selected); // @output [4,8,12]

    int nestedTotal = 0;
    error?[] actionResults = from var value in [1, 2]
        select from var nestedValue in [value, value + 1]
            do {
                nestedTotal += nestedValue;
            };
    io:println(actionResults[0] is () && actionResults[1] is ()); // @output true
    io:println(nestedTotal); // @output 8

    (int|error)[] fallibleValues = from var value in numbers->values()
        select numbers->fallible(value);
    io:println(fallibleValues[0], fallibleValues[1] is error, fallibleValues[2]); // @output 10true30

    int[]|error failedResult = failingCheckedQuery(numbers);
    io:println(failedResult is error); // @output true
}
