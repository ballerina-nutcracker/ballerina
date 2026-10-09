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

function flag(int i) returns boolean {
    return i == 1;
}

// Known limitation: the closure is only created on the iteration that breaks
// out of the loop, so no later iteration can observe it. The loop's group does
// not look at control flow, so x still cannot be narrowed in the loop body.
public function main() {
    int|string x = 1;
    function ()? saved = ();
    int i = 0;
    while i < 3 {
        if x is int {
            int y = x; // @error the loop's group includes x
            io:println(y);
        }
        if flag(i) {
            saved = function () {
                x = "x";
            };
            break;
        }
        i += 1;
    }
    if saved is function () {
        saved();
    }
    io:println(x);
}
