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

// The async send to a completes without waiting for a to receive it: a first
// receives from b, which waits for the default worker's second send.
public function main() {
    worker a returns int {
        int y = <- b;
        int x = <- function;
        return x + y;
    }
    worker b {
        int z = <- function;
        z * 10 -> a;
    }
    1 -> a;
    2 -> b;
    int r = wait a;
    io:println(r); // @output 21
}
