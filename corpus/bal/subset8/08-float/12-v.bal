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

// @productions float exact-equality equality multiplicative-expr if-else-stmt equality-expr floating-point-literal return-stmt unary-expr any function-call-expr local-var-decl-stmt

import ballerina/io;

public function main() {
    float nan0 = 0.0 / 0.0;
    float nan1 = -0.0 / 0.0;
    float pInf = 1.0 / 0.0;
    float nInf = -1.0 / 0.0;

    io:println(exactEq(42.0, 42.0)); // @output true
    io:println(exactEq(1.0, 2.0)); // @output false
    io:println(exactEq(0.0, 0.0)); // @output true
    io:println(exactEq(0.0, -0.0)); // @output false
    io:println(exactEq(nan0, nan1)); // @output true
    io:println(exactEq(nan0, 1.0)); // @output false
    io:println(exactEq(nan0, nInf)); // @output false
    io:println(exactEq(pInf, nInf)); // @output false
    io:println(exactEq(nInf, pInf)); // @output false
    io:println(exactEq(pInf, pInf)); // @output true
    io:println(exactEq(nInf, nInf)); // @output true

    io:println(100.0 === 10e1); // @output true
    io:println(0.0 === 0.0); // @output true
    io:println(0.0 === -0.0); // @output false
    io:println(0.0 !== -0.0); // @output true
    io:println(nan0 === nan1); // @output true
    io:println(nan0 !== nan1); // @output false
    io:println(pInf !== nInf); // @output true
}

function exactEq(float f1, float f2) returns any {
    boolean eq = f1 === f2;
    boolean neEq = f1 !== f2;
    if eq == neEq {
        return ();
    }
    return eq;
}
