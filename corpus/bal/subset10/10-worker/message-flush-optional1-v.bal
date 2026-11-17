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

// a can't fail, so the flush isn't needed to cover the send and the default
// worker may return before it; a still receives the message.
function run(boolean early) returns int {
    worker a returns int {
        return <- function;
    }
    1 -> a;
    if early {
        return 0;
    } else {
    }
    flush a;
    return wait a;
}

public function main() {
    io:println(run(true)); // @output 0
    io:println(run(false)); // @output 1
}
