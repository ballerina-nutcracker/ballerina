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

import ballerina/data.jsondata;
import ballerina/io;

type Address record {|
    string city;
    int zip;
|};

type Employee record {|
    string name;
    Address address;
    int[] scores;
    decimal salary;
    boolean active;
    string? manager;
|};

type Open record {
    string id;
};

type Strings record {|
    string id;
    string...;
|};

type Floats record {|
    float...;
|};

type Nested record {|
    record {|
        int x;
    |} a;
    int b;
|};

type ByteField record {|
    record {|
        int x;
    |} a;
    byte b;
|};

type OptionalAge record {|
    string name;
    int age?;
|};

type NilableFields record {|
    string name;
    string? alias;
|};

type IntMapField record {|
    map<int> m;
|};

type Pair [int, string];

type PairWithRest [int, string...];

type ReadonlyAddress readonly & Address;

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    json employee = {
        name: "Ann",
        address: {city: "Colombo", zip: 100},
        scores: [1, 2, 3],
        salary: 1500.50,
        active: true,
        manager: null
    };
    Employee|error e1 = jsondata:parseAsType(employee);
    show(e1);
    // @output {"name":"Ann","address":{"city":"Colombo","zip":100},"scores":[1,2,3],"salary":1500.5,"active":true,"manager":null}

    Open|error o1 = jsondata:parseAsType({id: "a", extra: [1, {k: "v"}], flag: false});
    show(o1);
    // @output {"id":"a","extra":[1,{"k":"v"}],"flag":false}

    Strings|error s1 = jsondata:parseAsType({id: "a", b: "x", c: "y"});
    show(s1);
    // @output {"id":"a","b":"x","c":"y"}
    Strings|error s2 = jsondata:parseAsType({id: "a", b: 1});
    show(s2);
    // @output error: incompatible expected type 'string' for value '1'

    Floats|error f1 = jsondata:parseAsType({a: 1, b: 2.5});
    show(f1);
    // @output {"a":1.0,"b":2.5}

    map<int>|error m1 = jsondata:parseAsType({a: 1, b: 2});
    show(m1);
    // @output {"a":1,"b":2}
    map<int>|error m2 = jsondata:parseAsType({a: "x"});
    show(m2);
    // @output error: incompatible expected type 'int' for value 'x'
    map<Address>|error m3 = jsondata:parseAsType({home: {city: "Kandy", zip: 200}});
    show(m3);
    // @output {"home":{"city":"Kandy","zip":200}}

    int[]|error a1 = jsondata:parseAsType([1, 2, 3]);
    show(a1);
    // @output [1,2,3]
    int[2]|error a2 = jsondata:parseAsType([1, 2, 3]);
    show(a2);
    // @output [1,2]
    int[2]|error a3 = jsondata:parseAsType([1, 2, 3], {allowDataProjection: false});
    show(a3);
    // @output error: array size is not compatible with the expected size
    int[4]|error a4 = jsondata:parseAsType([1, 2]);
    show(a4);
    // @output [1,2,0,0]
    int[][]|error a5 = jsondata:parseAsType([[1], [2, 3]]);
    show(a5);
    // @output [[1],[2,3]]
    int[]|error a6 = jsondata:parseAsType([1, "x"]);
    show(a6);
    // @output error: incompatible expected type 'int' for value 'x'

    Pair|error p1 = jsondata:parseAsType([1, "a"]);
    show(p1);
    // @output [1,"a"]
    Pair|error p2 = jsondata:parseAsType([1, "a", true]);
    show(p2);
    // @output [1,"a"]
    Pair|error p3 = jsondata:parseAsType([1, "a", true], {allowDataProjection: false});
    show(p3);
    // @output error: array size is not compatible with the expected size
    Pair|error p4 = jsondata:parseAsType([7]);
    show(p4);
    // @output [7,""]
    PairWithRest|error p5 = jsondata:parseAsType([1, "a", "b", "c"]);
    show(p5);
    // @output [1,"a","b","c"]

    Nested|error n1 = jsondata:parseAsType({a: {x: 1}, b: "s"});
    show(n1);
    // @output error: incompatible value 's' for type 'int' in field 'a.b'
    ByteField|error n2 = jsondata:parseAsType({a: {x: 1}, b: 300});
    show(n2);
    // @output error: incompatible value '300' for type 'byte' in field 'a.b'
    Nested|error n3 = jsondata:parseAsType({a: {x: "y"}, b: 1});
    show(n3);
    // @output error: incompatible value 'y' for type 'int' in field 'a.x'

    Address|error u1 = jsondata:parseAsType({city: "Galle", zip: 300, extra: 1});
    show(u1);
    // @output {"city":"Galle","zip":300}
    Address|error u2 = jsondata:parseAsType({city: "Galle", zip: 300, extra: 1}, {allowDataProjection: false});
    show(u2);
    // @output error: undefined field 'extra'
    Address|error u3 = jsondata:parseAsType({city: "Galle"});
    show(u3);
    // @output error: required field 'zip' not present in JSON

    OptionalAge|error q1 = jsondata:parseAsType({name: "a", age: ()}, {allowDataProjection: {nilAsOptionalField: true}});
    show(q1);
    // @output {"name":"a"}
    OptionalAge|error q2 = jsondata:parseAsType({name: "a", age: ()});
    show(q2);
    // @output error: incompatible value 'null' for type 'int' in field 'age'
    OptionalAge|error q3 = jsondata:parseAsType({name: "a"});
    show(q3);
    // @output {"name":"a"}

    NilableFields|error r1 = jsondata:parseAsType({name: "a"}, {allowDataProjection: {absentAsNilableType: true}});
    show(r1);
    // @output {"name":"a","alias":null}
    NilableFields|error r2 = jsondata:parseAsType({name: "a"});
    show(r2);
    // @output error: required field 'alias' not present in JSON

    IntMapField|error i1 = jsondata:parseAsType({m: 5});
    show(i1);
    // @output error: invalid type for field 'm'

    int|error c1 = jsondata:parseAsType(2.5);
    show(c1);
    // @output 2
    int|error c2 = jsondata:parseAsType(3.5);
    show(c2);
    // @output 4
    float|error c3 = jsondata:parseAsType(5);
    show(c3);
    // @output 5.0
    decimal|error c4 = jsondata:parseAsType(1.25);
    show(c4);
    // @output 1.25
    string|error c5 = jsondata:parseAsType(1);
    show(c5);
    // @output error: incompatible expected type 'string' for value '1'
    int|error c6 = jsondata:parseAsType("x");
    show(c6);
    // @output error: incompatible expected type 'int' for value 'x'
    byte|error c7 = jsondata:parseAsType(256);
    show(c7);
    // @output error: incompatible expected type 'byte' for value '256'
    "A"|"B"|error c8 = jsondata:parseAsType("B");
    show(c8);
    // @output B
    "A"|"B"|error c9 = jsondata:parseAsType("C");
    show(c9);
    // @output error: incompatible expected type '("A"|"B")' for value 'C'
    ()|error c10 = jsondata:parseAsType(());
    show(c10);
    // @output 
    string?|error c11 = jsondata:parseAsType(());
    show(c11);
    // @output 
    boolean|error c12 = jsondata:parseAsType(true);
    show(c12);
    // @output true

    int|string|error d1 = jsondata:parseAsType("text");
    show(d1);
    // @output text
    Address|int[]|error d2 = jsondata:parseAsType([1, 2]);
    show(d2);
    // @output [1,2]
    Address|int[]|error d3 = jsondata:parseAsType({city: "X", zip: 1});
    show(d3);
    // @output {"city":"X","zip":1}

    json|error j1 = jsondata:parseAsType({a: [1, {b: null}]});
    show(j1);
    // @output {"a":[1,{"b":null}]}
    anydata|error j2 = jsondata:parseAsType([1, "two"]);
    show(j2);
    // @output [1,"two"]

    ReadonlyAddress|error ro1 = jsondata:parseAsType({city: "Matara", zip: 400});
    show(ro1 is readonly);
    // @output true
    show(ro1);
    // @output {"city":"Matara","zip":400}
}
