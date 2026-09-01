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

function helper() {
    io:println("helper-done"); // @output helper-done
}

function sleeper() {
    runtime:sleep(0.05);
    io:println("sleeper-done"); // @output sleeper-done
}

public function main() {
    // helper is started after sleeper but must still complete first: a
    // sleeping strand must yield its cooperative thread, not hold it for the
    // whole sleep duration.
    future<()> f1 = start sleeper();
    future<()> f2 = start helper();
    ()|error r1 = wait f1;
    ()|error r2 = wait f2;
    io:println(r1 is error); // @output false
    io:println(r2 is error); // @output false
}
