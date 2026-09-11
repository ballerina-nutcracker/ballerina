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

const int SIZE = 2;

const int[3]|string A = [1];
const map<int[2]> M = {a: [1]};
const [int, int[2]] T = [1, [2]];
const (int[2])[] L = [[1]];
const int[2] E = [];
const record {| int x = SIZE; int[SIZE] y = [4]; |} R = {};
const record {| int x = SIZE * 3; |} S = {};

public function main() {
    io:println(A); // @output [1,0,0]
    io:println(M); // @output {"a":[1,0]}
    io:println(M is readonly); // @output true
    io:println(T); // @output [1,[2,0]]
    io:println(L); // @output [[1,0]]
    io:println(E); // @output [0,0]
    io:println(R); // @output {"x":2,"y":[4,0]}
    io:println(S); // @output {"x":6}
}
