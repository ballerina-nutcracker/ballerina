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

// A closure nested inside another closure in the loop still captures the outer
// local, and a lambda's own loop restricts the lambda's own locals.
public function main() {
    int|string x = 1;
    int i = 0;
    while i < 2 {
        if x is int {
            int y = x; // @error x is captured by the inner closure
            io:println(y);
        }
        function () run = function () {
            int|string own = 1;
            foreach int j in 0 ..< 2 {
                if own is int {
                    int y = own; // @error own is captured in this loop
                    io:println(y + j);
                }
                function () inner = function () {
                    x = "x";
                    own = "own";
                };
                inner();
            }
        };
        run();
        i += 1;
    }
}
