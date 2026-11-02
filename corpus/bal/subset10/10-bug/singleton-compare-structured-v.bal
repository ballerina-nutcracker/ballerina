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

const A = [1];
const M = {a: 1};
const N = {a: 1};
const B = [1, 2];
const P = [A, M];
const Q = [[1], {a: 1}];

public function main() {
    true same = A == A;
    io:println(same); // @output true

    false differ = A != A;
    io:println(differ); // @output false

    true sameMap = M == N;
    io:println(sameMap); // @output true

    true nested = P == Q;
    io:println(nested); // @output true

    true less = A < B;
    io:println(less); // @output true

    false greaterOrEqual = A >= B;
    io:println(greaterOrEqual); // @output false
}
