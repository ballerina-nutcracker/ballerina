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

import testorg/demo.calc;
import testorg/demo.store;
import ballerina/io;
import ballerina/test;

public function main() {
    io:println("main: starting up");
    calc:describe();
}

// Uses both modules together.
@test:Config {}
function testCalcAndStoreTogether() {
    int index = store:put(calc:add(20, 22));
    if store:get(index) != 42 {
        test:assertFail("expected 42 through calc and store");
    }
}

// The before hook fails its own assertion, so the body never runs.
@test:Config {before: brokenSetUp}
function testWithABrokenSetUp() {
    io:println("this body should not run");
}

// Panics on a nil unwrap in ordinary code: ERROR.
@test:Config {}
function testMissingConfiguration() {
    string? name = lookupName("nobody");
    string _ = <string>name;
}

function brokenSetUp() {
    test:assertFail("set up could not prepare the fixture");
}

function lookupName(string key) returns string? {
    if key == "admin" {
        return "Administrator";
    }
    return ();
}
