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

function flag() returns boolean {
    return false;
}

// The capturing branch cannot complete normally, so the code after the join is
// reachable only through the path with no capture.
public function main() {
    boolean condition = flag();
    int|string x = 1;
    if condition {
        function () setter = function () {
            x = "changed";
        };
        setter();
        return;
    }
    if x is int {
        io:println(x + 1); // @output 2
    }
}
