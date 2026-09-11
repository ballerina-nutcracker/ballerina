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
import ballerina/test;

@test:Config {}
function testAdd() {
    int sum = add(2, 3);
    io:println("add(2, 3) = ", sum);
    if sum != 5 {
        test:assertFail("add(2, 3) should be 5");
    }
}

// Fails an assertion: factorial(4) is 8, not 24.
@test:Config {}
function testFactorial() {
    int result = factorial(4);
    io:println("factorial(4) = ", result);
    if result != 24 {
        test:assertFail("factorial(4) should be 24 but was " + result.toString());
    }
}

// Panics in ordinary code — no assertion involved — so this is reported ERROR.
@test:Config {}
function testDivideByZero() {
    int _ = divide(10, 0);
}

@test:Config {before: announce}
function testDivide() {
    if divide(10, 2) != 5 {
        test:assertFail("divide(10, 2) should be 5");
    }
}

function announce() {
    io:println("about to divide");
}
