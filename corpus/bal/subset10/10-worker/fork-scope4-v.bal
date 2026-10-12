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

// A fork worker's name isn't in scope in its own body, so the body can declare
// a local with that name; after the fork the name refers to the worker again.
function localNamedLikeWorker() returns int {
    fork {
        worker w returns int {
            int w = 5;
            return w + 1;
        }
    }
    return wait w;
}

// A fork nested in a fork worker's body can declare a worker with the same name
// as the enclosing worker; after the nested fork the name refers to the nested
// worker, and after the outer fork to the outer worker.
function nestedForkWorkerNamedLikeWorker() returns int {
    fork {
        worker w returns int {
            fork {
                worker w returns int {
                    return 20;
                }
            }
            int inner = wait w;
            return inner + 1;
        }
    }
    return wait w;
}

// A lambda in a fork worker's body can take a parameter named like the worker.
function lambdaParamNamedLikeWorker() returns int {
    fork {
        worker w returns int {
            var f = function(int w) returns int {
                return w * 2;
            };
            return f(150);
        }
    }
    return wait w;
}

public function main() {
    io:println(localNamedLikeWorker()); // @output 6
    io:println(nestedForkWorkerNamedLikeWorker()); // @output 21
    io:println(lambdaParamNamedLikeWorker()); // @output 300
}
