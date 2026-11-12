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

int branches;
int loop;

function init() {
    int i = 0;
    if i == 0 {
        branches = 1;
    } else {
        branches = 2;
    }
    io:println(branches); // @output 1
    branches = branches + 1;

    loop = 1;
    while i < 2 {
        loop = loop + i;
        i += 1;
    }
}

public function main() {
    io:println(branches); // @output 2
    io:println(loop); // @output 2
}
