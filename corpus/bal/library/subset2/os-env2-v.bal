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

import ballerina/os;
import ballerina/io;

public function main() returns error? {
    // getUsername returns a non-empty name.
    io:println(os:getUsername().length() > 0); // @output true

    // listEnv snapshots the environment; a freshly set variable appears in it.
    check os:setEnv("BAL_TEST_LISTENV", "present");
    map<string> envs = os:listEnv();
    string? v = envs["BAL_TEST_LISTENV"];
    io:println(v); // @output present
    check os:unsetEnv("BAL_TEST_LISTENV");
}
