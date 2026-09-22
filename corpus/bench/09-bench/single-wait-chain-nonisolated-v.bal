// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

const int FutureCount = 10000;

function seed() returns int {
    return 0;
}

function increment(future<int> previous) returns int {
    int result = checkpanic wait previous;
    return result + 1;
}

public function main() returns error? {
    future<int> tail = start seed();
    foreach int _ in 1 ..< FutureCount {
        tail = start increment(tail);
    }
    int result = check wait tail;
    io:println(result); // @output 9999
}
