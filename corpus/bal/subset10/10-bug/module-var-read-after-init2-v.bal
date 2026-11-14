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

int n = 2;
int inBranch;
int whileTrue;
int whileCond;
int foreachBody;

function init() {
    if n > 0 {
        inBranch = 1;
        io:println(inBranch); // @output 1
    } else {
        inBranch = 2;
    }

    while true {
        whileTrue = 3;
        break;
    }
    io:println(whileTrue); // @output 3

    whileCond = n;
    int i = 0;
    while i < whileCond {
        i += 1;
    }

    foreach int j in 0 ..< n {
        foreachBody = j;
        io:println(foreachBody); // @output 0
                                 // @output 1
    }
    foreachBody = 5;
}

public function main() {
    io:println(inBranch); // @output 1
    io:println(whileTrue); // @output 3
    io:println(whileCond); // @output 2
    io:println(foreachBody); // @output 5
}
