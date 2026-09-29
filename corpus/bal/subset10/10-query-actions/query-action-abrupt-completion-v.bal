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

function valueOrError(int value) returns int|error {
    if value == 2 {
        return error("value two failed");
    }
    return value;
}

function returnFromQueryAction() returns string {
    from var value in [1, 2, 3]
    do {
        if value == 2 {
            return "returned from do";
        }
    };
    return "query completed";
}

function returnFromNestedQueryAction() returns string {
    from var outerValue in [1, 2]
    do {
        from var inner in [10, 20]
        do {
            if outerValue == 1 && inner == 20 {
                return "returned from nested do";
            }
        };
    };
    return "queries completed";
}

function successfulCheck() returns int|error {
    int total = 0;
    from var value in [1, 3]
    do {
        total += check valueOrError(value);
    };
    return total;
}

function failingCheck(string[] events) returns error? {
    from var value in [1, 2, 3]
    do {
        events.push("before");
        _ = check valueOrError(value);
        events.push("after");
    };
    events.push("query completed");
}

function failingNestedCheck(string[] events) returns error? {
    from var outerValue in [1, 2]
    do {
        _ = outerValue;
        events.push("outer before");
        from var innerValue in [1, 2, 3]
        do {
            events.push("inner before");
            _ = check valueOrError(innerValue);
            events.push("inner after");
        };
        events.push("outer after");
    };
    events.push("queries completed");
}

function successfulCheckpanic() returns int {
    int total = 0;
    from var value in [1, 3]
    do {
        total += checkpanic valueOrError(value);
    };
    return total;
}

public function main() {
    io:println(returnFromQueryAction()); // @output returned from do
    io:println(returnFromNestedQueryAction()); // @output returned from nested do
    io:println(successfulCheck()); // @output 4

    string[] events = [];
    error? result = failingCheck(events);
    io:println(result is error); // @output true
    io:println(events); // @output ["before","after","before"]

    events = [];
    result = failingNestedCheck(events);
    io:println(result is error); // @output true
    io:println(events); // @output ["outer before","inner before","inner after","inner before"]

    io:println(successfulCheckpanic()); // @output 4
}
