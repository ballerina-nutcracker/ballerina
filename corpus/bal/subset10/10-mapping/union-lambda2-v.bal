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

type Inner1 record {| int[] v; int m; |};

type Inner2 record {| string[] v; string m; |};

type Outer1 record {| Inner1|Inner2 inner; int tag; |};

type Outer2 record {| Inner1|Inner2 inner; string tag; |};

type Pair1 record {| [int[], int]|[string[], string] pair; int tag; |};

type Pair2 record {| [int[], int]|[string[], string] pair; string tag; |};

public function main() {
    int[] arr = [1, 2];
    Outer1|Outer2 o = {inner: {v: arr.map(x => x + 1), m: 1}, tag: 1};
    io:println(o); // @output {"inner":{"v":[2,3],"m":1},"tag":1}
    io:println(o is Outer1, " ", o.inner is Inner1); // @output true true
    Pair1|Pair2 p = {pair: [arr.map(x => x * 10), 1], tag: "t"};
    io:println(p); // @output {"pair":[[10,20],1],"tag":"t"}
    io:println(p is Pair2); // @output true
}
