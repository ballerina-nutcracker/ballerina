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

future<string>? pending = ();

// The default worker returns an error before receiving, so a's bare flush
// returns that error; the message is still undelivered when a terminates, so
// a terminates abnormally with the default worker's error.
function run(boolean early) returns error? {
    worker a returns string {
        1 -> function;
        error? e = flush;
        io:println(e is error ? e.message() : "flushed"); // @output default failed
        return "done";
    }
    pending = a;
    if early {
        return error("default failed");
    } else {
    }
    int _ = <- a;
}

public function main() {
    error? e = run(true);
    future<string>? f = pending;
    if f is future<string> {
        string|error r = trap wait f;
        io:println(r is error ? "a panicked: " + r.message() : r); // @output a panicked: default failed
    }
    io:println(e is error ? e.message() : "done"); // @output default failed
}
