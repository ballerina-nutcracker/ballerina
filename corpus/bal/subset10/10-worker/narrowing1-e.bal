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

// A worker runs concurrently with the rest of the default worker, so a
// variable it captures may be assigned at any point after the startup point.
// Narrowing established before the workers start does not survive them.
function capturedAssignment(int|string arg) returns int {
    int|string v = arg;
    if v !is int {
        return 0;
    }
    worker w returns int {
        v = "no longer an int";
        return 1;
    }
    int r = wait w;
    return v + r; // @error v is int|string again because the worker captures it
}

public function main() {
    io:println(capturedAssignment(1));
}
