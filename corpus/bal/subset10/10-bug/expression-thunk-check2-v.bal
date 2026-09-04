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

function checkedInt(boolean shouldFail) returns int|error {
    if shouldFail {
        return error("checked int failed");
    }
    return 10;
}

function fallibleDefaulted(int value, int increment = 1) returns int|error {
    return value + increment;
}

function checkedValues(boolean shouldFail) returns (int|error)[]|error {
    return [fallibleDefaulted(check checkedInt(shouldFail))];
}

public function main() {
    io:println(checkedValues(false)); // @output [11]

    (int|error)[]|error failed = checkedValues(true);
    if failed is error {
        io:println(failed.message()); // @output checked int failed
    }
}
