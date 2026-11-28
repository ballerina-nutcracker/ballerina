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

// A fork in a block nested in the initialization statements declares its
// workers in that block only, so a function worker declared after it may reuse
// the name and each wait reaches its own worker.
function nestedInitFork(boolean flag) returns int {
    int sum = 0;
    if flag {
        fork {
            worker w returns int {
                return 1;
            }
        }
        sum += wait w;
    }
    worker w returns int {
        return 10;
    }
    int r = wait w;
    return sum + r;
}

public function main() {
    io:println(nestedInitFork(true)); // @output 11
    io:println(nestedInitFork(false)); // @output 10
}
