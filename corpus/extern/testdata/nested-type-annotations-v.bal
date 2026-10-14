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

type Meta record {|
    string name;
|};

annotation Meta meta on type, record field;

@meta {name: "age"}
type Age int;

@meta {name: "names"}
type Names string[];

type Inner record {|
    @meta {name: "x"}
    int x;
|};

type Node record {|
    @meta {name: "value"}
    int value;
    Node? next;
|};

type Bare record {|
    int y;
|};

type Wrapper record {|
    Inner inner;
|};

type Tree record {|
    Tree? left;
    Inner data;
|};

type Outer record {|
    Age age;
    Names names;
    Inner inner;
    Inner[] list;
    Inner? maybe;
    () | Inner nilFirst;
    readonly & Inner frozen;
    Node node;
    int plain;
    Bare bare;
    Wrapper wrapper;
    Tree tree;
|};

public function main() {
    io:println(describe(Outer)); //@output type[] age:{type[meta=age]} frozen:{type[] x[meta=x]} inner:{type[] x[meta=x]} list:{type[] x[meta=x]} maybe:{type[] x[meta=x]} names:{type[meta=names]} nilFirst:{type[] x[meta=x]} node:{type[] value[meta=value] next:<cycle>} tree:{type[] data:{type[] x[meta=x]} left:<cycle>} wrapper:{type[] inner:{type[] x[meta=x]}}
    io:println(describe(Bare)); //@output type[]
}

function describe(typedesc<anydata> td) returns string = external;
