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
    int x;
|};

type B record {|
    string x;
|};

type C record {|
    [int]|[string] y;
|};

type D record {|
    int y;
|};

(A|B)[] g = [{x: 1}, {x: "s"}];

C|D h = {y: ["s"]};

public function main() {
    io:println(g[0] is A); // @output true
    io:println(g[1] is B); // @output true
    io:println(h is C); // @output true
    C c = <C>h;
    io:println(c.y is [string]); // @output true
}
