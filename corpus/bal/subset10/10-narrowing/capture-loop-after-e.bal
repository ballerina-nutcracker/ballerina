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

// A capture made in a loop body stays in effect after the loop, so the variable
// cannot be narrowed once the loop completes.
function whileLoop() {
    int|string x = 1;
    int i = 0;
    while i < 2 {
        function () setter = function () {
            x = "str";
        };
        setter();
        i += 1;
    }
    if x is int {
        int y = x; // @error x is captured in the while body
        io:println(y);
    }
}

function foreachLoop() {
    int|string x = 1;
    foreach int i in 0 ..< 2 {
        function () setter = function () {
            x = i;
        };
        setter();
    }
    if x is int {
        int y = x; // @error x is captured in the foreach body
        io:println(y);
    }
}

public function main() {
    whileLoop();
    foreachLoop();
}
