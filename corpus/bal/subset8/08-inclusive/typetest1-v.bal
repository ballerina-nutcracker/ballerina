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

type R1 record {
    int n;
    float x;
};

type R2 record {
    int? n;
    float? x;
};

type R3 record {
    int n;
    float x;
    float y;
};

public function main() {
    R1 r1 = {n: 1, x: 1.5};
    map<any?> m = r1;
    io:println(m is R1); // @output true
    io:println(m is R2); // @output true
    io:println(m is R3); // @output false
    R2 r2 = {n: 1, x: 1.5};
    m = r2;
    io:println(m is R1); // @output false
    io:println(m is R2); // @output true
    m = {n: 1, x: 1.5};
    io:println(m is R1); // @output false
    io:println(m is R2); // @output false
    any v = r1;
    io:println(v is map<any>); // @output true
    io:println(v is map<any?>); // @output true
    v = r2;
    io:println(v is map<any>); // @output true
    io:println(v is map<any?>); // @output true
}
