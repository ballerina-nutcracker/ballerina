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
    float? a = 5;
    float? b = 6;
    float? c = 1;

    float? v1 = a + b;
    io:println(v1); // @output 11.0
    io:println(a + b); // @output 11.0

    float? v2 = a + b + c;
    io:println(v2); // @output 12.0
    io:println(a + b + c); // @output 12.0

    float? v3 = a - b;
    io:println(v3); // @output -1.0

    float? v4 = a + b - c;
    io:println(v4); // @output 10.0

    io:println(b / c); // @output 6.0

    float? v5 = b / 3;
    io:println(v5); // @output 2.0

    float? v6 = -a;
    io:println(v6); // @output -5.0
    io:println(-c); // @output -1.0

    float d = 13;
    float? v7 = a + d;
    io:println(v7); // @output 18.0
    io:println(a + b + c + d); // @output 25.0

    float? e = ();
    float? v8 = a + e;
    io:println(v8); // @output 
    io:println(a + b + c + d + e); // @output 

    io:println(-a); //@output -5.0
}
