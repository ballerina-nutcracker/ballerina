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

// a keeps a separate queue for each receiver, so each one gets the message
// sent to it regardless of the order of the sends.
public function main() {
    worker a {
        1 -> b;
        2 -> function;
    }
    worker b returns int {
        return <- a;
    }
    int fromA = <- a;
    int inB = wait b;
    io:println(fromA, " ", inB); // @output 2 1
}
