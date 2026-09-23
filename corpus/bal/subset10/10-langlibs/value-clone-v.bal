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
import ballerina/lang.value;

type Person record {|
    string name;
    int age;
|};

type Team record {|
    string name;
    Person lead;
|};

isolated function cloneNumbers(int[] ns) returns int[] => ns.clone();

public function main() {
    Person p = {name: "Alice", age: 30};
    Person q = p.clone();
    io:println(q); // @output {"name":"Alice","age":30}
    io:println(p == q); // @output true
    io:println(p === q); // @output false

    Person r = value:clone(p);
    io:println(r); // @output {"name":"Alice","age":30}

    map<int> counts = {"a": 1, "b": 2};
    map<int> countsClone = counts.clone();
    io:println(countsClone); // @output {"a":1,"b":2}

    int[] a = [1, 2, 3];
    int[] b = a.clone();
    io:println(b); // @output [1,2,3]

    [int, string] t = [1, "one"];
    [int, string] tc = t.clone();
    io:println(tc); // @output [1,"one"]

    Team team = {name: "core", lead: p};
    Team teamClone = team.clone();
    io:println(teamClone); // @output {"name":"core","lead":{"name":"Alice","age":30}}
    io:println(teamClone.lead === team.lead); // @output false

    map<json> j = {"n": 1, "xs": [1, 2]};
    map<json> jc = j.clone();
    io:println(jc); // @output {"n":1,"xs":[1,2]}

    anydata ad = {"x": [1, 2]};
    anydata c = ad.clone();
    io:println(c); // @output {"x":[1,2]}

    io:println((1).clone()); // @output 1
    io:println("hi".clone()); // @output hi
    io:println((1.5).clone()); // @output 1.5
    decimal d = 2.5;
    io:println(d.clone()); // @output 2.5
    io:println(true.clone()); // @output true
    () n = ();
    io:println(n.clone()); // @output

    error e = error("boom");
    io:println(e.clone() === e); // @output true

    q.age = 31;
    io:println(p); // @output {"name":"Alice","age":30}
    io:println(q); // @output {"name":"Alice","age":31}
    b[0] = 100;
    io:println(a); // @output [1,2,3]
    io:println(b); // @output [100,2,3]
    a[1] = 200;
    io:println(a); // @output [1,200,3]
    io:println(b); // @output [100,2,3]

    io:println(cloneNumbers([7, 8])); // @output [7,8]
}
