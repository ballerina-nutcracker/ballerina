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

// A worker body has its own definite-assignment analysis for its own locals,
// and captures are checked against the state at the startup point.
function workerLocal() returns int {
    worker w returns int {
        int local;
        return local; // @error worker local read before assignment
    }
    return wait w;
}

function capturedLocal() returns int {
    int seed;
    worker w returns int {
        return seed * 2; // @error captured variable is not initialized at the startup point
    }
    seed = 5;
    return wait w;
}

public function main() {
    io:println(workerLocal());
    io:println(capturedLocal());
}
