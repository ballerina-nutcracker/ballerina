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

// The flush returns a's error. b and c never receive their messages, so the
// default worker panics when it terminates; trap keeps main running.
function flushAll() {
    worker a returns error? {
        check failWith("a failed");
        int _ = <- function;
    }
    worker b returns error? {
        check failWith("b failed");
        int _ = <- function;
    }
    worker c {
        int _ = <- function;
    }
    1 -> b;
    2 -> a;
    3 -> c;
    error? r = flush;
    io:println(r is error ? r.message() : "flushed"); // @output a failed
}

public function main() {
    error? r = trap flushAll();
    io:println(r is error ? r.message() : "terminated"); // @output b failed
}
