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

class NumberIterator {
    int[] items = [1, 2, 3];
    int idx = 0;

    public function next() returns record {|int value;|}? {
        if self.idx >= 3 {
            return ();
        }
        int val = self.items[self.idx];
        self.idx += 1;
        return {value: val};
    }
}

class NumberGenerator {
    *object:Iterable;

    public function iterator() returns NumberIterator {
        return new;
    }
}

public function main() returns error? {
    string[] names = ["John", "Jane", "Doe"];

    foreach var name in names {
        io:println("Hello, ", name); // @output Hello, John
    }
    // @output Hello, Jane
    // @output Hello, Doe

    map<int> scores = {alice: 10, bob: 20};
    foreach var score in scores {
        io:println(score); // @output 10
                           // @output 20
    }

    NumberGenerator gen = new;
    foreach var value in gen {
        io:println(value); // @output 1
                           // @output 2
                           // @output 3
    }

    foreach var i in 0..<3 {
        io:println(i); // @output 0
                       // @output 1
                       // @output 2
    }
}
