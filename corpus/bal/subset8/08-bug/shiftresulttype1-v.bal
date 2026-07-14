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
    int:Signed8 a = 2;
    int:Signed8 b = -120;
    int:Unsigned8 c = 1;
    int:Signed16 d = 8;
    int:Signed16 e = -32755;
    int:Unsigned16 f = 3;
    int:Signed32 g = 6;
    int:Signed32 h = -2147483647;
    int:Unsigned32 i = 5;

    int:Unsigned8 x1 = x >> a;
    io:println(x1); // @output 62
    int:Unsigned8 x2 = x >> b;
    io:println(x2); // @output 0

    int:Unsigned8 x3 = x >> c;
    io:println(x3); // @output 125

    int:Unsigned8 x4 = x >> d;
    io:println(x4); // @output 0
    int:Unsigned8 x5 = x >> e;
    io:println(x5); // @output 0

    int:Unsigned8 x6 = x >> f;
    io:println(x6); // @output 31

    int:Unsigned8 x7 = x >> g;
    io:println(x7); // @output 3
    int:Unsigned8 x8 = x >> h;
    io:println(x8); // @output 125

    int:Unsigned8 x9 = x >> i;
    io:println(x9); // @output 7
}
