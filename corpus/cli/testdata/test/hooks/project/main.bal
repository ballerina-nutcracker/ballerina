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

@test:Config {before: setUp, after: tearDown}
function testWithHooks() {
    test:assertFail("body failed so the captured hook output is shown");
}

@test:Config {before: explode}
function testBeforePanics() {
    io:println("the body must not run");
}

// The body passes, so only the failure `after` records proves it ran at all:
// a passing test's captured output is never printed.
@test:Config {after: failInAfter}
function testAfterRunsAfterAPassingBody() {
}

@test:Config {after: explode}
function testAfterPanics() {
}

function setUp() {
    io:println("setUp ran");
}

function tearDown() {
    io:println("tearDown ran");
}

function failInAfter() {
    test:assertFail("after ran");
}

function explode() {
    panic error("hook exploded");
}
