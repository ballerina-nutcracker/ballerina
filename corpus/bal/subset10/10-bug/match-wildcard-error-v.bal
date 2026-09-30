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

function side(string s) returns boolean {
    io:println("guard " + s);
    return true;
}

function wildcard(int|error v) returns string {
    match v {
        2 => {
            return "two";
        }
        _ => {
            return "wildcard";
        }
    }
    return "fell through";
}

function multiPattern(int|error v) returns string {
    match v {
        1 => {
            return "one";
        }
        2|_ => {
            return "multi";
        }
    }
    return "none";
}

function guarded(int|error v) {
    match v {
        _ if side("wildcard") => {
            io:println("guarded body");
        }
    }
}

function constGuarded(int v) {
    match v {
        2 if side("const") => {
            io:println("const body");
        }
    }
}

function narrowed(int|error v) returns int {
    match v {
        _ => {
            int i = v;
            return i;
        }
    }
    return -1;
}

function anyOrError(any|error v) {
    match v {
        _ => {
            io:println("any");
        }
    }
}

function anyMulti(any v) returns string {
    match v {
        1|_ => {
            return "any multi";
        }
    }
}

function readonlyTarget(readonly v) returns string {
    match v {
        _ => {
            return "matched";
        }
    }
    return "not matched";
}

function nilOrError(error? v) returns string {
    match v {
        _ => {
            return "matched";
        }
    }
    return "not matched";
}

public function main() {
    io:println(wildcard(1)); // @output wildcard
    io:println(wildcard(error("e"))); // @output fell through
    io:println(multiPattern(1), multiPattern(2), multiPattern(3), multiPattern(error("e"))); // @output onemultimultinone
    guarded(1); // @output guard wildcard
                // @output guarded body
    guarded(error("e"));
    constGuarded(3);
    constGuarded(2); // @output guard const
                     // @output const body
    io:println(narrowed(5)); // @output 5
    io:println(narrowed(error("e"))); // @output -1
    anyOrError("s"); // @output any
    anyOrError(error("e"));
    io:println(anyMulti(3)); // @output any multi
    io:println(readonlyTarget(1), " ", readonlyTarget(error("e"))); // @output matched not matched
    io:println(nilOrError(()), " ", nilOrError(error("e"))); // @output matched not matched
    io:println("done"); // @output done
}
