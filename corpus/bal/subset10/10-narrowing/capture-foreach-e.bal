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

// x is captured by a closure in the loop body, so it is never narrowed anywhere
// in the loop, even though no closure can run between this test and the use.
public function main() {
    int|string x = 1;
    foreach int i in 0 ..< 2 {
        if x is int {
            int y = x; // @error a later capture in the loop reaches this use
            io:println(y + i);
        }
        function () setter = function () {
            x = "str";
        };
        setter();
    }
}
