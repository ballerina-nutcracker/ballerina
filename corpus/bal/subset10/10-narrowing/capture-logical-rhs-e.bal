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

function flag(function () f) returns boolean {
    f();
    return true;
}

// A capture in an evaluated RHS operand reaches the region the composition
// selects, so the narrowed use inside it is rejected.
public function main() {
    int|string x = 1;
    int|string y = 2;
    if y is int && flag(function () {
        x = "changed";
    }) {
        if x is int {
            int z = x; // @error x is captured by the logical RHS operand
            io:println(z);
        }
    }
}
