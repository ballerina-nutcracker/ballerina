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
type IntStringTuple [int, string];

type Person record {|
    string name;
    int age;
|};

type Closed record {|
    string x;
|};

type PersonOrClosed Person|Closed;

type WithRequiredScore record {|
    string name;
    int score;
|};

public function main() {
    checkpanic run();
}

function run() returns error? {
    json badBool = "2022";
    io:println(badBool.fromJsonWithType(boolean) is error); // @output true

    json badArr = ["a", "b"];
    io:println(badArr.fromJsonWithType(IntArray) is error); // @output true

    json badNumArr = [1, 2];
    io:println(badNumArr.fromJsonWithType(StringArray) is error); // @output true

    json badString = "foobar";
    io:println(badString.fromJsonWithType(int) is error); // @output true

    json missingField = {"name": "Alice"};
    io:println(missingField.fromJsonWithType(Person) is error); // @output true

    json badAge = {"name": "Alice", "age": "old"};
    io:println(badAge.fromJsonWithType(Person) is error); // @output true

    json extraField = {"x": "a", "y": 1};
    io:println(extraField.fromJsonWithType(Closed) is error); // @output true

    json nilVal = ();
    io:println(nilVal.fromJsonWithType(Person) is error); // @output true

    json notPerson = {"a": 1};
    io:println(notPerson.fromJsonWithType(PersonOrClosed) is error); // @output true

    json shortTuple = [1];
    io:println(shortTuple.fromJsonWithType(IntStringTuple) is error); // @output true

    json longTuple = [1, "a", 2];
    io:println(longTuple.fromJsonWithType(IntStringTuple) is error); // @output true

    json missingScore = {"name": "Dave"};
    io:println(missingScore.fromJsonWithType(WithRequiredScore) is error); // @output true
    return;
}
