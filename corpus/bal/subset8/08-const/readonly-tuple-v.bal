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
// @productions module-const-decl tuple-type-descriptor list-constructor-expr type-test-expr
import ballerina/io;

const [int, string] A = [1, "two"];
const [int, int...] B = [1, 2, 3];

public function main() {
    any a = A;
    io:println(a is readonly); // @output true
    io:println(a is [1, "two"]); // @output true
    readonly & [int, string] r = A;
    io:println(r); // @output [1,"two"]
    any b = B;
    io:println(b is readonly); // @output true
    io:println(b is readonly & [int, int...]); // @output true
}
