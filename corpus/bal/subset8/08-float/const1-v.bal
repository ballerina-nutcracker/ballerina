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

// @productions module-const-decl multiplicative-expr floating-point-literal additive-expr
import ballerina/io;

const A = 1f + 2f;
const B = 3f - 1f;
const C = 10f / 2.5;
const D = 2f * 4f;
const E = 1.5f + 2f;
const F = 5f / 2f;
const G = 1.5 * 2.5;

public function main() {
    io:println(A); // @output 3.0
    io:println(B); // @output 2.0
    io:println(C); // @output 4.0
    io:println(D); // @output 8.0
    io:println(E); // @output 3.5
    io:println(F); // @output 2.5
    io:println(G); // @output 3.75

}
