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

// A 3-byte multi-byte character straddling getFormattedString's 80-character
// chunking boundary, past maxArgLength so the value actually gets rewrapped.
final string longNonAsciiValue = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX" +
        "字XXXXXXXXXX";

function callAssertLongNonAscii() returns error? {
    test:assertEquals(1, longNonAsciiValue);
}

public function main() {
    error? failure = trap callAssertLongNonAscii();
    if failure is error {
        // Printing the raw message isn't reliable here (multi-line,
        // whitespace-padded diff-style content — see testerina-assert-v.bal's
        // own comment on why it avoids this). A byte-based chunk boundary
        // instead of a rune-based one at this exact character layout doesn't
        // corrupt visibly, but does change the message's rune count by 2 (a
        // subtle but deterministic, exact-length regression signal).
        io:println("messageLength: ", failure.message().length());
    }
    // @output messageLength: 149
}
