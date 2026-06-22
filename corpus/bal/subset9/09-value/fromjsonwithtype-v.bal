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

type IntArray int[];
type StringArray string[];
type IntMap map<int>;
type IntStringTuple [int, string];
type OpenIntStringTuple [int, string, int...];

type Person record {|
    string name;
    int age;
|};

type PersonNilAge record {|
    string name;
    int? age;
|};

type OpenPerson record {
    string name;
};

type Address record {|
    string city;
|};

type Employee record {|
    string name;
    Address address;
|};

public function main() returns error? {
    json arrForAny = [1, 2];
    anydata ad = check arrForAny.fromJsonWithType(anydata);
    io:println(ad); // @output [1,2]

    // primitives
    json num = 2022;
    int|error n = num.fromJsonWithType(int);
    io:println(n); // @output 2022

    boolean b = check true.fromJsonWithType(boolean);
    io:println(b); // @output true
    boolean f = check false.fromJsonWithType(boolean);
    io:println(f); // @output false

    json strVal = "hello";
    string s = check strVal.fromJsonWithType(string);
    io:println(s); // @output hello

    // arrays
    json arr = [1, 2, 3, 4];
    int[] intArray = check arr.fromJsonWithType(IntArray);
    io:println(intArray); // @output [1,2,3,4]

    int[] inferred = check arr.fromJsonWithType();
    io:println(inferred); // @output [1,2,3,4]

    json vowels = ["a", "e", "i", "o", "u"];
    string[] chars = check vowels.fromJsonWithType(StringArray);
    io:println(chars); // @output ["a","e","i","o","u"]

    // maps
    json m = {"a": 1, "b": 2};
    map<int> mi = check m.fromJsonWithType(IntMap);
    io:println(mi); // @output {"a":1,"b":2}

    // records
    json p = {"name": "Alice", "age": 30};
    Person person = check p.fromJsonWithType(Person);
    io:println(person); // @output {"name":"Alice","age":30}

    // required nilable field absent in source → nil injected
    json noAge = {"name": "Alice"};
    PersonNilAge nilAgePerson = check noAge.fromJsonWithType(PersonNilAge);
    io:println(nilAgePerson); // @output {"name":"Alice","age":null}

    // open record — extra fields are preserved
    json extra = {"name": "Eve", "role": "admin"};
    OpenPerson op = check extra.fromJsonWithType(OpenPerson);
    io:println(op); // @output {"name":"Eve","role":"admin"}

    // nested record
    json emp = {"name": "Bob", "address": {"city": "NYC"}};
    Employee empVal = check emp.fromJsonWithType(Employee);
    io:println(empVal); // @output {"name":"Bob","address":{"city":"NYC"}}

    // tuples
    json tuple = [1, "x"];
    IntStringTuple tupleVal = check tuple.fromJsonWithType(IntStringTuple);
    io:println(tupleVal); // @output [1,"x"]

    // open tuple with rest elements
    json restTuple = [1, "x", 2, 3];
    OpenIntStringTuple openTupleVal = check restTuple.fromJsonWithType(OpenIntStringTuple);
    io:println(openTupleVal); // @output [1,"x",2,3]

    // open tuple with only fixed members (no rest elements)
    json fixedOnly = [1, "y"];
    OpenIntStringTuple fixedOnlyVal = check fixedOnly.fromJsonWithType(OpenIntStringTuple);
    io:println(fixedOnlyVal); // @output [1,"y"]
    return;
}
