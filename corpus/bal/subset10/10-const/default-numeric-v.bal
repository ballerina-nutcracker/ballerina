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

type Numeric record {|
    decimal d = 1;
    float f = 2;
    byte b = 3;
    float[] fs = [1, 2];
    int|string u = "s";
    "a"|"b" lit = "b";
    readonly int ro = 6;
    map<decimal> md = {k: 4};
|};

const Numeric A = {};

public function main() {
    io:println(A.d is decimal); // @output true
    io:println(A.f is float); // @output true
    io:println(A.b is byte); // @output true
    io:println(A.fs[0] is float); // @output true
    io:println(A.md["k"] is decimal); // @output true
    io:println(A); // @output {"d":1,"f":2.0,"b":3,"fs":[1.0,2.0],"u":"s","lit":"b","ro":6,"md":{"k":4}}
    Numeric r = {};
    io:println(r == A); // @output true
}
