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

type Config record {|
    int x;
    int y = 2;
|};

type Calculator object {
    public function add(int x, int y = 2) returns int;
    public function configured(*Config config) returns int;
};

class CalculatorImpl {
    *Calculator;

    public function add(int x, int y = 2) returns int {
        return x + y;
    }

    public function configured(*Config config) returns int {
        return config.x + config.y;
    }
}

type Node object {
    public function value(int fallback = 4) returns int;
    public function next() returns Node?;
};

class NodeImpl {
    public function value(int fallback = 4) returns int {
        return fallback;
    }

    public function next() returns Node? {
        return ();
    }
}

object {
    public function add(int x, int y = 2) returns int;
} moduleCalculator = new CalculatorImpl();

function useInline(object {
    public function add(int x, int y = 2) returns int;
} value) returns int {
    return value.add(x = 8);
}

function returnInline() returns object {
    public function add(int x, int y = 2) returns int;
} {
    return new CalculatorImpl();
}

public function main() {
    Calculator calculator = new CalculatorImpl();
    io:println(calculator.add(3)); // @output 5
    io:println(calculator.add(y = 5, x = 4)); // @output 9
    io:println(calculator.configured(x = 6)); // @output 8

    object {
        public function add(int x, int y = 2) returns int;
    } localCalculator = calculator;
    io:println(localCalculator.add(x = 5)); // @output 7
    io:println(moduleCalculator.add(6)); // @output 8
    io:println(useInline(calculator)); // @output 10
    io:println(returnInline().add(y = 4, x = 7)); // @output 11

    Node node = new NodeImpl();
    io:println(node.value()); // @output 4
}
