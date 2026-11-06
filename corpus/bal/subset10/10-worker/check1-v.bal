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

function parse(boolean ok) returns int|error {
    if ok {
        return 1;
    }
    return error("parse failed");
}

// A check expression in a worker body returns the error from the worker, not
// from the enclosing function, so it is allowed even though main returns nil.
public function main() {
    worker w returns int|error {
        int value = check parse(false);
        io:println("not reached");
        return value;
    }
    int|error result = wait w;
    io:println(result); // @output error("parse failed")
}
