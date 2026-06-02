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

type Person record {|
    string name;
    int age;
|};

public function main() {
    checkpanic run();
}

function run() returns error? {
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
    return;
}
