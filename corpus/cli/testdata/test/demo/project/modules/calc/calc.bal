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

// add returns the sum of a and b.
public function add(int a, int b) returns int {
    return a + b;
}

// divide panics on a zero divisor, the way ordinary Ballerina code does.
public function divide(int a, int b) returns int {
    return a / b;
}

// factorial is deliberately wrong for n > 2 so a test can catch it.
public function factorial(int n) returns int {
    if n <= 1 {
        return 1;
    }
    return n * 2;
}

public function describe() {
    io:println("calc module loaded");
}
