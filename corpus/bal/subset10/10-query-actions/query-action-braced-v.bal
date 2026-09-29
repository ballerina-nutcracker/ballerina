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

client class NumberClient {
    remote function value() returns int|error {
        return 7;
    }

    remote function failing() returns int|error {
        return error("remote failed");
    }
}

function collectBraced(int[] xs) returns error? {
    int[] seen = [];
    error? result = (from var x in xs do {
        seen.push(x);
    });
    io:println(result is ()); // @output true
    io:println(seen); // @output [1,2,3]

    seen = [];
    () checked = (check from var x in xs do {
        seen.push(x * 2);
    });
    io:println(checked is ()); // @output true
    io:println(seen); // @output [2,4,6]

    seen = [];
    () panicked = (checkpanic from var x in xs do {
        seen.push(x * 3);
    });
    io:println(panicked is ()); // @output true
    io:println(seen); // @output [3,6,9]
}

function callBraced(NumberClient numbers) returns error? {
    int value = (check numbers->value());
    io:println(value); // @output 7
    int|error failed = (trap numbers->failing());
    io:println(failed is error); // @output true
    int unreachable = (check numbers->failing());
    io:println(unreachable);
}

public function main() {
    error? collected = collectBraced([1, 2, 3]);
    error? called = callBraced(new);
    io:println(collected is ()); // @output true
    io:println(called is error); // @output true
}
