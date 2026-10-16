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

type Info record {|
    int value;
|};

type Flag record {|
    boolean value;
|};

annotation Info classInfo on class;
annotation Info serviceInfo on service;
annotation Info fieldInfo on object field;
annotation Info recordFieldInfo on record field;
annotation Flag methodFlag on object function;
annotation Info returnInfo on return;

int mutableValue = 9;

function computeValue() returns int => 42;

@classInfo {value: mutableValue}
isolated class Counter {
    @fieldInfo {value: mutableValue}
    private int count = 0;

    @methodFlag {value: mutableValue > 1}
    isolated function init() {
    }

    @methodFlag {value: computeValue() > 1}
    isolated function get() returns @returnInfo {value: mutableValue} int {
        lock {
            return self.count;
        }
    }
}

@serviceInfo {value: computeValue()}
isolated service class Service {
    @methodFlag {value: mutableValue > 1}
    isolated remote function call() returns @returnInfo {value: computeValue()} int {
        return 1;
    }
}

type Rec record {|
    @recordFieldInfo {value: computeValue()}
    int a;
|};

public function main() {
    Counter counter = new;
    io:println(counter.get()); // @output 0
    Rec rec = {a: 1};
    io:println(rec.a); // @output 1
    Info? classValue = Counter.@classInfo;
    if classValue is Info {
        io:println(classValue.value); // @output 9
    }
    Info? serviceValue = Service.@serviceInfo;
    if serviceValue is Info {
        io:println(serviceValue.value); // @output 42
    }
}
