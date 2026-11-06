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

int counter = 0;

function spawn() {
    worker w {
        counter = 1;
    }
}

function yieldOnce() returns int {
    return 0;
}

// spawn returns without waiting for its named worker, and nothing ever refers
// to the worker, yet the worker was started and runs to completion: the loop
// yields until it has.
public function main() {
    spawn();
    while counter == 0 {
        future<int> f = start yieldOnce();
        int|error r = wait f;
        _ = r is int;
    }
    io:println(counter); // @output 1
}
