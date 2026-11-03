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

isolated int counter = 0;

isolated function isolatedFn(int n) returns int {
    return n;
}

public function main() {
    isolated function () returns int f = isolated function () returns int {
        future<int> r = start isolatedFn(42);
        return checkpanic wait r;
    };
    isolated function () returns int g = f;
    lock {
        counter += 1;
        g = isolated function () returns int {
            return isolatedFn(7);
        };
    }
    io:println(f()); // @output 42
    io:println(g()); // @output 7
}
