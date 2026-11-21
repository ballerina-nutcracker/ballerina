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

type Leaf record {|
    int n = 1;
    string s = "leaf";
|};

type Branch record {|
    Leaf leaf = {};
    Leaf[] leaves = [{}, {n: 2}];
    map<Leaf> byName = {a: {}};
|};

type Root record {|
    Branch branch = {};
    map<Leaf>[2] pairs = [];
|};

const Root A = {};
const Root B = {};

public function main() {
    io:println(A.branch.leaf.n); // @output 1
    io:println(A.branch.leaves[1].n); // @output 2
    io:println(A.branch.byName["a"]); // @output {"n":1,"s":"leaf"}
    io:println(A.pairs[0]); // @output {}
    io:println(A.branch.leaf is readonly); // @output true
    io:println(A.branch.leaf is Leaf); // @output true
    io:println(A is readonly); // @output true
    io:println(A == B); // @output true
    io:println(A.branch.leaf == B.branch.leaf); // @output true
}
