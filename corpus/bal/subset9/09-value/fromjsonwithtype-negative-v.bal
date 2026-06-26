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

type Closed record {|
    string x;
|};

type PersonOrClosed Person|Closed;

type PersonNilRequired record {|
    string name;
    int? age;
|};

public function main() returns error? {
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

    // open tuple: source shorter than fixed members
    json shortOpenTuple = [1];
    io:println(shortOpenTuple.fromJsonWithType(OpenIntStringTuple) is error); // @output true

    // null to non-nilable scalar
    json nullVal = ();
    io:println(nullVal.fromJsonWithType(int) is error); // @output true

    // number to boolean
    json numBool = 42;
    io:println(numBool.fromJsonWithType(boolean) is error); // @output true

    // array where map expected
    json arrForMap = [1, 2];
    io:println(arrForMap.fromJsonWithType(IntMap) is error); // @output true

    // map where array expected
    json mapForArr = {"a": 1};
    io:println(mapForArr.fromJsonWithType(IntArray) is error); // @output true

    // required nilable field absent is an error — the field must be present, even if null
    json missingNilableReq = {"name": "Bob"};
    io:println(missingNilableReq.fromJsonWithType(PersonNilRequired) is error); // @output true

    return;
}
