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
    int[] items = [7, 8, 9];
    int index = 0;

    public function next() returns record {|int value;|}? {
        if self.index >= self.items.length() {
            return ();
        }
        int value = self.items[self.index];
        self.index += 1;
        return {value: value};
    }
}

class NumberGenerator {
    *object:Iterable;

    public function iterator() returns NumberIterator {
        return new;
    }
}

function capturedSum((function () returns int)[] functions) returns int {
    int sum = 0;
    foreach function () returns int fn in functions {
        sum += fn();
    }
    return sum;
}

public function main() {
    (function () returns int)[] functions = [];
    foreach int value in 0 ... 3 {
        if value == 1 {
            continue;
        }
        functions.push(function () returns int { return value; });
    }
    io:println(capturedSum(functions)); // @output 5

    functions = [];
    foreach int value in [1, 2, 3] {
        functions.push(function () returns int { return value; });
    }
    io:println(capturedSum(functions)); // @output 6

    functions = [];
    map<int> values = {a: 4, b: 5, c: 6};
    foreach int value in values {
        functions.push(function () returns int { return value; });
    }
    io:println(capturedSum(functions)); // @output 15

    functions = [];
    NumberGenerator generator = new;
    foreach int value in generator {
        functions.push(function () returns int { return value; });
    }
    io:println(capturedSum(functions)); // @output 24

    (function () returns xml)[] xmlFunctions = [];
    xml<xml:Element|xml:Text|xml:Comment> sequence = xml `<item/>text<!--comment-->`;
    foreach xml:Element|xml:Text|xml:Comment item in sequence {
        xmlFunctions.push(function () returns xml { return item; });
    }
    foreach function () returns xml fn in xmlFunctions {
        io:println(fn()); // @output <item/>
                          // @output text
                          // @output <!--comment-->
    }
}
