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
type F1 function(int) returns 1|2|3;
type F2 function(string) returns 2|3|4;

type Fx F1&F2;

public function main() {
    Fx fx = foo;
    1|2|3 r1 = fx(1);
    io:println(r1); // @output 2
    fx = bar;
    r1 = fx(1);
    io:println(fx(r1)); // @output 2
    2|3|4 r2 = fx("aa");
    io:println(r2); // @output 3
}

function foo(int|string a) returns 2 {
    int|string _ = a;
    return 2;
}

function bar(int|string a) returns 2|3 {
    if a is int {
        return 2;
    }
    return 3;
}
