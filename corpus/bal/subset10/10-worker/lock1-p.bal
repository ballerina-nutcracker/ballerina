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

isolated int counter = 0;

// A worker declarator can only appear at the top of a function body, never
// inside a lock statement, so worker startup while a lock is held is only
// reachable across a call. The existing dynamic start check rejects it.
isolated function spawner() returns int {
    worker w returns int { // @panic
        return 1;
    }
    return wait w;
}

public function main() {
    int result;
    lock {
        counter = spawner();
        result = counter;
    }
    io:println(result);
}
