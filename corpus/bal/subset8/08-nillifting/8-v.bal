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
    map<int> m = {};
    m["a"] = 5;
    m["b"] = 6;
    m["c"] = 1;

    io:println(m["a"] + m["b"]); // @output 11

    io:println(m["a"] + m["b"] + m["c"]); // @output 12

    io:println(m["a"] - m["b"]); // @output -1

    io:println(m["a"] + m["b"] - m["c"]); // @output 10

    io:println(m["b"] / m["c"]); // @output 6

    int? v5 = m["b"] / 3;
    io:println(v5); // @output 2

    int? v6 = -m["a"];
    io:println(v6); // @output -5
    io:println(-m["c"]); // @output -1

    int d = 13;
    int? v7 = m["a"] + d;
    io:println(v7); // @output 18
    io:println(m["a"] + m["b"] + m["c"] + d); // @output 25

    int? e = ();
    int? v8 = m["a"] + e;
    io:println(v8); // @output 
    io:println(m["a"] + m["b"] + m["c"] + d + e); // @output 

    io:println(-m["a"]); //@output -5
}
