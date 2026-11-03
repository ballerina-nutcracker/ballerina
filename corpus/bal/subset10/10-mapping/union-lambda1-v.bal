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

type C record {| int[] v; int m; |};

type D record {| boolean[] v; int m; |};

public function main() {
    int[] arr = [1, 2];
    A|B r = {v: arr.map(x => x + 1), m: 1};
    io:println(r); // @output {"v":[2,3],"m":1}
    io:println(r is A); // @output true
    C|D r2 = {v: arr.map(x => x > 1), m: 1};
    io:println(r2); // @output {"v":[false,true],"m":1}
    io:println(r2 is D); // @output true
}
