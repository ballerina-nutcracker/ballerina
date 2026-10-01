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

// Each invocation of a function has its own workers and queues, even while an
// outer invocation of the same function is still exchanging messages.
function sum(int n) returns int {
    worker a returns int {
        int m = <- function;
        return m <= 0 ? 0 : m + sum(m - 1);
    }
    n -> a;
    return wait a;
}

isolated function doubledSum(int n) returns int {
    worker a returns int {
        int m = <- function;
        int r = m <= 0 ? 0 : m + doubledSum(m - 1);
        r -> function;
        return r;
    }
    n -> a;
    int received = <- a;
    int waited = wait a;
    return received + waited;
}

public function main() {
    io:println(sum(4)); // @output 10
    io:println(doubledSum(3)); // @output 22
}
