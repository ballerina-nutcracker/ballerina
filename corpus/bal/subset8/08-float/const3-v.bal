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

// @productions float module-const-decl equality multiplicative-expr if-else-stmt equality-expr floating-point-literal relational-expr return-stmt unary-expr additive-expr any function-call-expr local-var-decl-stmt
import ballerina/io;

const A = 0.0 / 0.0;
const B = -0.0 / 0.0;
const C = 1.0 / 0.0;
const D = -1.0 / 0.0;
const E = -0.0;
const F = 0.0;
const G = 1.5 * 0f;
const H = 1.5 <= 2.5;
const I = 1.5 >= 2.5;
const J = -0.0 + 0f;

public function main() {
    io:println(eq(A, B)); // @output true
    io:println(eq(C, D)); // @output false
    io:println(eq(E, F)); // @output true
    io:println(eq(A, C)); // @output false
    io:println(eq(B, D)); // @output false
    io:println(G); // @output 0.0
    io:println(H); // @output true
    io:println(I); // @output false
    io:println(J); // @output 0.0
}

function eq(float f1, float f2) returns any {
    boolean eq = f1 == f2;
    boolean neEq = f1 != f2;
    if eq == neEq {
        return ();
    }
    return eq;
}
