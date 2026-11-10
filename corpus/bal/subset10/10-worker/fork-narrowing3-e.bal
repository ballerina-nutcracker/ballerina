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

// The forking worker runs concurrently with the fork's workers, so its
// assignments after the fork may happen before a worker's read (spec 7.12).
function narrowedBeforeForkAssignedAfter(int|string arg) returns int {
    int|string v = arg;
    if v is int {
        fork {
            worker w returns int {
                return v; // @error v may have been assigned by the forking worker
            }
        }
        v = "x";
        return wait w;
    }
    return 0;
}

function narrowedInWorkerAssignedAfter(int|string arg) returns int {
    int|string v = arg;
    fork {
        worker w returns int {
            if v is int {
                return v; // @error v may have been assigned by the forking worker
            }
            return 0;
        }
    }
    v = "x";
    return wait w;
}

public function main() {
    io:println(narrowedBeforeForkAssignedAfter(1));
    io:println(narrowedInWorkerAssignedAfter(1));
}
