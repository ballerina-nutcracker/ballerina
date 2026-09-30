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

// Capturing x un-narrows it, so the closure body sees int|string rather than the
// enclosing int refinement. x is reassigned to a string before the call, so
// treating it as an int inside the closure would be unsound.
// The loop forces a binding-chain merge inside the closure, which must not
// drop the function boundary.
public function main() {
    int|string x = 1;
    if x is int {
        function () returns int f = function () returns int {
            foreach int i in 0 ..< 1 {
                io:println(i);
            }
            return x; // @error x is not narrowed inside the capturing closure
        };
        x = "changed";
        io:println(f() + 1);
    }
}
