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

type F1 function(1|2) returns "a"|"b";
type F2 function(2|3) returns "b"|"c";

type Fx F1&F2;

function foo(1|2|3 a) returns "a"|"b" {
    1|2|3 _ = a;
    return "b";
}

public function main() {
    F1 f = foo;
    Fx fx = <Fx>f; // @panic bad type cast
    io:println(fx);
}
