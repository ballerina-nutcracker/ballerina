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
import ballerina/lang.runtime;

public function main() {
    io:println("before"); // @output before
    runtime:sleep(0.01);
    io:println("after"); // @output after

    // Non-positive durations must be a no-op, not hang or error.
    runtime:sleep(0);
    io:println("zero-ok"); // @output zero-ok

    runtime:sleep(-1);
    io:println("negative-ok"); // @output negative-ok
}
