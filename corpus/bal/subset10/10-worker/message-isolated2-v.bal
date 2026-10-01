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

// In an isolated function every worker runs on its own thread; here two
// workers and the default worker all wait for delivery of their messages.
isolated function exchange() returns int {
    worker a returns int {
        1 -> function;
        return 10;
    }
    worker b returns int {
        2 -> function;
        return 20;
    }
    int x = <- a;
    int y = <- b;
    int p = wait a;
    int q = wait b;
    return x + y + p + q;
}

public function main() {
    int total = 0;
    foreach int _ in 0 ..< 20 {
        total += exchange();
    }
    io:println(total); // @output 660
}
