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

// Message actions on the right of an assignment.
public function main() {
    worker a returns int {
        int x = <- function;
        int y = <- function;
        int z = <- function;
        return x + y + z;
    }
    worker b {
        2 -> function;
        3 -> function;
        4 -> function;
    }
    int x = 0;
    x = <- b;
    _ = <- b;
    int|error y = 0;
    y = checkpanic <- b;
    error? e = ();
    e = 1 ->> a;
    10 -> a;
    e = flush a;
    100 ->> a;
    int r = wait a;
    io:println(x, " ", y, " ", e is (), " ", r); // @output 2 4 true 111
}
