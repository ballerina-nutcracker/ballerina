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

function callAssertExactEqualsMismatch() {
    test:assertExactEquals(1, "1");
}

function callAssertEqualsStringMismatch() {
    test:assertEquals("hello", "world");
}

public function main() {
    int sum = 1 + 1;
    test:assertTrue(sum == 2);
    test:assertFalse(sum == 3);
    test:assertEquals(sum, 2);
    test:assertEquals("hello", "hello");
    test:assertNotEquals(1, 2);
    test:assertExactEquals(1, 1);
    test:assertNotExactEquals(1, "1");

    // assertEquals'/assertNotEquals'-style diff-formatted failure messages
    // intentionally include lines with trailing whitespace (padding), which
    // this corpus harness's @output annotations can't represent verbatim
    // (outputRe trims trailing space/tab when parsing) — assertFail's plain
    // message has no such formatting, so it's used here instead.
    error? failure = trap test:assertFail("custom failure");
    if failure is error {
        io:println("caught: ", failure.message());
    }

    // Exercises assertNotExactEquals'/getInequalityErrorMsg's diff-message
    // construction (getBallerinaType/getStringDiff) without printing the
    // message itself, for the same trailing-whitespace reason as above —
    // only whether it failed is checked, not its exact text.
    error? typeMismatch = trap callAssertExactEqualsMismatch();
    io:println("typeMismatchCaught: ", typeMismatch is error);

    // Exercises assertEquals' same-type (string vs string) mismatch branch,
    // which calls getStringDiff — same "check only that it failed" pattern.
    error? stringMismatch = trap callAssertEqualsStringMismatch();
    io:println("stringMismatchCaught: ", stringMismatch is error);

    io:println("all assertions completed");
    // @output caught: custom failure
    // @output typeMismatchCaught: true
    // @output stringMismatchCaught: true
    // @output all assertions completed
}
