// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

type IntSupplier function() returns int;

int evaluationCount = 0;
boolean bodyRan = false;

function one() returns int {
    return 1;
}

function two() returns int {
    return 2;
}

function identity(int value) returns int {
    return value;
}

function readFirst(int[] values) returns int {
    return values[0];
}

function delayed() returns int {
    bodyRan = true;
    return 10;
}

isolated function parallel() returns int {
    return 20;
}

function failArgument() returns int {
    panic error("argument");
}

function failBody() returns int {
    panic error("body");
}

function mark() returns int {
    evaluationCount += 1;
    io:println(evaluationCount);
    return evaluationCount;
}

function defaultAndRest(int first, int second = mark(), int... rest) returns int {
    return first + second + rest.length();
}

function named(int first, int second) returns int {
    return first * 10 + second;
}

class Worker {
    private int value;

    function init(int value) {
        self.value = value;
    }

    function run() returns int {
        return self.value;
    }
}

client class Client {
    private int value;

    function init(int value) {
        self.value = value;
    }

    remote function run() returns int {
        return self.value;
    }
}

public function main() {
    future<()> printlnResult = start io:println("started println");
    error? printlnError = wait printlnResult; // @output started println
    io:println(printlnError is ()); // @output true

    int scalar = 5;
    future<int> scalarResult = start identity(scalar);
    scalar = 6;
    int|error scalarValue = wait scalarResult;
    io:println(scalarValue); // @output 5

    IntSupplier selected = one;
    future<int> functionResult = start selected();
    selected = two;
    int|error functionValue = wait functionResult;
    io:println(functionValue); // @output 1

    int captured = 7;
    IntSupplier closure = function() returns int {
        return captured;
    };
    future<int> closureResult = start closure();
    captured = 8;
    int|error closureValue = wait closureResult;
    io:println(closureValue); // @output 8

    Worker target = new (11);
    future<int> methodResult = start target.run();
    target = new (12);
    int|error methodValue = wait methodResult;
    io:println(methodValue); // @output 11

    Client endpoint = new (13);
    future<int> remoteResult = start endpoint->run();
    endpoint = new (14);
    int|error remoteValue = wait remoteResult;
    io:println(remoteValue); // @output 13

    int[] values = [15];
    future<int> referenceResult = start readFirst(values);
    values[0] = 16;
    int|error referenceValue = wait referenceResult;
    io:println(referenceValue); // @output 16

    future<int> defaultResult = start defaultAndRest(mark());
                                                        // @output 1
                                                        // @output 2
    int|error defaultValue = wait defaultResult;
    io:println(defaultValue); // @output 3
    future<int> restResult = start defaultAndRest(mark(), mark(), mark(), mark());
                                                        // @output 3
                                                        // @output 4
                                                        // @output 5
                                                        // @output 6
    int|error restValue = wait restResult;
    io:println(restValue); // @output 9
    future<int> namedResult = start named(second = mark(), first = mark());
                                                        // @output 7
                                                        // @output 8
    int|error namedValue = wait namedResult;
    io:println(namedValue); // @output 78

    var argumentPanic = trap start identity(failArgument());
    io:println(argumentPanic is error); // @output true

    future<int> bodyPanic = start failBody();
    var bodyPanicResult = trap wait bodyPanic;
    io:println(bodyPanicResult is error); // @output true

    future<int> delayedResult = start delayed();
    io:println(bodyRan); // @output false
    int|error delayedValue = wait delayedResult;
    io:println(delayedValue); // @output 10
    io:println(bodyRan); // @output true

    future<int> parallelResult = start parallel();
    int|error parallelValue = wait parallelResult;
    io:println(parallelValue); // @output 20
}
