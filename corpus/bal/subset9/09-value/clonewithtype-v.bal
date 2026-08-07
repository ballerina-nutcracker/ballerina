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

type Person record {|
    string name;
    int age;
|};

type PersonNilAge record {|
    string name;
    int? age;
|};

type PersonOptAge record {|
    string name;
    int? age?;
|};

type Point record {|
    float x;
    float y;
|};

type Shape record {|
    string label;
    Point origin;
|};

type IntArray int[];
type FloatArray float[];

type PersonOrInt Person|int;

public function main() returns error? {
    // primitives: anydata → specific type
    anydata n = 42;
    int i = check n.cloneWithType(int);
    io:println(i); // @output 42

    anydata s = "hello";
    string str = check s.cloneWithType(string);
    io:println(str); // @output hello

    // numeric conversion: int → float
    anydata intVal = 7;
    float f = check intVal.cloneWithType(float);
    io:println(f); // @output 7.0

    // map → closed record
    anydata raw = {name: "Alice", age: 30};
    Person p = check raw.cloneWithType(Person);
    io:println(p); // @output {"name":"Alice","age":30}

    // anydata array → typed array
    anydata nums = [1, 2, 3];
    IntArray ia = check nums.cloneWithType(IntArray);
    io:println(ia); // @output [1,2,3]

    // int array (as anydata) → float array via numeric conversion
    anydata ints = [10, 20, 30];
    FloatArray fa = check ints.cloneWithType(FloatArray);
    io:println(fa); // @output [10.0,20.0,30.0]

    // nested: map with int fields → record with float fields
    anydata coords = {x: 3, y: 4};
    Point pt = check coords.cloneWithType(Point);
    io:println(pt); // @output {"x":3.0,"y":4.0}

    // nested record
    anydata shapeRaw = {label: "origin", origin: {x: 0, y: 0}};
    Shape sh = check shapeRaw.cloneWithType(Shape);
    io:println(sh); // @output {"label":"origin","origin":{"x":0.0,"y":0.0}}

    // union: picks first matching type
    anydata personMap = {name: "Bob", age: 25};
    PersonOrInt poi = check personMap.cloneWithType(PersonOrInt);
    io:println(poi); // @output {"name":"Bob","age":25}

    // required nilable field present as () — nil is a valid value
    anydata withNull = {name: "Carol", age: ()};
    PersonNilAge nilAge = check withNull.cloneWithType(PersonNilAge);
    io:println(nilAge); // @output {"name":"Carol","age":null}

    // optional nilable field absent — ok to omit
    anydata partial = {name: "Dave"};
    PersonOptAge optAge = check partial.cloneWithType(PersonOptAge);
    io:println(optAge); // @output {"name":"Dave"}

    return;
}
