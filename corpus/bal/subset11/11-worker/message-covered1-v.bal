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

// An async send to a worker that can fail is reported by a later sync send or
// flush to it.
function bySyncSend() returns int|error {
    worker a returns int|error {
        int x = check <- function;
        int y = check <- function;
        return x + y;
    }
    1 -> a;
    check (2 ->> a);
    return wait a;
}

function byFlushPeer() returns int|error {
    worker a returns int|error {
        return check <- function;
    }
    3 -> a;
    check flush a;
    return wait a;
}

function byFlush() returns int|error {
    worker a returns int|error {
        return check <- function;
    }
    4 -> a;
    check flush;
    return wait a;
}

public function main() returns error? {
    io:println(check bySyncSend()); // @output 3
    io:println(check byFlushPeer()); // @output 3
    io:println(check byFlush()); // @output 4
}
