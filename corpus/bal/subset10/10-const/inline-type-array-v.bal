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

const int SIZE = 3;

const int[3] A = [1];
const int[3] B = [1, 2, 3];
const int[SIZE] C = [7];
const int[2][3] D = [[1]];

public function main() {
    io:println(A); // @output [1,0,0]
    io:println(A is readonly); // @output true
    io:println(B); // @output [1,2,3]
    io:println(C); // @output [7,0,0]
    io:println(D); // @output [[1,0,0],[0,0,0]]
    io:println(D is readonly); // @output true
}
