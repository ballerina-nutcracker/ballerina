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
    int i = 1;
    io:println(<decimal>i); // @output 1
    i = 0;
    io:println(<decimal>i); // @output 0
    i = -1;
    io:println(<decimal>i); // @output -1
    i = 2147483647;
    io:println(<decimal>i); // @output 2147483647
    i = 2147483646;
    io:println(<decimal>i); // @output 2147483646
    i = 2147483648;
    io:println(<decimal>i); // @output 2147483648
    i = -2147483648;
    io:println(<decimal>i); // @output -2147483648
    i = -2147483647;
    io:println(<decimal>i); // @output -2147483647
    i = -2147483649;
    io:println(<decimal>i); // @output -2147483649
    i = 9223372036854775807;
    io:println(<decimal>i); // @output 9223372036854775807
    i = -9223372036854775807 - 1;
    io:println(<decimal>i); // @output -9223372036854775808
}
