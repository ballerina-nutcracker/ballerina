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

type InvalidIntDetail record {|
    int value;
|};

type InvalidI32Detail record {|
    int:Signed32 value;
|};

type InvalidIntError error<InvalidIntDetail>;

type InvalidI32Error error<InvalidI32Detail>;

type DistinctIntError distinct error<InvalidIntDetail>;

type AnotherDistinctIntError distinct error<InvalidIntDetail>;

public function main() {
    InvalidI32Error e1 = createInvalidI32Error(5);
    io:println(e1 is InvalidIntError); // @output true

    InvalidIntError e2 = createInvalidIntError(5);
    io:println(e2 is DistinctIntError); // @output false

    DistinctIntError e3 = createDistinctInvalidIntError(5);
    io:println(e3 is InvalidIntError); // @output true
    io:println(e3 is AnotherDistinctIntError); // @output false
}

function createInvalidIntError(int value) returns InvalidIntError {
    return error("Invalid int", value = value);
}

function createDistinctInvalidIntError(int value) returns DistinctIntError {
    return error("Invalid int", value = value);
}

function createInvalidI32Error(int:Signed32 value) returns InvalidI32Error {
    return error("Invalid i32", value = value);
}
