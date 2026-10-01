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

// c fails, so b's multiple receive fails as a whole; a's sync send still sees
// its message taken.
public function main() {
    worker a returns error? {
        error? sent = 1 ->> b;
        io:println(sent is error ? "failed" : "taken"); // @output taken
    }
    worker b returns error? {
        record {int a; int c;} _ = check <- {a, c};
    }
    worker c returns error? {
        check failWith("c failed");
        2 -> b;
        check flush b;
    }
    error? ra = wait a;
    error? rb = wait b;
    io:println(ra is error ? ra.message() : "a done"); // @output a done
    io:println(rb is error ? rb.message() : "b done"); // @output c failed
}
