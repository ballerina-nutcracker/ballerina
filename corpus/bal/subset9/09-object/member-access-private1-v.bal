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

class A {
    private int x;

    function init(int x) {
        self.x = x;
    }

    private function double() returns int {
        return self.x * 2;
    }

    function sum(A other) returns int {
        return self.x + other.x + other.double();
    }

    function sumAll(A[] others) returns int {
        int total = 0;
        foreach A other in others {
            total += other.x;
        }
        return total;
    }

    function adder() returns function (A) returns int {
        return function(A other) returns int {
            return self.x + other.double();
        };
    }
}

public function main() {
    A a = new (1);
    A b = new (2);
    io:println(a.sum(b)); // @output 7
    io:println(a.sumAll([a, b])); // @output 3
    function (A) returns int adder = a.adder();
    io:println(adder(b)); // @output 5
}
