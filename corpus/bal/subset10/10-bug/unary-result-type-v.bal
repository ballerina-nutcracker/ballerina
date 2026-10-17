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

const byte B = 5;

public function main() {
    byte b = 5;
    var r = -b;
    r = 1000;
    io:println(r); // @output 1000
    var c = ~b;
    c = -9000;
    io:println(c); // @output -9000
    int:Signed8 s = -128;
    int:Signed8 p = +s;
    io:println(p); // @output -128
    int n = -s;
    io:println(n); // @output 128
    byte? nb = 7;
    int? m = -nb;
    io:println(m); // @output -7
    int? q = ~nb;
    io:println(q); // @output -8
    1.0|2.0 f = 1.0;
    var g = -f;
    g = 5.0;
    io:println(g); // @output 5.0
    byte z = ~(-1);
    io:println(z); // @output 0
    -5 k = -B;
    io:println(k); // @output -5
    byte|float bf = 5;
    var w = -bf;
    w = 1000;
    io:println(w); // @output 1000
    w = 2.5;
    io:println(w); // @output 2.5
}
