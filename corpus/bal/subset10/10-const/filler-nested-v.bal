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

type L2 int[2][2];
type L3 int[2][2][2];
type Row int[2];

const L2 A = [];
const L3 B = [];
const L2 C = [[7, 8]];
const [Row, Row] D = [[7]];

public function main() {
    io:println(A); // @output [[0,0],[0,0]]
    io:println(A is readonly); // @output true
    any[] rows = A;
    io:println(rows[0] is readonly); // @output true
    io:println(rows[1] is readonly); // @output true
    io:println(B); // @output [[[0,0],[0,0]],[[0,0],[0,0]]]
    io:println(B is readonly); // @output true
    io:println(C); // @output [[7,8],[0,0]]
    io:println(D); // @output [[7,0],[0,0]]
    any[] mixed = D;
    io:println(mixed[0] is readonly); // @output true
    io:println(mixed[1] is readonly); // @output true
}
