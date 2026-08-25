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
import ballerina/lang.map;

int evaluations = 0;

function receiver() returns json {
    evaluations += 1;
    return {value: 42};
}

function printError(string label, json|error value) {
    if value is error {
        io:println(label);
    }
}

function printNil(string label, json|error value) {
    if value is () {
        io:println(label);
    }
}

class Counter {
    public int value = 9;
}

type OpenRecord record {|
    int fixed;
    json...;
|};

type OpenJSONRecord record {|
    json required;
    json...;
|};

public function main() {
    json j = {a: {b: 3}, presentNil: ()};

    json|error required = j.a;
    if required is json {
        io:println(required); // @output {"b":3}
    }

    json|error nested = j.a.b;
    io:println(nested); // @output 3

    json|error parenthesized = (j.a).b;
    io:println(parenthesized); // @output 3

    map<json> outerMap = {a: {b: 4}};
    json|error nestedMap = outerMap.a.b;
    io:println(nestedMap); // @output 4

    map<int?> getMap = {present: 7, nilValue: ()};
    int? getPresent = map:get(getMap, "present");
    io:println(getPresent); // @output 7
    int?|error getNil = trap getMap.get("nilValue");
    if getNil is () {
        io:println("map get present nil"); // @output map get present nil
    }
    int?|error getMissing = trap getMap.get("missing");
    if getMissing is error {
        io:println("map get missing error"); // @output map get missing error
    }

    json|error optionalPresent = j?.a;
    if optionalPresent is json {
        io:println(optionalPresent); // @output {"b":3}
    }

    printError("required missing", j.missing); // @output required missing
    printNil("optional missing", j?.missing); // @output optional missing

    json|error presentNil = j.presentNil;
    if presentNil is () {
        io:println("required present nil"); // @output required present nil
    }
    if j.absent is error {
        io:println("required absent differs"); // @output required absent differs
    }

    json nilReceiver = ();
    printError("required nil receiver", nilReceiver.value); // @output required nil receiver
    printNil("optional nil receiver", nilReceiver?.value); // @output optional nil receiver

    json scalarReceiver = 1;
    printError("required non-mapping receiver", scalarReceiver.value); // @output required non-mapping receiver
    printError("optional non-mapping receiver", scalarReceiver?.value); // @output optional non-mapping receiver

    printError("propagated earlier error", j.missing.next); // @output propagated earlier error

    OpenJSONRecord openJSON = {required: 1, "nested": {value: 12}};
    json openJSONRequired = openJSON.required;
    io:println(openJSONRequired); // @output 1
    map<json> openJSONMap = openJSON;
    json|error openJSONValue = openJSONMap.nested.value;
    io:println(openJSONValue); // @output 12

    json|error evaluatedOnce = receiver().value;
    io:println(evaluatedOnce); // @output 42
    io:println(evaluations); // @output 1

    json user = {
        name: "Tharmigan",
        age: 30
    };
    io:println(user.name); // @output Tharmigan
    io:println(user.age); // @output 30

    strictAccesses();
}

function strictAccesses() {
    record {| int value; |} rec = {value: 8};
    int recordValue = rec.value;
    io:println(recordValue); // @output 8

    Counter counter = new;
    io:println(counter.value); // @output 9

    OpenRecord openRecord = {fixed: 10};
    openRecord.fixed = 11;
    io:println(openRecord.fixed); // @output 11
}
