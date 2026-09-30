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

type T record {|
    int v;
|};

type IntOrString int|string;

type Node record {|
    int v;
    Node? next;
|};

type Five 5;

public function main() {
    typedesc<int> ti = int;
    typedesc<T> tt = T;
    typedesc<anydata> ta = int;
    typedesc<anydata> tu = IntOrString;

    io:println(ti is typedesc<int>); // @output true
    io:println(ti is typedesc<anydata>); // @output true
    io:println(ti is typedesc<int|string>); // @output true
    io:println(ti is typedesc); // @output true
    io:println(ti is typedesc<string>); // @output false
    io:println(ta is typedesc<int>); // @output true
    io:println(ta is typedesc<string>); // @output false
    io:println(tt is typedesc<T>); // @output true
    io:println(tt is typedesc<record {| int v; |}>); // @output true
    io:println(tt is typedesc<record {| string v; |}>); // @output false
    io:println(tu is typedesc<int>); // @output false
    io:println(tu is typedesc<IntOrString>); // @output true

    typedesc<Node> tn = Node;
    io:println(tn is typedesc<Node>); // @output true
    io:println(tn is typedesc<int>); // @output false

    typedesc<int> tf = Five;
    io:println(tf is typedesc<5>); // @output true
    io:println(tf is typedesc<6>); // @output false

    any a = ti;
    io:println(a is typedesc<int>); // @output true
    io:println(a is readonly); // @output true

    typedesc<int> narrowed = <typedesc<int>>ta;
    io:println(narrowed is typedesc<int>); // @output true
}
