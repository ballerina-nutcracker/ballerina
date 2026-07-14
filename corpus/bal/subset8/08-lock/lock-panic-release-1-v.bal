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

// Validates that a static `panic` inside a `lock` body emits a
// matching `LockEnd` before the `Panic` terminator (BIR-gen). After
// `trap` recovers the panic, the same strand re-enters the lock and
// reads/writes the protected variable.
//
// Caveat: the runtime mutex is re-entrant per strand, so on a single
// strand the second `lock` would superficially succeed even without
// the release. The teeth of this fixture are (a) the BIR expected
// output (must show `LockEnd` before `Panic` in `bump`) and (b) the
// multi-strand Go-level test in runtime/internal/exec.
import ballerina/io;

isolated int counter = 0;

function bump() {
    lock {
        counter = counter + 1;
        panic error("boom");
    }
}

public function main() {
    error? e = trap bump();
    io:println(e is error); // @output true
    lock {
        counter = counter + 1;
        io:println(counter); // @output 2
    }
}
