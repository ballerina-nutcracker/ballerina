// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

const int PairCount = 5000;

isolated function value(int input) returns int {
    return input;
}

public function main() returns error? {
    int checksum = 0;
    foreach int _ in 0 ..< PairCount {
        future<int> first = start value(1);
        future<int> second = start value(2);
        int winner = check wait first | second;
        int loser;
        if winner == 1 {
            loser = check wait second;
        }
        else {
            loser = check wait first;
        }
        checksum += winner + loser;
    }
    io:println(checksum); // @output 15000
}
