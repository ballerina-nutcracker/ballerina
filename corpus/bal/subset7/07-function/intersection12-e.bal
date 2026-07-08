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
type S1 1|2|3|4|5|7|8|13|11|12|14;
type S2 4|7|13|8|12|14|10|9|15|16;
type S3 6|5|3|4|7|13|8|9|10;

type T1 1|2|3|4|5|7|8|13|11|12|14;
type T2 4|7|13|8|12|14|10|9|15|16;
type T3 6|5|3|4|7|13|8|9|10;

type F1 function(S1, S1) returns T1;
type F2 function(S2, S2) returns T2;
type F3 function(S3, S3) returns T3;

type F F1 & F2 & F3;

type A 11|12|15|13;

public function main() {
    F f = foo;
    A a = 11;
    T1|T2 v1 = f(a, a); // @error
    io:println(v1);
}

function foo(S1|S2|S3 s1, S1|S2|S3 s2) returns T1&T2&T3 {
        S1|S2|S3 _ = s1;
        S1|S2|S3 _ = s2;
    return 13;
    }
