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

// The copy is a snapshot: invoking the capturing lambda inside the copy's
// narrowed region changes the original but not the copy, so the copy keeps its
// refinement.
public function main() {
    int|string x = 1;
    function () setter = function () {
        x = "changed";
    };
    int|string copy = x;
    if copy is int {
        setter();
        int y = copy + 1;
        io:println(y); // @output 2
        io:println(copy); // @output 1
        io:println(x); // @output changed
    }
}
