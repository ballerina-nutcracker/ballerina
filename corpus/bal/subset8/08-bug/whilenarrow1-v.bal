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

type IntList record {
    int i;
    IntList? next;
};

public function main() {
    IntList l = {i: 42, next: {i: 20, next: ()}};
    io:println(sum(l)); // @output 62
}

function sum(IntList? intList) returns int {
    IntList? temp = intList;
    int total = 0;
    while true {
        if temp is () {
            break;
        }
        else {
            total += temp.i;
            temp = temp.next;
        }
    }
    return total;
}
