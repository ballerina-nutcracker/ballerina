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
    [int]|[int, int]|[int, int, int] t1 = [1, 2, 3];
    io:println(t1 is [int, int, int]); // @output true
    io:println(t1 is [byte, byte, byte]); // @output false

    [int]|[int, int]|int|byte t2 = [1, 2];
    io:println(t2 is [int, int]); // @output true
    io:println(t2 is [byte, byte]); // @output false

    [int...]|int|byte t3 = [1, 2];
    io:println(t3 is [int...]); // @output true
    io:println(t3 is [int, int]); // @output false
    io:println(t3 is [byte...]); // @output false
}
