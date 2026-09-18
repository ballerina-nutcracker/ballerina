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

// Reads the trace the CLI wrote before the runtime was created, then fails.
// The reported size proves the recording was complete and observable during
// module initialization, and that the runtime failure did not change it.
public function main() {
    string|io:Error content = io:fileReadString("traces.json");
    if content is io:Error {
        io:println("trace unreadable");
        return;
    }
    io:println("trace bytes: ", content.length());
    panic error("runtime failure after the trace was written");
}
