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

function twoReferences() {
    fork {
        worker w returns int {
            return 1;
        }
    }
    int a = wait w;
    future<int> b = w; // @error second reference
    io:println(a);
    io:println(b);
}

function nestedLoop() {
    foreach int _ in 0 ..< 2 {
        fork {
            worker w returns int {
                return 1;
            }
        }
        foreach int _ in 0 ..< 2 {
            int r = wait w; // @error reference inside a loop nested after the fork
            io:println(r);
        }
    }
}

function insideLambda() {
    fork {
        worker w returns int {
            return 1;
        }
    }
    var f = function () returns int {
        return wait w; // @error reference inside an anonymous function
    };
    io:println(f());
}

function forkInLoopRefersToFunctionWorker() {
    worker named returns int {
        return 1;
    }
    foreach int _ in 0 ..< 2 {
        fork {
            worker f returns int {
                return wait named; // @error each iteration starts a new f
            }
        }
        int r = wait f;
        io:println(r);
    }
}
