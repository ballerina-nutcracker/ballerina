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

// The function's workers exchange messages, but the fork's worker isn't one of
// them, so waiting on it inside a nested block is allowed.
function inMessagingNamedWorker(boolean flag) returns int {
    worker host returns int {
        int v = <- function;
        fork {
            worker a returns int {
                return v;
            }
        }
        if flag {
            return wait a;
        }
        return 0;
    }
    5 -> host;
    return wait host;
}

function inMessagingDefaultWorker(boolean flag) returns int {
    worker other {
        1 -> function;
    }
    int v = <- other;
    fork {
        worker a returns int {
            return v + 1;
        }
    }
    if flag {
        return wait a;
    }
    return 0;
}

public function main() {
    io:println(inMessagingNamedWorker(true)); // @output 5
    io:println(inMessagingNamedWorker(false)); // @output 0
    io:println(inMessagingDefaultWorker(true)); // @output 2
    io:println(inMessagingDefaultWorker(false)); // @output 0
}
