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

function acceptInt(int value) returns string { _ = value; return "int"; }
function acceptString(string value) returns string { _ = value; return "string"; }

function classify(int|string value,
        string result = value is int ? acceptInt(value) : acceptString(value)) returns string {
    return result;
}

public function main() {
    io:println(classify(1)); // @output int
    io:println(classify("one")); // @output string

    function(int|string value,
        string result = value is int ? acceptInt(value) : acceptString(value)) returns string fn =
        function(int|string actual, string supplied = "unused") returns string { _ = actual; return supplied; };
    io:println(fn(2)); // @output int
    io:println(fn("two")); // @output string
}
