// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;

int conditionCalls = 0;
int optionalBaseCalls = 0;
int defaultCalls = 0;

type Value record {|
    int x?;
|};

type IntFn function() returns int;

function panicValues() returns int[] {
    panic error("inside thunk");
}

function conditionValues() returns int[] {
    conditionCalls += 1;
    if conditionCalls < 4 {
        return [conditionCalls];
    }
    return [];
}

function optionalBase() returns Value {
    optionalBaseCalls += 1;
    return {x: 42};
}

function nextDefault() returns int {
    defaultCalls += 1;
    return defaultCalls;
}

function defaulted(int value = nextDefault()) returns int {
    return value;
}

function queryDefault(int[] values = from int value in [1, 2]
        select value * 2) returns int[] {
    return values;
}

function spreadDefault(int[] values = [0, ...[1, 2]]) returns int[] {
    return values;
}

class CaptureCounter {
    private int value = 0;

    function run() returns int {
        int local = 1;
        IntFn outerFn = function () returns int {
            IntFn inner = function () returns int {
                local += 1;
                self.value += local;
                return self.value;
            };
            int[] values = [inner(), ...[inner()]];
            return values[1];
        };
        return outerFn();
    }
}

public function main() {
    error|int[] trapped = trap [0, ...panicValues()];
    io:println(trapped is error); // @output true

    boolean andResult = false && [0, ...panicValues()].length() > 0;
    boolean orResult = true || [0, ...panicValues()].length() > 0;
    io:println(andResult, ":", orResult); // @output false:true

    int loopCount = 0;
    while [0, ...conditionValues()].length() > 1 {
        loopCount += 1;
    }
    io:println(loopCount, ":", conditionCalls); // @output 3:4

    io:println(defaulted(), ":", defaulted(), ":", defaultCalls); // @output 1:2:2
    io:println(queryDefault()); // @output [2,4]
    io:println(spreadDefault()); // @output [0,1,2]

    int? optionalValue = (optionalBase())?.x;
    io:println(optionalValue, ":", optionalBaseCalls); // @output 42:1

    error|int[] checkpanicTrapped = trap [0, ...checkpanic panicValues()];
    io:println(checkpanicTrapped is error); // @output true

    CaptureCounter counter = new;
    io:println(counter.run()); // @output 5
}
