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

function boom() {
    panic error("default panicked");
}

// The default worker panics before its send, so a's receive re-raises that
// panic, which a traps.
function run() {
    worker a returns string {
        int|error r = trap <- function;
        return r is error ? "trapped " + r.message() : "received";
    }
    pending = a;
    boom();
    1 -> a;
}

public function main() {
    error? e = trap run();
    io:println(e is error ? e.message() : "done"); // @output default panicked
    future<string>? f = pending;
    if f is future<string> {
        string|error r = wait f;
        io:println(r); // @output trapped default panicked
    }
}
