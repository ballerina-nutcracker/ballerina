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

// A sibling's name is in scope in a fork worker's body, so a local there
// cannot reuse it.
function localNamedLikeSibling() {
    fork {
        worker a {
            int b = 1; // @error Variable already defined: b
            io:println(b);
        }
        worker b {
        }
    }
}

// A fork worker's name is in scope in the blocks nested after the fork.
function localInNestedBlockAfterFork(boolean flag) {
    fork {
        worker w {
        }
    }
    if flag {
        int w = 1; // @error Variable already defined: w
        io:println(w);
    }
}

// A fork worker body is nested in the enclosing function, so its locals cannot
// shadow the function's locals.
function workerLocalShadowsEnclosingLocal() {
    int x = 1;
    fork {
        worker w {
            int x = 2; // @error Variable already defined: x
            io:println(x);
        }
    }
    io:println(x);
}

// A function's worker name is in scope in its trailing statements, including
// a fork nested in them.
function forkInTrailingBlockReusesFunctionWorkerName(boolean flag) {
    worker w {
    }
    if flag {
        fork {
            worker w { // @error Variable already defined: w
            }
        }
    }
}

function collidesWithParameter(int w) {
    fork {
        worker w { // @error Variable already defined: w
        }
    }
    io:println(w);
}
