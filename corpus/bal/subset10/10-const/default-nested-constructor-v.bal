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

isolated function next() returns int {
    return 42;
}

type Inner record {|
    int x = next();
    int y = 1;
|};

// A mapping-constructor default that supplies every non-constant field of its
// inherent type is itself constant; one that omits such a field is not.
type Supplied record {|
    Inner inner = {x: 5};
    int k = 0;
|};

type Omitted record {|
    Inner inner = {y: 2};
    int k = 0;
|};

type Node record {|
    int value = 1;
    Node? next = ();
|};

type Holder record {|
    Node node = {};
    Node other = {value: 3};
|};

const Supplied A = {};
const Omitted B = {inner: {x: 7}};
const Holder C = {};

public function main() {
    io:println(A); // @output {"inner":{"x":5,"y":1},"k":0}
    io:println(A is readonly); // @output true
    io:println(B); // @output {"inner":{"x":7,"y":1},"k":0}
    Omitted o = {};
    io:println(o); // @output {"inner":{"y":2,"x":42},"k":0}
    io:println(C); // @output {"node":{"value":1,"next":null},"other":{"value":3,"next":null}}
}
