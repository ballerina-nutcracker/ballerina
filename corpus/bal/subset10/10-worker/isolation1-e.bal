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

int mutableGlobal = 0;

function notIsolated() returns int {
    return 1;
}

isolated function readsMutableGlobal() returns int {
    worker w returns int {
        return mutableGlobal; // @error access of a mutable variable from an isolated worker
    }
    return wait w;
}

isolated function callsNonIsolated() returns int {
    worker w returns int {
        return notIsolated(); // @error invocation of a non-isolated function from an isolated worker
    }
    return wait w;
}

public function main() {
    io:println(readsMutableGlobal());
    io:println(callsNonIsolated());
}
