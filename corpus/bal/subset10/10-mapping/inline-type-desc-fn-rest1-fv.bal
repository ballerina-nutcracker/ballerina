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

type A record {|
    boolean|function (int a, int b = 2, int... rest) returns int x;
|};

type B record {|
    string x;
|};

function total(int a, int b = 2, int... rest) returns int {
    int sum = a + b;
    foreach int r in rest {
        sum += r;
    }
    return sum;
}

public function main() {
    any fn = total;
    A|B tested = {x: fn is function (int a, int b = 2, int... rest) returns int}; // @error inline type with a default in a candidate trial is unimplemented
    A|B cast = {x: <function (int a, int b = 2, int... rest) returns int>fn}; // @error inline type with a default in a candidate trial is unimplemented
    io:println(tested);
    if cast is A {
        boolean|function (int a, int b = 2, int... rest) returns int x = cast.x;
        if x is function (int a, int b = 2, int... rest) returns int {
            io:println(x(1, 2, 3));
        }
    }
}
