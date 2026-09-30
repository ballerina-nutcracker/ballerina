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
    int i = -1;
    byte b = 200;
    b &= i;
    io:println(b); // @output 200
    b >>= i;
    io:println(b); // @output 0

    byte c = 5;
    c |= 2;
    io:println(c); // @output 7
    c ^= 1;
    io:println(c); // @output 6

    int:Unsigned16 w = 300;
    w &= i;
    io:println(w); // @output 300

    int j = 12;
    byte k = 7;
    j &= k;
    io:println(j); // @output 4

    int:Unsigned16 u = 65535;
    u >>>= 4;
    io:println(u); // @output 4095

    byte[] bs = [255];
    bs[0] &= 15;
    io:println(bs[0]); // @output 15

    float f = 1.5;
    f *= 2;
    io:println(f); // @output 3.0
    f /= 4;
    io:println(f); // @output 0.75

    decimal d = 1.5;
    d *= 3;
    io:println(d); // @output 4.5
}
