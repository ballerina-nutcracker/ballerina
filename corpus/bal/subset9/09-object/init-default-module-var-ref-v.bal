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

// Skipped: https://github.com/ballerina-nutcracker/ballerina/issues/915
//
// `base` is typed `never` inside the `init` default value expression, so this
// fails with "incompatible type: expected int, got never" pointing at `base`.
// A default that reads a preceding parameter instead is fine.
final int base = 5;

isolated function twice(int x) returns int => x * 2;

class Holder {
    int n;

    function init(int n = twice(base)) {
        self.n = n;
    }
}

public function main() {
    Holder h = new;
    io:println(h.n); // @output 10

    Holder j = new (3);
    io:println(j.n); // @output 3
}
