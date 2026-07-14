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

// @productions equality boolean if-else-stmt equality-expr boolean-literal return-stmt function-call-expr assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

function checkEquality(boolean b1, boolean b2) returns boolean {
    return b1 == b2;
}

function checkInEquality(boolean b1, boolean b2) returns boolean {
    return b1 != b2;
}

public function main() {
    boolean b = checkEquality(true, true);
    if b {
        io:println(4); // @output 4
    }
    else {
        io:println(5);
    }
    b = checkEquality(false, false);
    if b {
        io:println(6); // @output 6
    }
    else {
        io:println(7);
    }
    b = checkInEquality(true, true);
    if b {
        io:println(8);
    }
    else {
        io:println(9); // @output 9
    }
}
