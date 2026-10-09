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

// A variable declared in the loop body is fresh on every iteration, but once a
// closure in the same iteration captures it, it is no longer narrowed.
public function main() {
    int i = 0;
    while i < 2 {
        int|string v = i;
        function () setter = function () {
            v = "changed";
        };
        setter();
        if v is int {
            int y = v; // @error v is captured earlier in this iteration
            io:println(y);
        }
        i += 1;
    }
}
