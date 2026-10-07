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

const S = "xyz";

type AB "ab"|"cd";

function hello() returns string {
    return "hello";
}

public function main() {
    string:Char ch = "q";
    io:println(ch[0]); // @output q
    "abc" lit = "abc";
    io:println(lit[1]); // @output b
    AB u = "cd";
    io:println(u[1]); // @output d
    io:println(S[2]); // @output z
    io:println("literal"[0]); // @output l
    io:println(hello()[4]); // @output o
    string[] arr = ["hey"];
    io:println(arr[0][2]); // @output y
    byte b = 1;
    io:println("abc"[b]); // @output b
    var v = "abc"[0];
    string:Char z = v;
    string:Char[] cs = [z, "abc"[2]];
    io:println(cs); // @output ["a","c"]
    string acc = "";
    foreach int i in 0 ..< 3 {
        acc += "abc"[2 - i];
    }
    io:println(acc); // @output cba
}
