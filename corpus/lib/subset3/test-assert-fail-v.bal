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
import ballerina/test;

// Under `bal run` no host installs pal.Testing.Fail, so assertFail degrades to
// a plain panic carrying the message.
public function main() {
    error? trapped = trap test:assertFail("boom");
    io:println(trapped is error); // @output true
    if trapped is error {
        io:println(trapped.message()); // @output boom
    }

    error? defaulted = trap test:assertFail();
    if defaulted is error {
        io:println(defaulted.message()); // @output Test Failed!
    }
}
