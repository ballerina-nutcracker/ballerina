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

class Counter {
    int value;

    function init(int initial = seed + 1) {
        self.value = initial;
    }

    function bump(int amount = step) returns int {
        self.value += amount;
        return self.value;
    }

    function scale(int factor = labels.length()) returns int {
        return self.value * factor;
    }
}

int seed = 7;
int step = 3;
final string[] labels = ["a", "b"];

public function main() {
    Counter c = new;
    io:println(c.value); // @output 8
    io:println(c.bump()); // @output 11
    io:println(c.bump(5)); // @output 16
    io:println(c.scale()); // @output 32

    Counter d = new (100);
    io:println(d.value); // @output 100
}
