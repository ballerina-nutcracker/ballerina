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

function run(function () f) returns boolean {
    f();
    return true;
}

function flag() returns boolean {
    return true;
}

// The composition carries a capture made by the RHS into the outcome the RHS
// did not produce as well. That is the same conservative direction as the loop
// policy: it can only reject, never accept an unsound refinement.
public function main() {
    int|string x = 1;
    boolean c = flag();
    if c && run(function () {
        x = "changed";
    }) {
        io:println("taken");
    } else {
        if x is int {
            int y = x; // @error the RHS capture reaches the false outcome too
            io:println(y);
        }
    }
}
