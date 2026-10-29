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

class Buffer {
    string lexeme = "abc";
    int count = 0;
    int item = 0;
    int value = 0;
    int values = 0;

    function init() {
        int count = 2;
        self.count = count;
    }

    function take() returns string {
        string lexeme = self.lexeme;
        self.lexeme = "";
        return lexeme;
    }

    function capture() returns int[] {
        int count = 3;
        function () returns int next = function () returns int {
            count += 1;
            self.count += 10;
            return count;
        };
        int result = next();
        return [result, count, self.count];
    }

    function lambdas() returns int {
        function (int) returns int required = function (int value) returns int {
            return value + self.value;
        };
        function (int...) returns int rest = function (int... values) returns int {
            return values.length() + self.values;
        };
        return required(5) + rest(1, 2);
    }

    function loop() returns int {
        int total = 0;
        foreach int item in [1, 2] {
            total += item;
            self.item += 10;
        }
        return total + self.item;
    }
}

class Parent {
    int inherited = 7;
}

class Child {
    *Parent;

    function init() {
        self.inherited = 7;
    }

    function read() returns int {
        int inherited = 9;
        return inherited + self.inherited;
    }
}

client class Resource {
    int count = 8;

    resource function get value() returns int {
        int count = 2;
        return count + self.count;
    }
}

public function main() {
    Buffer buffer = new;
    io:println(buffer.count); // @output 2
    io:println(buffer.take()); // @output abc
    io:println(buffer.lexeme == ""); // @output true
    io:println(buffer.capture()); // @output [4,4,12]
    io:println(buffer.lambdas()); // @output 7
    io:println(buffer.loop()); // @output 23
    Child child = new;
    io:println(child.read()); // @output 16
    Resource clientValue = new;
    int result = clientValue->/value;
    io:println(result); // @output 10
}
