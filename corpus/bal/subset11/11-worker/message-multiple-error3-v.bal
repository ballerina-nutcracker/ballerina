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

function failWith(string message) returns error? {
    return error(message);
}

// a fails, so r's multiple receive takes no message: b's sync send waits for r
// to terminate and reports r's error.
function syncSend() {
    worker a returns error? {
        check failWith("a failed");
        check (1 ->> r);
    }
    worker b returns error? {
        return 2 ->> r;
    }
    worker r returns error? {
        record {| int a; int b; |} _ = check <- {a, b};
    }
    error? ra = wait a;
    error? rb = wait b;
    error? rr = wait r;
    io:println(ra, " ", rb, " ", rr); // @output error("a failed") error("a failed") error("a failed")
}

// Same, with b's message sent async: b's flush reports r's error, and the
// message is still undelivered when b terminates, so b terminates abnormally.
function asyncSendFlush() {
    worker a returns error? {
        check failWith("a failed");
        1 -> r;
        check flush r;
    }
    worker b {
        2 -> r;
        error? e = flush r;
        io:println("flush ", e is error ? e.message() : "nil"); // @output flush a failed
    }
    worker r returns error? {
        record {| int a; int b; |} _ = check <- {a, b};
    }
    error? ra = wait a;
    error? rb = trap wait b;
    error? rr = wait r;
    io:println(ra, " ", rb, " ", rr); // @output error("a failed") error("a failed") error("a failed")
}

public function main() {
    syncSend();
    asyncSendFlush();
}
