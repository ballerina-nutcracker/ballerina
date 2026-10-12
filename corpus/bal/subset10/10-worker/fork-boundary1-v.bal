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

// A fork's messages belong to its own workers: the worker or default worker
// that contains the fork terminates without waiting for their delivery.
function inNamedWorker() {
    worker host returns int {
        fork {
            worker a {
                1 -> b;
            }
            worker b returns int {
                return <- a;
            }
        }
        return wait b;
    }
    int r = wait host;
    io:println(r); // @output 1
}

function inTrailingStatements() returns int {
    worker other {
        io:println("other"); // @output other
    }
    wait other;
    fork {
        worker a {
            2 -> b;
        }
        worker b returns int {
            return <- a;
        }
    }
    return wait b;
}

function earlyReturn(boolean early) returns int {
    if early {
        return 0;
    }
    fork {
        worker a {
            3 -> b;
        }
        worker b returns int {
            return <- a;
        }
    }
    return wait b;
}

public function main() {
    inNamedWorker();
    io:println(inTrailingStatements()); // @output 2
    io:println(earlyReturn(true)); // @output 0
    io:println(earlyReturn(false)); // @output 3
}
