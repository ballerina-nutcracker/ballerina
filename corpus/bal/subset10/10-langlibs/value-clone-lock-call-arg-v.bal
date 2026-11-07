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

isolated int[] data = [];

isolated function twice(int[] xs) returns int[] => [...xs, ...xs];

isolated function store(int[] xs) {
    lock {
        data = twice(xs.clone());
    }
}

isolated function get() returns int[] {
    lock {
        return data.clone();
    }
}

public function main() {
    int[] xs = [1, 2];
    store(xs);
    xs.push(3);
    io:println(get()); // @output [1,2,1,2]
}
