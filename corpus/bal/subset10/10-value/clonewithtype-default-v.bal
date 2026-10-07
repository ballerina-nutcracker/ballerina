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

isolated function defaultId() returns int {
    return 40 + 2;
}

type Person record {
    string name;
    int age = 18;
};

type Address record {|
    string city = "Colombo";
    string country = "Sri Lanka";
|};

type Employee record {|
    string name;
    int id = defaultId();
    Address address = {};
    Address[] previous = [];
|};

type Base record {
    boolean active = true;
};

type Manager record {
    *Base;
    string name;
    string level = "L1";
};

type Node record {|
    int value = 0;
    string label = "node";
    Node? next = ();
    Node[] children = [];
|};

type PersonRef Person;

type PersonRefRef PersonRef;

type OptPerson Person?;

type Holder record {|
    PersonRef owner;
    PersonRefRef[] members = [];
    map<PersonRef> byName = {};
|};

type Settings record {
    int? retries = 3;
    float ratio = 2;
    string[] tags = [];
};

isolated function failingDefault() returns int {
    panic error("default must not run");
}

type Tagged record {|
    int a = failingDefault();
    string tag;
|};

type Flag record {|
    int y = 1;
|};

type TaggedInner record {|
    int a = failingDefault();
|};

type TaggedOuter record {|
    TaggedInner inner;
    string z;
|};

type LooseOuter record {|
    map<anydata> inner;
    int z;
|};

type Order record {|
    record {|
        string currency = "LKR";
        int amount;
    |} price;
    record {|int qty = 1;|}[] lines = [];
|};

public function main() returns error? {
    json value = {name: "John"};
    Person person = check value.cloneWithType(Person);
    io:println(person); // @output {"name":"John","age":18}

    Person fromJson = check value.fromJsonWithType(Person);
    io:println(fromJson); // @output {"name":"John","age":18}

    json[] people = [{name: "A"}, {name: "B", age: 5}];
    Person[] persons = check people.cloneWithType();
    io:println(persons); // @output [{"name":"A","age":18},{"name":"B","age":5}]

    json employeeData = {name: "Ann", address: {city: "Kandy"}, previous: [{}, {country: "India"}]};
    Employee employee = check employeeData.cloneWithType(Employee);
    io:println(employee); // @output {"name":"Ann","address":{"city":"Kandy","country":"Sri Lanka"},"previous":[{"city":"Colombo","country":"Sri Lanka"},{"country":"India","city":"Colombo"}],"id":42}

    Manager manager = check value.cloneWithType(Manager);
    io:println(manager); // @output {"name":"John","active":true,"level":"L1"}

    json nodeData = {value: 1, next: {label: "second", next: {value: 3}}, children: [{}, {value: 9}]};
    Node node = check nodeData.cloneWithType();
    io:println(node); // @output {"value":1,"next":{"label":"second","next":{"value":3,"children":[],"label":"node","next":null},"children":[],"value":0},"children":[{"children":[],"label":"node","next":null,"value":0},{"value":9,"children":[],"label":"node","next":null}],"label":"node"}

    PersonRefRef refRef = check value.cloneWithType(PersonRefRef);
    io:println(refRef); // @output {"name":"John","age":18}

    OptPerson optPerson = check value.cloneWithType();
    io:println(optPerson); // @output {"name":"John","age":18}

    Holder holder = check {owner: value, members: [value], byName: {j: value}}.cloneWithType();
    io:println(holder); // @output {"owner":{"name":"John","age":18},"members":[{"name":"John","age":18}],"byName":{"j":{"name":"John","age":18}}}

    Settings explicitNil = check {retries: ()}.cloneWithType();
    io:println(explicitNil); // @output {"retries":null,"ratio":2.0,"tags":[]}

    Settings withRest = check {extra: true}.cloneWithType();
    io:println(withRest); // @output {"extra":true,"ratio":2.0,"retries":3,"tags":[]}

    Settings first = check {}.cloneWithType();
    first.tags.push("x");
    Settings second = check {}.cloneWithType();
    io:println(first.tags, second.tags); // @output ["x"][]

    Order 'order = check {price: {amount: 10}, lines: [{}, {qty: 4}]}.cloneWithType();
    io:println('order); // @output {"price":{"amount":10,"currency":"LKR"},"lines":[{"qty":1},{"qty":4}]}

    record {|string mode = "fast"; int level;|} local = check {level: 2}.cloneWithType();
    io:println(local); // @output {"level":2,"mode":"fast"}

    record {|*Base; string tag = "t";|} localIncluded = check {}.cloneWithType();
    io:println(localIncluded); // @output {"active":true,"tag":"t"}

    Tagged|Flag picked = check {}.cloneWithType();
    io:println(picked); // @output {"y":1}

    TaggedOuter|LooseOuter loose = check {inner: {}, z: 1}.cloneWithType();
    io:println(loose); // @output {"inner":{},"z":1}

    record {}|Flag complete = check {}.cloneWithType();
    io:println(complete); // @output {}

    final int base = 5;
    record {|int x = base;|}|error captured = {}.cloneWithType();
    io:println(captured is error); // @output true

    json missingName = {age: 3};
    Person|error noDefault = missingName.cloneWithType(Person);
    io:println(noDefault is error); // @output true
}
