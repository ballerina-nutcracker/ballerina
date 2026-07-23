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

int count = 0;

function argument() returns int {
    count += 1;
    io:println(count);
    return count;
}

function combine(int first, int second) returns int {
    io:println(1000);
    return first * 10 + second;
}

isolated function add(int first, int second) returns int {
    return first + second;
}

isolated function failure() returns error {
    return error("bad");
}

isolated function unionFailure() returns int|error {
    return error("union bad");
}

isolated function crash() {
    panic error("started");
}

public function main() {
    future<int> ordered = start combine(argument(), argument());
    io:println(99); // @output 99
    // wait is going to start the process
                    // @output 1
                    // @output 2
                    // @output 1000
    int|error orderedResult = wait ordered;
    io:println(orderedResult); // @output 12

    future<int> f = start add(20, 22);
    int|error first = wait f;
    int|error second = wait f;
    io:println(first); // @output 42
    io:println(second); // @output error("multiple waits on the same future is not allowed")

    future<error> errorFuture = start failure();
    error errorResult = wait errorFuture;
    io:println(errorResult.message()); // @output bad

    future<int|error> unionFuture = start unionFailure();
    int|error unionResult = wait unionFuture;
    io:println(unionResult is error); // @output true

    future<()> panicFuture = start crash();
    var trappedFirst = trap wait panicFuture;
    var trappedSecond = trap wait panicFuture;
    io:println(trappedFirst is error); // @output true
    io:println(trappedSecond); // @output error("multiple waits on the same future is not allowed")
}
