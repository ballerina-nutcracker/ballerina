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

// Narrowing established inside a fork worker belongs to that worker, so it
// does not apply in the forking worker after the fork (spec 7.12).
function narrowedInForkWorker(int|string arg) returns int {
    final int|string v = arg;
    fork {
        worker w {
            if v !is int {
                return;
            }
            int _ = v;
        }
    }
    wait w;
    int n = v; // @error the narrowing inside w does not reach here
    return n;
}

// A fork statement always completes normally, even when none of its workers
// does, so the branch holding it keeps the narrowing of its condition and
// does not narrow v to never (spec 7.12, normal completion).
function forkCompletesNormally(int|string arg) returns int {
    int|string v = arg;
    if v is string {
        fork {
            worker a {
                return;
            }
            worker b {
                panic error("stop");
            }
        }
    }
    int n = v; // @error v is int|string after the if
    return n;
}

public function main() {
    io:println(narrowedInForkWorker(1));
    io:println(forkCompletesNormally(1));
}
