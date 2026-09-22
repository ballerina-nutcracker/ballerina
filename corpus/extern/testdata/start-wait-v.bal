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

isolated function strandId() returns int = external;

isolated function startedStrandId() returns int {
    return strandId();
}

function queuedStrandId(int value) returns int {
    io:println(value);
    return strandId();
}

public function main() {
    int parent = strandId();
    future<int> childFuture = start startedStrandId();
    int|error child = wait childFuture;
    io:println(child is int && parent != child); // @output true

    future<int> first = start queuedStrandId(1);
    future<int> second = start queuedStrandId(2);
    io:println(0); // @output 0
    int|error secondId = wait second;
    int|error firstId = wait first;
    io:println(firstId is int && secondId is int &&
            parent != firstId && parent != secondId && firstId != secondId); // @output 1
                                                                            // @output 2
                                                                            // @output true
}
