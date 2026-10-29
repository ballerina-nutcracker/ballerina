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

isolated client class Client {
    isolated resource function get [int a]/[int b](int c) returns int {
        return a * 100 + b * 10 + c;
    }
}

function seg() returns int {
    return 1;
}

public function main() {
    Client c = new;
    future<int> r = start c->/[seg()]/[2](3);
    int|error v = wait r;
    io:println(v); // @output 123
}
