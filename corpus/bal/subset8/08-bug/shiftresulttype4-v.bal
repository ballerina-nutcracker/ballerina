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
    int:Unsigned8 x = 250;
    int:Unsigned16 y = 500;
    int:Unsigned32 z = 1000000000;

    int:Unsigned8 x1 = x >> 1;
    io:println(x1); // @output 125

    int:Unsigned16 y2 = y >> 1;
    io:println(y2); // @output 250

    int:Unsigned32 z3 = z >> 1;
    io:println(z3); // @output 500000000

    int x11 = x << 1;
    io:println(x11); // @output 500

    int y22 = y << 1;
    io:println(y22); // @output 1000

    int z33 = z << 1;
    io:println(z33); // @output 2000000000
}
