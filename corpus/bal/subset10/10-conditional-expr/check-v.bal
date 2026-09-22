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

function value(boolean good) returns int|error {
    return good ? 10 : error("bad");
}
function condition(boolean good) returns boolean|error {
    return good ? true : error("condition");
}
function checked(boolean flag) returns int|error {
    return check (flag ? value(true) : value(false));
}
function panicValue() returns int {
    panic error("panic");
}

public function main() {
    io:println(checked(true)); // @output 10
    io:println(checked(false) is error); // @output true
    int safe = true ? 1 : (checkpanic panicValue());
    io:println(safe); // @output 1
    int|error conditionPanic = trap ((checkpanic condition(false)) ? 1 : 2);
    int|error thenPanic = trap (true ? (checkpanic panicValue()) : 2);
    int|error elsePanic = trap (false ? 1 : (checkpanic panicValue()));
    io:println(conditionPanic is error); // @output true
    io:println(thenPanic is error); // @output true
    io:println(elsePanic is error); // @output true
}
