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

int progress = 0;

future<string>? pending = ();

// The sync send to the default worker completes once the default worker has
// executed its receive, so the default worker's statement before it has run.
function delivered() {
    worker a returns string {
        error? sent = 1 ->> function;
        return sent is () ? "taken after " + progress.toString() : "failed";
    }
    progress = 1;
    int _ = <- a;
    string r = wait a;
    io:println(r); // @output taken after 1
}

// The default worker returns an error before its receive, so a's sync send
// completes with that error.
function failed(boolean early) returns error? {
    worker a returns string {
        error? sent = 1 ->> function;
        return sent is error ? sent.message() : "taken";
    }
    pending = a;
    if early {
        return error("default failed");
    } else {
    }
    int _ = <- a;
}

public function main() {
    delivered();
    error? e = failed(true);
    io:println(e is error ? e.message() : "done"); // @output default failed
    future<string>? f = pending;
    if f is future<string> {
        string|error r = wait f;
        io:println(r); // @output default failed
    }
}
