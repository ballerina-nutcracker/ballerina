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
int afterWhile;
int afterForeach;
int firstWhile;
int firstForeach;

function init() {
    int i = 0;
    while i < n {
        afterWhile = i;
        i += 1;
    }
    io:println(afterWhile); // @error
    afterWhile = 0;

    foreach int j in 0 ..< n {
        afterForeach = j;
    }
    io:println(afterForeach); // @error
    afterForeach = 0;

    i = 0;
    while i < n {
        io:println(firstWhile); // @error
        firstWhile = i;
        i += 1;
    }
    firstWhile = 0;

    foreach int j in 0 ..< n {
        io:println(firstForeach); // @error
        firstForeach = j;
    }
    firstForeach = 0;
}

public function main() {
    io:println(afterWhile, afterForeach, firstWhile, firstForeach);
}
