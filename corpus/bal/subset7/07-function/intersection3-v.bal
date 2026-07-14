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
type F function(int...) returns byte;
type F1 function(int, int) returns int;

type Fx F&F1;

public function main() {
    Fx f = foo;
    byte rest = f(1, 2, 3, 4);
    io:println(rest); // @output 0
}

function foo(int... args) returns byte {
    int[] _ = args;
    return 0;
}
