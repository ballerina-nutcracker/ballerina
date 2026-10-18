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

type A record {| int[] v; int m; |};

type B record {| string[] v; string m; |};

type C record {| (0|1)[] v; int m; |};

type D record {| string[] v; string m; |};

public function main() {
    (int|string)[] mixed = [1, "a", 2];
    A|B r = {v: mixed.map(x => x is int ? x : 0), m: 1};
    io:println(r); // @output {"v":[1,0,2],"m":1}
    int[] arr = [1, 2, 3];
    C|D c = {v: arr.map(x => x == 1 ? x : 0), m: 1};
    io:println(c); // @output {"v":[1,0,0],"m":1}
}
