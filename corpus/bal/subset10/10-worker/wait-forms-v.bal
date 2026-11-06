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

public function main() {
    worker w1 returns int {
        return 1;
    }
    worker w2 returns string {
        return "two";
    }
    worker w3 returns int {
        return 3;
    }
    worker w4 returns int {
        return 3;
    }

    record {|
        int w1;
        string w2;
    |} multiple = wait {w1, w2};
    io:println(multiple.w1); // @output 1
    io:println(multiple.w2); // @output two

    // Both alternates produce the same value, so the result does not depend on
    // which worker the scheduler completes first.
    int alternate = wait w3 | w4;
    io:println(alternate); // @output 3
}
