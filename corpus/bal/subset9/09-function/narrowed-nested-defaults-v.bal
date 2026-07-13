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

public function main() {
    (function(function(int value) returns int callback, int result) returns int)|int narrowedWithLambdaArg =
        function(function(int value) returns int callback, int result) returns int {
            _ = callback;
            return result;
        };
    if narrowedWithLambdaArg is function(function(int x = 10) returns int callback, int result = callback()) returns int {
        io:println(narrowedWithLambdaArg(function(int actual) returns int {
            return actual + 1;
        })); // @output 11
    }

    (function(record {| int value; |} input) returns int)|int narrowedWithInlineRecord =
        function(record {| int value; |} input) returns int {
            return input.value;
        };
    if narrowedWithInlineRecord is function(record {| int value = 12; |} input = {}) returns int {
        io:println(narrowedWithInlineRecord()); // @output 12
    }
}
