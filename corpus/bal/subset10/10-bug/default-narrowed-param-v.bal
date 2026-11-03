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

function f(int|string a, int b = a is int ? a : 0) returns int {
    return b + 1;
}

function g(int|string a, function () returns int fn = () => a is int ? a : -1) returns int {
    return fn();
}

function chain(int|string x, int y = x is int ? x : 0, int z = x is int ? y + 1 : -1) returns int[] {
    return [y, z];
}

class C {
    function m(int|string a, int b = a is int ? a : 0) returns int {
        return b + 1;
    }
}

public function main() {
    io:println(f(3)); // @output 4
    io:println(f("x")); // @output 1
    io:println(f(3, 10)); // @output 11
    io:println(g(5)); // @output 5
    io:println(g("s")); // @output -1
    io:println(chain(3)); // @output [3,4]
    io:println(chain("s")); // @output [0,-1]
    C c = new;
    io:println(c.m(7)); // @output 8
    io:println(c.m("y")); // @output 1
}
