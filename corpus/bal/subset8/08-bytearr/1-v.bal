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
    byte[] b1 = [];
    b1[0] = 42;
    b1[1] = 43;
    b1[3] = 44;
    io:println(b1[0]); // @output 42
    io:println(b1[1]); // @output 43
    io:println(b1[2]); // @output 0
    io:println(b1[3]); // @output 44

    byte[] b2 = [88, 89, 90];
    io:println(b2[0]); // @output 88
    io:println(b2[1]); // @output 89
    io:println(b2[2]); // @output 90
}
