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

// An alternate wait completes with the first non-error result, skipping a
// worker that terminated normally with an error, whichever completes first.
public function main() {
    worker failed returns int|error {
        return error("first failed");
    }
    worker succeeded returns int|error {
        return 5;
    }
    int|error r = wait failed | succeeded;
    io:println(r); // @output 5
}
