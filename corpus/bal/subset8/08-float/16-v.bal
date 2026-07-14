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

// @productions float string string-literal exact-equality equality multiplicative-expr if-else-stmt equality-expr floating-point-literal return-stmt unary-expr any function-call-expr local-var-decl-stmt int-literal

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

    io:println(exactEqAF("not-float", nInf)); // @output false
    io:println(exactEqAF(1, 1.0)); // @output false
    io:println(exactEqFA(1.0, 1)); // @output false
    io:println(exactEqFA(8.0, 8.0)); // @output true
}

function exactEq(float f1, float f2) returns any {
    string b1 = exactEqAF(f1, f2);
    string b2 = exactEqFA(f1, f2);
    string b3 = exactEqAA(f1, f2);
    if b1 != b2 {
        return "a1";
    }
    if b2 != b3 {
        return "a2";
    }
    return b1;
}

function exactEqAF(any f1, float f2) returns string {
    boolean eq = f1 === f2;
    boolean neEq = f1 !== f2;
    if eq == neEq {
        return "b";
    }
    else if eq {
        return "true";
    }
    return "false";
}

function exactEqFA(float f1, any f2) returns string {
    boolean eq = f1 === f2;
    boolean neEq = f1 !== f2;
    if eq == neEq {
        return "c";
    }
    else if eq {
        return "true";
    }
    return "false";
}

function exactEqAA(any f1, any f2) returns string {
    boolean eq = f1 === f2;
    boolean neEq = f1 !== f2;
    if eq == neEq {
        return "d";
    }
    else if eq {
        return "true";
    }
    return "false";
}
