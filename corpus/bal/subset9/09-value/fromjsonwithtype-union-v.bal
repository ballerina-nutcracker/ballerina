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

type IntOrString int|string;
type JsonOrInt json|int;
type IntNilable int?;

type Person record {|
    string name;
    int age;
|};

type PersonOpt record {|
    string name;
    int age?;
|};

type WithNilable record {|
    string name;
    int? score;
|};

type PersonNilAge record {|
    string name;
    int? age;
|};

type AnyMap map<any>;

public function main() returns error? {
    json num = 42;
    IntOrString asUnion = check num.fromJsonWithType(IntOrString);
    io:println(asUnion); // @output 42

    // string value directly matches string member of union.
    json str = "hello";
    IntOrString asStr = check str.fromJsonWithType(IntOrString);
    io:println(asStr); // @output hello

    json arr = [1, 2];
    JsonOrInt asJson = check arr.fromJsonWithType(JsonOrInt);
    io:println(asJson); // @output [1,2]

    json nilVal = ();
    IntNilable asNilable = check nilVal.fromJsonWithType(IntNilable);
    io:println(asNilable is IntNilable); // @output true

    json partial = {"name": "Bob"};
    PersonOpt withOptional = check partial.fromJsonWithType(PersonOpt);
    io:println(withOptional); // @output {"name":"Bob"}

    json missingNilable = {"name": "Carol"};
    WithNilable withScore = check missingNilable.fromJsonWithType(WithNilable);
    io:println(withScore); // @output {"name":"Carol","score":null}

    json withNullAge = {"name": "Ann", "age": ()};
    PersonNilAge withAge = check withNullAge.fromJsonWithType(PersonNilAge);
    io:println(withAge); // @output {"name":"Ann","age":null}

    json openMap = {"a": 1};
    AnyMap anyMap = check openMap.fromJsonWithType(AnyMap);
    io:println(anyMap); // @output {"a":1}
    return;
}
