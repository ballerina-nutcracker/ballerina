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

// A fork worker's name is in scope in the rest of the block containing the
// fork, including the workers of a later fork and nested blocks.
function laterForkWorkerWaits() returns int {
    fork {
        worker a returns int {
            return 1;
        }
    }
    fork {
        worker b returns int {
            int x = wait a;
            return x + 1;
        }
    }
    return wait b;
}

function nestedBlockWaits(boolean flag) returns int {
    fork {
        worker a returns int {
            return 3;
        }
    }
    if flag {
        int x = wait a;
        return x;
    }
    return 0;
}

public function main() {
    io:println(laterForkWorkerWaits()); // @output 2
    io:println(nestedBlockWaits(true)); // @output 3
    io:println(nestedBlockWaits(false)); // @output 0
}
