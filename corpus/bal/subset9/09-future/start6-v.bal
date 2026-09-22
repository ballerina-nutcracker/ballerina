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

int count = 0;

client class Client {
    resource function get [int first]/[int second](int third) returns int {
        return first * 100 + second * 10 + third;
    }
}

function receiver(Client value) returns Client {
    io:println(0); // @output 0
    return value;
}

function next() returns int {
    count += 1;
    // @output 1
    // @output 2
    // @output 3
    io:println(count);
    return count;
}

public function main() {
    Client endpoint = new;
    future<int> result = start receiver(endpoint)->/[next()]/[next()](next());
    int|error value = wait result;
    io:println(value); // @output 123
}
