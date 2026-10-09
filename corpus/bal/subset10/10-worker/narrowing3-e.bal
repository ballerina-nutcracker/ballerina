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

// Workers run concurrently with each other and with the rest of the default
// worker, so a mutable variable any worker captures cannot be narrowed in any
// worker body or in the default worker statements after the workers.
function narrowedInOwnBody(int|string arg) returns int {
    int|string v = arg;
    worker w returns int {
        if v is int {
            return v; // @error v is captured by this worker
        }
        return 0;
    }
    return wait w;
}

function narrowedInSiblingBody(int|string arg) returns int {
    int|string v = arg;
    worker a {
        v = "no longer an int";
    }
    worker b returns int {
        if v is int {
            return v; // @error v is captured by worker a
        }
        return 0;
    }
    return wait b;
}

function narrowedAfterWorkers(int|string arg) returns int {
    int|string v = arg;
    worker w {
        io:println(v);
    }
    if v is int {
        return v; // @error v is captured by worker w
    }
    return 0;
}

function finalStillNarrows(int|string arg) returns int {
    final int|string v = arg;
    worker w returns int {
        if v is int {
            return v;
        }
        return 0;
    }
    return wait w;
}

public function main() {
    io:println(narrowedInOwnBody(1));
    io:println(narrowedInSiblingBody(1));
    io:println(narrowedAfterWorkers(1));
    io:println(finalStillNarrows(1));
}
