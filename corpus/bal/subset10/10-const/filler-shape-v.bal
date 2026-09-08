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

const L2 A = [];

public function main() {
    readonly & [[0, 0], [0, 0]] exact = A;
    io:println(exact); // @output [[0,0],[0,0]]
    any[] rows = A;
    io:println(rows[0] is readonly & [0, 0]); // @output true
    io:println(rows[0] is readonly & [1, 0]); // @output false
    io:println(A is readonly & [[0, 0], [0, 0]]); // @output true
}
