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

type T [int, int, int];

type T1 [string, int, int];

public function main() {
    T[] a = [];
    io:println(a); // @output []
    a[1] = [1, 2, 3];
    io:println(a); // @output [[0,0,0],[1,2,3]]
    T[3] b = [];
    io:println(b); // @output [[0,0,0],[0,0,0],[0,0,0]]

    T1[] x = [];
    io:println(x); // @output []
    x[1] = ["a", 2, 3];
    io:println(x); // @output [["",0,0],["a",2,3]]
    T1[3] y = [];
    io:println(y); // @output [["",0,0],["",0,0],["",0,0]]
}
