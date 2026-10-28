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
    int[] values = [1, 2, 3, 4];
    io:println(values.reduce(isolated function(int acc, int v) returns int => acc + v, 0)); // @output 10

    int[] empty = [];
    io:println(empty.reduce(isolated function(int acc, int v) returns int => acc + v, 42)); // @output 42

    string joined = values.reduce(isolated function(string acc, int v) returns string => acc + v.toString(), ">");
    io:println(joined); // @output >1234

    float avg = values.reduce(isolated function(float acc, int v) returns float => acc + <float>v, 0.0) / 4.0;
    io:println(avg); // @output 2.5

    [int, string] t = [1, "a"];
    string mixed = t.reduce(isolated function(string acc, int|string v) returns string => acc + v.toString(), "");
    io:println(mixed); // @output 1a
}
