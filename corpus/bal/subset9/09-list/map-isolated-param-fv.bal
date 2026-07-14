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
import ballerina/lang.array;

function increment(int value) returns int {
    return value + 1;
}

isolated function isolatedIncrement(int value) returns int {
    return value + 1;
}

isolated function mapInIsolatedFunction(int[] values) returns int[] {
    int[] explicitResult = values.map(isolatedIncrement);
    return explicitResult.map((value) => value + 1);
}

function mapInsideLock(int[] values) returns int {
    lock {
        int explicitLength = values.map(isolatedIncrement).length();
        int inferredLength = values.map((value) => value + 1).length();
        return explicitLength + inferredLength;
    }
}

type MappedRecord record {|
    int[] values = [1, 2].map(isolatedIncrement);
|};

type MapResult record {|
    int value;
|};

type ValueRecord record {|
    int value;
|};

class NonIsolatedInit {
    function init() {
        int _ = increment(0);
    }
}

isolated function mapRecordFields(ValueRecord[] values) returns int[] {
    return values.map(value => value.value);
}

isolated function mapWithFunctionReference(int[] values) returns int[] {
    return values.map(value => [value].map(isolatedIncrement)[0]);
}

function createFactory() returns function () returns NonIsolatedInit {
    function () returns NonIsolatedInit factory = () => new;
    return factory;
}

isolated function withMappedDefault(int[] values = [2].map((value) => value + 1)) returns int {
    return values[0];
}

function mapWithCapture(int[] values, int offset) returns int[] {
    return values.map(function(int value) returns int {
        return value + offset;
    });
}

function mapWithInferredCapture(int[] values) returns int[] {
    int[] state = [20];
    return values.map((value) => value + state[0]);
}

public function main() {
    int[] values = [1, 2];
    io:println(values.map(increment)); // @output [2,3]
    io:println(values.map(function(int value) returns boolean {
        return value > 1;
    })); // @output [false,true]
    io:println(values.map((value) => value + 2)); // @output [3,4]
    int[] mutableResult = values.map(increment);
    mutableResult.push(10);
    io:println(mutableResult); // @output [2,3,10]
    io:println(array:map(arr = values, func = isolatedIncrement)); // @output [2,3]
    io:println(values.map(func = isolatedIncrement)); // @output [2,3]
    io:println(mapInIsolatedFunction(values)); // @output [3,4]
    io:println(mapInsideLock(values)); // @output 4
    MappedRecord recordValue = {};
    io:println(recordValue.values); // @output [2,3]
    io:println(withMappedDefault()); // @output 3
    io:println(mapWithCapture(values, 10)); // @output [11,12]
    io:println(mapWithInferredCapture(values)); // @output [21,22]
    io:println(mapRecordFields([{value: 4}, {value: 5}])); // @output [4,5]
    io:println(mapWithFunctionReference(values)); // @output [2,3]
    function () returns NonIsolatedInit _ = createFactory();
    io:println("created"); // @output created
    MapResult[] recordResults = values.map((value) => {value: value});
    io:println(recordResults); // @output [{"value":1},{"value":2}]
    io:println((<int[]>[]).map(increment)); // @output []
    var mappedEmpty = [].map(increment);
    mappedEmpty.push(1);
    io:println(mappedEmpty); // @output [1]
    [int, string] tupleValue = [1, "one"];
    io:println(tupleValue.map((value) => value is int)); // @output [true,false]
}
