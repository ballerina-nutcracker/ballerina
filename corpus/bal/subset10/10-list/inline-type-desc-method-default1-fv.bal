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

type Run object {
    function run(int p) returns int;
};

class Runner {
    function run(int p = 0) returns int {
        return p + 1;
    }
}

public function main() {
    any v = new Runner();
    any fn = function (int a, int b = 2) returns int => a + b;
    [Run, int]|[Run, string] objects = [<object { function run(int p = 3) returns int; }>v, 1]; // @error inline type with a default in a candidate trial is unimplemented
    [boolean, int]|[boolean, string] functions = [fn is function (int a, int b = 2) returns int, "s"]; // @error inline type with a default in a candidate trial is unimplemented
    io:println(objects is [Run, int]);
    io:println(objects[0].run(4));
    io:println(functions);
}
