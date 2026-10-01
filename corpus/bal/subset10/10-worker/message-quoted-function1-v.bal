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

// A worker quoted as 'function is an ordinary worker; `function` still names
// the default worker.
public function main() {
    worker 'function returns int {
        int x = <- function;
        x + 1 -> function;
        return x;
    }
    worker w returns int {
        record {int 'function;} r = <- {function};
        return r.'function;
    }
    1 -> 'function;
    int y = <- 'function;
    io:println(y); // @output 2
    y * 10 -> w;
    int z = wait w;
    io:println(z); // @output 20
}
