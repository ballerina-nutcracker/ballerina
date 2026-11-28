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

// A flush has a's failure type only right after an async send to a; after an
// intervening sync send or flush it is nil, so it can be a bare statement.
function afterSyncSend() returns error? {
    worker a returns error? {
        int _ = check <- function;
        int _ = check <- function;
    }
    1 -> a;
    check (2 ->> a);
    flush a;
    flush;
    return wait a;
}

function afterFlush() returns error? {
    worker a returns error? {
        int _ = check <- function;
    }
    1 -> a;
    check flush a;
    flush a;
    flush;
    return wait a;
}

public function main() returns error? {
    check afterSyncSend();
    check afterFlush();
    io:println("flushed"); // @output flushed
}
