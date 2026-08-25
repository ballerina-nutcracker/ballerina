// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the
// License at
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

int lhsCalls = 0;
int rhsCalls = 0;

function lhs(int? value) returns int? {
    lhsCalls += 1;
    return value;
}

function rhs() returns int {
    rhsCalls += 1;
    return 42;
}

function abort() returns never {
    panic error("failed");
}

function neverResult() {
    never _ = abort() ?: 1;
}

function mixedResult(int|string? value) returns int|string {
    return value ?: 1;
}

function nilSubtype(null|null value) returns int {
    return value ?: 70;
}

function optionalNever(never? value) returns int {
    return value ?: 80;
}

function checkedValue(boolean good) returns int|error {
    return good ? 90 : error("checked");
}

function checkedRhs(int? lhs, boolean good) returns int|error {
    return lhs ?: check checkedValue(good);
}

function checkedOuter(int?|error lhs) returns int|error {
    return check (lhs ?: 100);
}

function nullableCheckedValue(boolean good) returns int?|error {
    return good ? 110 : error("panic");
}

public function main() {
    int nonNil = lhs(10) ?: rhs();
    int? nonNilComposite = lhs(30) ?: lhs(40) + 1;
    int fallback = lhs(()) ?: rhs();
    string staticallyNonNil = "value" ?: 100;
    () nilValue = ();
    int staticallyNil = nilValue ?: 50;
    int nested = lhs(()) ?: lhs(20) ?: rhs();
    int lazyOuterBranch = false ? lhs(30) ?: rhs() : 60;
    int|string mixedString = mixedResult("mixed");
    int|string mixedNil = mixedResult(());
    int nilSubtypeResult = nilSubtype(());
    int optionalNeverResult = optionalNever(());
    int|error lazyCheck = checkedRhs(120, false);
    int|error failedCheck = checkedRhs((), false);
    int|error outerNil = checkedOuter(());
    int|error outerError = checkedOuter(error("outer"));
    int|error lazyCheckpanic = trap (130 ?: (checkpanic nullableCheckedValue(false)) + 1);
    int?|error trappedCheckpanic = trap (() ?: (checkpanic nullableCheckedValue(false)) + 1);
    int|error lhsCheckpanic = trap ((checkpanic nullableCheckedValue(false)) ?: 140);
    int?|error wrappedLhsCheckpanic = trap (((checkpanic nullableCheckedValue(false)) + 1) ?: 150);
    int|error outerCheckpanic = trap checkpanic (() ?: checkedValue(false));

    io:println(nonNil); // @output 10
    io:println(nonNilComposite); // @output 30
    io:println(fallback); // @output 42
    io:println(staticallyNonNil); // @output value
    io:println(staticallyNil); // @output 50
    io:println(nested); // @output 20
    io:println(lazyOuterBranch); // @output 60
    io:println(mixedString); // @output mixed
    io:println(mixedNil); // @output 1
    io:println(nilSubtypeResult); // @output 70
    io:println(optionalNeverResult); // @output 80
    io:println(lazyCheck); // @output 120
    io:println(failedCheck is error); // @output true
    io:println(outerNil); // @output 100
    io:println(outerError is error); // @output true
    io:println(lazyCheckpanic); // @output 130
    io:println(trappedCheckpanic is error); // @output true
    io:println(lhsCheckpanic is error); // @output true
    io:println(wrappedLhsCheckpanic is error); // @output true
    io:println(outerCheckpanic is error); // @output true
    io:println(lhsCalls); // @output 5
    io:println(rhsCalls); // @output 1
}
