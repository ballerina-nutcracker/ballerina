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

// @productions match-stmt match-guard wildcard-match-pattern const-pattern is-expr method-call-expr function-call-expr
import ballerina/io;

public function main() {
    wildcardGuard(1);         // @output one
    wildcardGuardError(error("e")); // @output no match
    wildcardGuardError(1);    // @output one
    constGuard("a");          // @output a
    constGuard(5);            // @output other
    intGuard(1);              // @output one
    intGuard("x");            // @output other
    typeTestGuard(1);         // @output 2
    sideEffectGuard(1);       // @output guard evaluated
                              // @output one
    sideEffectGuard(2);       // @output other
}

function wildcardGuard(int|error v) {
    match v {
        _ if v + 1 == 2 => {
            io:println("one");
        }
    }
}

function wildcardGuardError(int|error v) {
    match v {
        _ if v + 1 == 2 => {
            io:println("one");
            return;
        }
    }
    io:println("no match");
}

function constGuard(int|string v) {
    match v {
        "a" if v.length() == 1 => {
            io:println("a");
        }
        _ => {
            io:println("other");
        }
    }
}

function intGuard(int|string v) {
    match v {
        1 if v + 1 == 2 => {
            io:println("one");
        }
        _ => {
            io:println("other");
        }
    }
}

function typeTestGuard(int|string v) {
    match v {
        _ if v is int => {
            io:println(v + 1);
        }
    }
}

function sideEffectGuard(int v) {
    match v {
        1 if guard() => {
            io:println("one");
        }
        _ => {
            io:println("other");
        }
    }
}

function guard() returns boolean {
    io:println("guard evaluated");
    return true;
}
