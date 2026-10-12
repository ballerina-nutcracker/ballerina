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

function beforeFork() {
    int r = wait w; // @error w isn't in scope before the fork
    fork {
        worker w returns int {
            return 1;
        }
    }
    io:println(r);
}

function outsideBlock(boolean flag) {
    if flag {
        fork {
            worker w returns int {
                return 1;
            }
        }
    }
    int r = wait w; // @error w isn't in scope outside the fork's block
    io:println(r);
}

function shadowsLocal() {
    int w = 1;
    fork {
        worker w returns int { // @error Variable already defined: w
            return 2;
        }
    }
    io:println(w);
}

// A fork in the initialization statements declares its workers in the
// function body's scope, where the function's own workers are declared too.
function sameNameAsFunctionWorker() {
    fork {
        worker w returns int {
            return 1;
        }
    }
    worker w returns int { // @error Variable already defined: w
        return 2;
    }
}

function twoForksInOneBlock() {
    fork {
        worker w returns int {
            return 1;
        }
    }
    fork {
        worker w returns int { // @error Variable already defined: w
            return 2;
        }
    }
}
