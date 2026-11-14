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

type Node record {|
    int value;
    Node? next;
|};

type TaggedNode record {|
    *Node;
    string tag;
|};

type Tree record {|
    Tree[] children;
|};

type NamedTree record {|
    *Tree;
    string name;
|};

type DefaultTree record {|
    DefaultTree? left = ();
    int n = 5;
|};

type Ext record {|
    *DefaultTree;
    string s;
|};

type Opts record {|
    int v = 7;
    Alias? next = ();
|};

type Alias Opts;

type ChainedOpts record {|
    int w = 8;
    OuterAlias? next = ();
|};

type OuterAlias InnerAlias;

type InnerAlias ChainedOpts;

function optsValue(*Alias opts) returns int => opts.v;

function chainedValue(*OuterAlias opts) returns int => opts.w;

public function main() {
    TaggedNode n = {value: 1, next: {value: 2, next: ()}, tag: "t"};
    io:println(n.next); // @output {"value":2,"next":null}
    NamedTree t = {children: [{children: []}], name: "root"};
    io:println(t.children.length()); // @output 1
    Ext e = {s: "a"};
    io:println(e); // @output {"s":"a","left":null,"n":5}
    io:println(optsValue(v = 3)); // @output 3
    io:println(optsValue()); // @output 7
    io:println(chainedValue(w = 4)); // @output 4
    io:println(chainedValue()); // @output 8
}
