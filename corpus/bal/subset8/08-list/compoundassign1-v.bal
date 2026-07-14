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
    int[] x = [10];
    x[0] += 3;
    io:println(x[0]); // @output 13
    x[0] -= 2;
    io:println(x[0]); // @output 11
    x[0] *= 4;
    io:println(x[0]); // @output 44
    x[0] /= 2;
    io:println(x[0]); // @output 22
    x[0] &= 5;
    io:println(x[0]); // @output 4
    x[0] |= 9;
    io:println(x[0]); // @output 13
    x[0] ^= 2;
    io:println(x[0]); // @output 15
    x[0] <<= 7;
    io:println(x[0]); // @output 1920
    x[0] >>= 6;
    io:println(x[0]); // @output 30
    x[0] >>>= 2;
    io:println(x[0]); // @output 7
}
