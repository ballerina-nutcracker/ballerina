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

function boom() {
    panic error("default panicked");
}

future<int>? pending = ();

// The default worker panics after its async send; the message is already
// queued, so a receives it.
function run() {
    worker a returns int {
        int x = <- function;
        return x * 2;
    }
    pending = a;
    5 -> a;
    boom();
}

public function main() {
    error? e = trap run();
    io:println(e is error ? e.message() : "done"); // @output default panicked
    future<int>? f = pending;
    if f is future<int> {
        int|error v = wait f;
        io:println(v); // @output 10
    }
}
