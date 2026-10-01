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

function flushPeer() {
    worker a returns int {
        int x = <- function;
        int y = <- function;
        return x + y;
    }
    1 -> a;
    2 -> a;
    flush a;
    int r = wait a;
    io:println(r); // @output 3
}

function flushAll() {
    worker a returns int {
        return <- function;
    }
    worker b returns int {
        return <- function;
    }
    1 -> a;
    2 -> b;
    flush;
    map<int> r = wait {a, b};
    io:println(r); // @output {"a":1,"b":2}
}

function flushNothing() {
    worker a {
    }
    flush a;
    flush;
    wait a;
    io:println("nothing to flush"); // @output nothing to flush
}

public function main() {
    flushPeer();
    flushAll();
    flushNothing();
}
