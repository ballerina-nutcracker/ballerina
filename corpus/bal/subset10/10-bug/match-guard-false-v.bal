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

function guarded(int v, boolean b) {
    match v {
        _ if b => {
            io:println("guarded");
        }
    }
    io:println("after");
}

function guardedThenConst(int v, boolean b) returns string {
    match v {
        1 if b => {
            return "guarded one";
        }
        1 => {
            return "one";
        }
    }
    return "none";
}

function side(string s) returns boolean {
    io:println("guard " + s);
    return true;
}

function guardThenDefault(int v, boolean b) returns int {
    match v {
        1 if b => {
            return 10;
        }
        _ => {
            return 20;
        }
    }
}

function guardedExhaustive(boolean v, boolean b) returns int {
    match v {
        true if b => {
            return 1;
        }
        true => {
            return 2;
        }
        false => {
            return 3;
        }
    }
}

function failingGuard() returns boolean|error {
    return error("guard error");
}

function checkGuard(int v) returns error? {
    match v {
        _ if check failingGuard() => {
            io:println("check body");
        }
    }
    io:println("check after");
}

function shortCircuit(int v, boolean b) {
    match v {
        1|2 if b && side("short") => {
            io:println("short body");
        }
        _ => {
            io:println("short default");
        }
    }
}

public function main() {
    guarded(1, true); // @output guarded
                      // @output after
    guarded(1, false); // @output after
    io:println(guardedThenConst(1, true)); // @output guarded one
    io:println(guardedThenConst(1, false)); // @output one
    io:println(guardedThenConst(2, false)); // @output none
    io:println(guardThenDefault(1, true), guardThenDefault(1, false)); // @output 1020
    io:println(guardedExhaustive(true, true), guardedExhaustive(true, false), guardedExhaustive(false, true)); // @output 123
    error? e = checkGuard(1);
    io:println(e is error); // @output true
    shortCircuit(1, false); // @output short default
    shortCircuit(3, true); // @output short default
    shortCircuit(2, true); // @output guard short
                           // @output short body
}
