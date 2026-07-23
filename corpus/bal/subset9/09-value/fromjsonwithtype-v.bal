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

type Person record {|
    string name;
    int age;
|};

type PersonNilAge record {|
    string name;
    int? age;
|};

public function main() returns error? {
    // json→anydata: the call site is a json value (not just anydata)
    json arrForAny = [1, 2];
    anydata ad = check arrForAny.fromJsonWithType(anydata);
    io:println(ad); // @output [1,2]

    // json primitives
    json num = 2022;
    int n = check num.fromJsonWithType(int);
    io:println(n); // @output 2022

    boolean b = check true.fromJsonWithType(boolean);
    io:println(b); // @output true
    boolean f = check false.fromJsonWithType(boolean);
    io:println(f); // @output false

    json strVal = "hello";
    string s = check strVal.fromJsonWithType(string);
    io:println(s); // @output hello

    // required nilable field absent → error (field must be present, even if null)
    json noAge = {"name": "Alice"};
    io:println(noAge.fromJsonWithType(PersonNilAge) is error); // @output true

    // required nilable field present as json null → nil assigned
    json nullAge = {"name": "Alice", "age": null};
    PersonNilAge nilAgePerson = check nullAge.fromJsonWithType(PersonNilAge);
    io:println(nilAgePerson); // @output {"name":"Alice","age":null}

    // typedesc inferred from context
    json arr = [1, 2, 3, 4];
    int[] inferred = check arr.fromJsonWithType();
    io:println(inferred); // @output [1,2,3,4]

    // basic record conversion
    json p = {"name": "Alice", "age": 30};
    Person person = check p.fromJsonWithType(Person);
    io:println(person); // @output {"name":"Alice","age":30}

    return;
}
