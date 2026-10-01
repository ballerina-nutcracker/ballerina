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

// A receiver that fails or panics after taking its message doesn't fail the
// sync send or flush that waited for it.
function failAfterSyncSend() {
    worker a returns error? {
        int _ = check <- function;
        return error("after");
    }
    error? sent = 1 ->> a;
    io:println(sent is error ? "failed" : "delivered"); // @output delivered
    error? r = wait a;
    io:println(r is error ? r.message() : "done"); // @output after
}

function panicAfterFlush() {
    worker a returns error? {
        int _ = check <- function;
        panic error("after");
    }
    1 -> a;
    error? flushed = flush a;
    io:println(flushed is error ? "failed" : "flushed"); // @output flushed
    error? r = trap wait a;
    io:println(r is error ? r.message() : "done"); // @output after
}

public function main() {
    failAfterSyncSend();
    panicAfterFlush();
}
