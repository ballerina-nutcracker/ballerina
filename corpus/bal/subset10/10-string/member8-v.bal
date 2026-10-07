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

function quote(string:Char c) returns string {
    return "'" + c + "'";
}

public function main() {
    string s = "ab😀";
    string:Char c = s[2];
    io:println(c.length()); // @output 1
    io:println(s[0] + s[1]); // @output ab
    io:println(quote(s[1])); // @output 'b'
    int i = 1;
    io:println(s[i + 1] == "😀"); // @output true
    io:println(isOutOfRange("")); // @output true
    io:println(isOutOfRange("é😀")); // @output true
}

function isOutOfRange(string s) returns boolean {
    string:Char|error r = trap s[s.length()];
    return r is error;
}
