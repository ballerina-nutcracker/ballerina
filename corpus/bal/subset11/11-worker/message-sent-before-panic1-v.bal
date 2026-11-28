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
    panic error("a panicked");
}

// a panics only after its send, so the receive gets the queued message rather
// than a's panic.
function exchange() returns int {
    worker a {
        1 -> function;
        boom();
    }
    return <- a;
}

public function main() {
    int total = 0;
    foreach int _ in 0 ..< 20 {
        total += exchange();
    }
    io:println(total); // @output 20
}
