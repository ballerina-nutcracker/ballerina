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

type R record {| R? next = {next: ()}; |};
type A record {| int x; int y = 2; |};
type B record {| string x; |};

function choose(A|B value) returns A|B {
    return value;
}

final var result = choose({x:
    (from int n in [1]
     group by record {| R r = {}; int x = 1; |} row = {r: {next: ()}, x: n}
     select row.x)[0]});

public function main() {
    io:println(result.x); // @output 1
    io:println(result is A); // @output true
    if result is A {
        io:println(result.y); // @output 2
    }
}
