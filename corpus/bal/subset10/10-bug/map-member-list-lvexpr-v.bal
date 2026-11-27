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

type R record {|
    int[] xs?;
|};

type Q record {|
    int[][] xss?;
|};

public function main() {
    map<int[]> m = {};
    m["k"][0] = 1;
    io:println(m); // @output {"k":[1]}
    R r = {};
    r.xs[0] = 2;
    io:println(r); // @output {"xs":[2]}

    m["k"][0] += 5;
    io:println(m); // @output {"k":[6]}
    r.xs[0] *= 3;
    io:println(r); // @output {"xs":[6]}

    map<R> mr = {};
    mr["a"].xs[1] = 3;
    io:println(mr); // @output {"a":{"xs":[0,3]}}

    map<int[][]> mll = {};
    mll["k"][1][0] = 4;
    io:println(mll); // @output {"k":[[],[4]]}

    Q q = {};
    q.xss[0][1] = 7;
    io:println(q); // @output {"xss":[[0,7]]}

    map<map<int[]>> mml = {};
    mml["a"]["b"][2] = 8;
    io:println(mml); // @output {"a":{"b":[0,0,8]}}
}
