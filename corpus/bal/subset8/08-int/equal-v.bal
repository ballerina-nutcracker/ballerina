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

// @productions equality boolean if-else-stmt equality-expr return-stmt unary-expr function-call-expr assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    boolean b = intEq(17, 17);
    if b {
        io:println(17); // @output 17
    }
    else {
        io:println(0);
    }

    b = intNotEq(21, 21);
    if b {
        io:println(0);
    }
    else {
        io:println(21); // @output 21
    }

    b = intEq(-17, -17);
    if b {
        io:println(-17); // @output -17
    }
    else {
        io:println(0);
    }

    b = intNotEq(9223372036854775807, -9223372036854775807);
    if b {
        io:println(9223372036854775807); // @output 9223372036854775807
    }
    else {
        io:println(0);
    }
}

function intEq(int i, int j) returns boolean {
    return i == j;
}

function intNotEq(int i, int j) returns boolean {
    return i != j;
}
