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
    int a = 99999999999999999999; // @error out of range for int
    int b = 9223372036854775808; // @error out of range for int
    int c = -99999999999999999999; // @error out of range for int
    io:println(99999999999999999999); // @error out of range for int
    int x = 1;
    x += 99999999999999999999; // @error out of range for int
    int e = 0xFFFFFFFFFFFFFFFFFF; // @error out of range for int
    int f = -0x8000000000000001; // @error out of range for int
    int|float g = 99999999999999999999; // @error out of range for int
    io:println(a, b, c, x, e, f, g);
}
