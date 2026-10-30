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

const A = 5;
const B = A is int;
const C = A !is string;
const D = "s" is int;
const E = (A is int|string) && !(A is float);
const F = 1.5 is int|float;
const G = () is ();
const H = [1, 2] is int[];
const I = {a: 1} !is map<string>;
const J = [1, 2] is readonly;
const K = 2.5d is decimal;
const L = B is true;

public function main() {
    io:println(B); // @output true
    io:println(C); // @output true
    io:println(D); // @output false
    io:println(E); // @output true
    io:println(F); // @output true
    io:println(G); // @output true
    io:println(H); // @output true
    io:println(I); // @output true
    io:println(J); // @output true
    io:println(K); // @output true
    io:println(L); // @output true
}
