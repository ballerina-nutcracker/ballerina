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

boolean started = false;

function early(boolean stop) returns int {
    if stop {
        return 0;
    }
    worker w returns int {
        started = true;
        return 1;
    }
    return wait w;
}

function other() returns int {
    return 2;
}

public function main() {
    // Returning from default-worker-init ends the function before its named
    // workers would start, so w never runs. The wait on f yields, which would
    // let w run had it been started.
    io:println(early(true)); // @output 0
    future<int> f = start other();
    int|error r = wait f;
    io:println(r); // @output 2
    io:println(started); // @output false
    io:println(early(false)); // @output 1
    io:println(started); // @output true
}
