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

// Cloning an immutable value gives back the same value, so the received list
// is identical to the one that was sent.
public function main() {
    final int[] & readonly values = [1, 2];
    worker a returns boolean {
        int[] & readonly received = <- function;
        return received === values;
    }
    values -> a;
    boolean same = wait a;
    io:println(same); // @output true
}
