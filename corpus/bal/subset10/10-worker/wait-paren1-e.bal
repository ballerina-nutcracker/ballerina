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

// A parenthesized wait operand is paired like the bare worker reference.
function single() {
    worker a returns int {
        int x = <- function;
        return x;
    }
    int|error r = wait ((a));
    5 -> a; // @error worker interaction after wait action
    io:println(r);
}

function multiple() {
    worker a returns int {
        int x = <- function;
        return x;
    }
    map<int|error> r = wait {x: (a)};
    5 -> a; // @error worker interaction after wait action
    io:println(r);
}

function alternate() {
    worker a returns int {
        int x = <- function; // @error worker message deadlock
        return x;
    }
    worker b returns int {
        int x = <- function;
        return x;
    }
    int|error r = wait (a) | (b);
    5 -> a;
    6 -> b;
    io:println(r);
}

public function main() {
    single();
    multiple();
    alternate();
}
