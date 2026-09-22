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
// @productions module-const-decl array-type-descriptor list-constructor-expr type-test-expr
import ballerina/io;

const byte[] A = [1, 2];
const int[] B = [3, 4];
const C = [[1, 2], [3, 4]];

public function main() {
    any a = A;
    io:println(a is readonly); // @output true
    readonly & byte[] r = A;
    io:println(r); // @output [1,2]
    io:println(a is byte[2]); // @output true
    any b = B;
    io:println(b is readonly & int[]); // @output true
    io:println(b is [3, 4]); // @output true
    io:println(C is readonly); // @output true
    any[] nested = C;
    io:println(nested[0] is readonly); // @output true
}
