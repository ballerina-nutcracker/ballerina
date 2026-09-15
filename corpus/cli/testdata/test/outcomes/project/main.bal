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
function testAssertFail() {
    io:println("printed before failing");
    test:assertFail("boom");
}

@test:Config {}
function testTrappedAssertFail() {
    error? first = trap test:assertFail("first failure");
    error? second = trap test:assertFail("second failure");
    io:println(first is error && second is error);
}

@test:Config {}
function testPanics() {
    panic error("x");
}

@test:Config {}
function testReturnsError() returns error? {
    return error("returned error");
}

@test:Config {enable: false}
function testDisabled() {
    io:println("a disabled test never runs");
}

const boolean ENABLED = false;

// A fully constant attachment carries no literal `enable` expression, so the
// evaluated annotation value is what decides here.
@test:Config {enable: ENABLED}
function testDisabledByAConstant() {
}
