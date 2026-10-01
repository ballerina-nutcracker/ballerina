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
    panic error("b panicked");
}

// b panics before receiving, so a can't deliver its async send and terminates
// abnormally with b's termination value.
public function main() {
    worker a {
        1 -> b;
    }
    worker b {
        boom();
        int _ = <- a;
    }
    error? ra = trap wait a;
    io:println(ra is error ? ra.message() : "a done"); // @output b panicked
    error? rb = trap wait b;
    io:println(rb is error ? rb.message() : "b done"); // @output b panicked
}
