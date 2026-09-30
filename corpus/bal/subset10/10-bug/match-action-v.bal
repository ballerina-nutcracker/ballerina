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

client class Client {
    remote function value() returns int {
        return 7;
    }

    remote function values() returns int[] {
        return [1, 2];
    }

    remote function checked() returns int|error {
        return 8;
    }

    remote function failing() returns int|error {
        return error("boom");
    }

    resource function get num() returns int {
        return 9;
    }

    resource function get items() returns int[] {
        return [5, 6];
    }
}

function one() returns int {
    return 1;
}

function list() returns int[] {
    return [3, 4];
}

function failing() returns int|error {
    return error("boom");
}

public function main() returns error? {
    future<int> f = start one();
    match wait f {
        1 => {
            io:println("wait one"); // @output wait one
        }
        _ => {
            io:println("wait other");
        }
    }

    match start one() {
        _ => {
            io:println("start"); // @output start
        }
    }

    Client c = new;
    match c->value() {
        7 => {
            io:println("remote seven"); // @output remote seven
        }
        _ => {
            io:println("remote other");
        }
    }

    match check c->checked() {
        8 => {
            io:println("check eight"); // @output check eight
        }
    }

    match trap c->value() {
        7 => {
            io:println("trap seven"); // @output trap seven
        }
    }

    match c->/num {
        9 => {
            io:println("resource nine"); // @output resource nine
        }
    }

    future<int|error> failed = start failing();
    match wait failed {
        _ => {
            io:println("wait wildcard");
        }
    }

    match c->failing() {
        _ => {
            io:println("remote wildcard");
        }
    }

    foreach int x in c->values() {
        io:println(x); // @output 1
                       // @output 2
    }

    foreach int x in c->/items {
        io:println(x); // @output 5
                       // @output 6
    }

    future<int[]> g = start list();
    foreach int x in checkpanic wait g {
        io:println(x); // @output 3
                       // @output 4
    }
    io:println("done"); // @output done
}
