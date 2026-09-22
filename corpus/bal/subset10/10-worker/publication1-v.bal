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

// Stress sibling-future publication from an isolated function, where workers
// may run in parallel. Each worker reads exactly one sibling slot, so the
// latch must publish every slot before any worker body runs.
isolated function chain(int seed) returns int {
    worker a returns int {
        int fromB = wait b;
        return fromB + 1;
    }
    worker b returns int {
        int fromC = wait c;
        return fromC + 1;
    }
    worker c returns int {
        int fromD = wait d;
        return fromD + 1;
    }
    worker d returns int {
        return seed;
    }
    return wait a;
}

public function main() {
    int total = 0;
    int i = 0;
    while i < 200 {
        total += chain(i);
        i += 1;
    }
    io:println(total); // @output 20500
}
