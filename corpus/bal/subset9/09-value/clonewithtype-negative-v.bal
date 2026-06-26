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

type PersonNilRequired record {|
    string name;
    int? age;
|};

type Closed record {|
    string x;
|};

type StringArray string[];
type IntArray int[];

public function main() returns error? {
    // string cannot convert to boolean
    anydata badBool = "true";
    io:println(badBool.cloneWithType(boolean) is error); // @output true

    // int array cannot convert to string array
    anydata badArr = [1, 2, 3];
    io:println(badArr.cloneWithType(StringArray) is error); // @output true

    // extra field in source for closed record
    anydata extra = {name: "Alice", age: 30, extra: "oops"};
    io:println(extra.cloneWithType(Person) is error); // @output true

    // missing required non-nilable field
    anydata missing = {name: "Alice"};
    io:println(missing.cloneWithType(Person) is error); // @output true

    // map cannot convert to array
    anydata mapVal = {a: 1};
    io:println(mapVal.cloneWithType(IntArray) is error); // @output true

    // array cannot convert to record
    anydata arrVal = [1, 2];
    io:println(arrVal.cloneWithType(Closed) is error); // @output true

    // required nilable field absent — field must be present even if null
    anydata missingNilable = {name: "Bob"};
    io:println(missingNilable.cloneWithType(PersonNilRequired) is error); // @output true

    return;
}
