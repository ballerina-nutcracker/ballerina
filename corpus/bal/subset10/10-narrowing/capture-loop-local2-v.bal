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

// v is declared in the loop body, so every iteration has a fresh binding and no
// earlier iteration's closure can reach it. It is not in the loop's group and
// narrows before the capture in the same iteration.
public function main() {
    int total = 0;
    int i = 0;
    while i < 2 {
        int|string v = i;
        if v is int {
            total += v;
        }
        function () setter = function () {
            v = "changed";
        };
        setter();
        io:println(v); // @output changed
                       // @output changed
        i += 1;
    }
    io:println(total); // @output 1
}
