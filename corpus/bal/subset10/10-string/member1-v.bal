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
    string s = "abc";
    io:println(s[0]); // @output a
    io:println(s[1]); // @output b
    io:println(s[2]); // @output c
    string:Char c = s[1];
    io:println(c); // @output b

    var inferred = "xy";
    var ch = inferred[0];
    string:Char inferredChar = ch;
    io:println(inferredChar); // @output x

    string u = "aé€😀z";
    io:println(u[1]); // @output é
    io:println(u[2]); // @output €
    io:println(u[3]); // @output 😀
    io:println(u[4]); // @output z

    string|error r = trap s[3];
    io:println(r is error); // @output true
    r = trap s[-1];
    io:println(r is error); // @output true
    r = trap u[5];
    io:println(r is error); // @output true
    r = trap s[1];
    io:println(r); // @output b
}
