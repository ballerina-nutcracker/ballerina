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

// @productions match-stmt match-guard wildcard-match-pattern const-pattern is-expr logical-expr method-call-expr function-call-expr
import ballerina/io;

public function main() {
    multiBlock(5);            // @output 6
    multiPattern(2);          // @output two
    multiPattern(3);          // @output three
    multiPattern(9);          // @output other
    nonVarRef();              // @output four
    otherVar(1, 3);           // @output 4
    otherVar(1, "x");         // @output other
    guardFalse(1);            // @output other
    orGuard("a");             // @output a
    seq(1);                   // @output g1
                              // @output g3
                              // @output c3
    seq(2);                   // @output g2
                              // @output c2
}

function multiBlock(int|string v) {
    match v {
        _ if v is int && v > 0 => {
            io:println(v + 1);
        }
    }
}

function multiPattern(int|string v) {
    match v {
        1|2|"a" if v is int && v * 2 == 4 => {
            io:println("two");
        }
        3 if v + 1 == 4 => {
            io:println("three");
        }
        _ => {
            io:println("other");
        }
    }
}

function val() returns int|string {
    return 4;
}

function nonVarRef() {
    match val() {
        4 if true => {
            io:println("four");
        }
        _ => {
            io:println("other");
        }
    }
}

function otherVar(int v, int|string w) {
    match v {
        1 if w is int => {
            io:println(w + v);
        }
        _ => {
            io:println("other");
        }
    }
}

function guardFalse(int v) {
    match v {
        1 if v > 5 => {
            io:println("big");
        }
        _ => {
            io:println("other");
        }
    }
}

function orGuard(int|string v) {
    match v {
        "a" if v.length() == 0 || v == "a" => {
            io:println("a");
        }
        _ => {
            io:println("other");
        }
    }
}

function seq(int v) {
    match v {
        1 if trace("g1", false) => {
            io:println("c1");
        }
        2 if trace("g2", true) => {
            io:println("c2");
        }
        _ if trace("g3", true) => {
            io:println("c3");
        }
    }
}

function trace(string s, boolean b) returns boolean {
    io:println(s);
    return b;
}
