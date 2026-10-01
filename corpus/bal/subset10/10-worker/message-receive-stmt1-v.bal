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

int progress = 0;

// A receive whose static type is nil can be an action statement; it still
// waits for its message.
public function main() {
    worker a {
        progress = 1;
        () -> function;
        progress = 2;
        () ->> function;
    }
    <- a;
    io:println(progress >= 1); // @output true
    <- a;
    io:println(progress); // @output 2
}
