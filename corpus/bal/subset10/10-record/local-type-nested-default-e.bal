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

int counter = 3;

function g(record {| int y; |} r) returns int => r.y;

function arrayType() {
    record {| int y = "a"; |}[] arr = []; // @error default nested in a local array type is not an int
    io:println(arr);
}

function mapType() {
    map<record {| int y = counter; |}> m = {}; // @error default nested in a local map type reads a mutable module variable
    io:println(m);
}

function functionType() {
    function (record {| int y = "b"; |}) returns int fn = g; // @error default nested in a local function type is not an int
    io:println(fn({y: 1}));
}

public function main() {
    arrayType();
    mapType();
    functionType();
}
