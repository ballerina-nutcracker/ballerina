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

public function main() {
    error? t = trap test:assertTrue(false);
    io:println("assertTrue: ", t is error ? t.message() : "no error");
    // @output assertTrue: Assertion Failed!

    error? f = trap test:assertFalse(true);
    io:println("assertFalse: ", f is error ? f.message() : "no error");
    // @output assertFalse: Assertion Failed!

    error? x = trap test:assertFail();
    io:println("assertFail: ", x is error ? x.message() : "no error");
    // @output assertFail: Test Failed!
}
