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

// Exercises the native (palnative) OS closures end-to-end through the real
// `bal` binary so their coverage flows into the palnative profile. In-process
// corpus tests run under NewTestPal, which wires os.* directly and never hits
// these closures. Output is kept deterministic: the machine-specific
// getUserHome/getUsername are asserted non-empty rather than printed verbatim.
import ballerina/io;
import ballerina/os;

public function main() returns error? {
    // GetUserHome + GetUsername closures.
    io:println(os:getUserHome().length() > 0);   // @output true
    io:println(os:getUsername().length() > 0);   // @output true

    // SetEnv + GetEnv round-trip.
    check os:setEnv("BAL_CLI_PAL_VAR", "hello");
    io:println(os:getEnv("BAL_CLI_PAL_VAR"));     // @output hello

    // ListEnv snapshot contains the freshly set variable.
    map<string> envs = os:listEnv();
    io:println(envs["BAL_CLI_PAL_VAR"]);          // @output hello

    // UnsetEnv removes it; GetEnv then returns "".
    check os:unsetEnv("BAL_CLI_PAL_VAR");
    io:println(os:getEnv("BAL_CLI_PAL_VAR") == ""); // @output true
}
