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
    private int index = 0;
    private int[] values;

    function init(int[] values) {
        self.values = values;
    }

    public isolated function next() returns record {|int value;|}? {
        if self.index >= self.values.length() {
            return ();
        }
        int value = self.values[self.index];
        self.index += 1;
        return {value};
    }
}

class Numbers {
    *object:Iterable;
    private int[] values;

    function init(int[] values) {
        self.values = values;
    }

    public function iterator() returns NumberIterator {
        return new (self.values);
    }
}

class FailingIterator {
    private int index = 0;

    public isolated function next() returns record {|int value;|}|error? {
        if self.index < 2 {
            self.index += 1;
            return {value: self.index * 10};
        }
        return error("iterator completion failed");
    }
}

class FailingNumbers {
    *object:Iterable;

    public function iterator() returns FailingIterator {
        return new;
    }
}

public function main() {
    int[] doubled = from var value in new Numbers([1, 2, 3])
        select value * 2;
    io:println(doubled); // @output [2,4,6]

    stream<int, ()> numbers = new (new NumberIterator([4, 5, 6]));
    int[] filtered = from var value in numbers
        where value != 5
        select value;
    io:println(filtered); // @output [4,6]

    [int, int][] pairs = from var left in [10, 20]
        join var right in new Numbers([20, 30])
        on left equals right
        select [left, right];
    io:println(pairs); // @output [[20,20]]

    stream<int, error?> failing = new (new FailingIterator());
    int[]|error streamed = from var value in failing
        select value;
    io:println(streamed is error); // @output true
    if streamed is error {
        io:println(streamed.message()); // @output iterator completion failed
    }

    int[]|error ordered = from var value in new FailingNumbers()
        order by value
        select value;
    io:println(ordered is error); // @output true

    int|error failedCount = from var value in new FailingNumbers()
        collect value.length();
    io:println(failedCount is error); // @output true

    int count = from var value in new Numbers([7, 8])
        collect value.length();
    io:println(count); // @output 2
}
