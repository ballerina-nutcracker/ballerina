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

@test:Config {before: seed, after: report}
function testPutAndGet() {
    int index = put(42);
    if get(index) != 42 {
        test:assertFail("stored value did not round-trip");
    }
}

// Reads past the end of the array, so this panics in ordinary code: ERROR.
@test:Config {}
function testGetOutOfRange() {
    int _ = get(9999);
}

// Two assertions fail in one test; both messages are reported.
@test:Config {}
function testSizeExpectations() {
    error? first = trap test:assertFail("size should start at 0");
    error? second = trap test:assertFail("size should never shrink");
    io:println("both assertions trapped: ", first is error && second is error);
}

@test:Config {enable: false}
function testNotReadyYet() {
    io:println("this never runs");
}

function seed() {
    io:println("seeding the store");
}

function report() {
    io:println("store size is now ", size());
}
