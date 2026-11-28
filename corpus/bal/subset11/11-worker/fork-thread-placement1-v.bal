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

function spin() returns int {
    int i = 0;
    int acc = 0;
    while i < 100000 {
        acc += i;
        i += 1;
    }
    return acc;
}

// main is not isolated, so the fork's worker runs on the forking worker's
// thread and can't run until main yields at the wait (spec 7.2, 7.5).
public function main() {
    int x = 0;
    fork {
        worker w {
            x = 1;
        }
    }
    int _ = spin();
    int observed = x;
    wait w;
    io:println(observed); // @output 0
    io:println(x); // @output 1
}
