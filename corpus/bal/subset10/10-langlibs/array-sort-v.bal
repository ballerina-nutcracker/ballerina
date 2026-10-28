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
import ballerina/lang.array;

type Person record {|
    string name;
    int age;
|};

public function main() {
    int[] values = [3, 1, 2];
    io:println(values.sort()); // @output [1,2,3]
    io:println(values.sort(array:DESCENDING)); // @output [3,2,1]
    io:println(values); // @output [3,1,2]

    int[] sorted = values.sort(direction = array:ASCENDING);
    sorted.push(4);
    io:println(sorted); // @output [1,2,3,4]

    string[] words = ["pear", "fig", "apple"];
    io:println(words.sort()); // @output ["apple","fig","pear"]
    io:println(words.sort(key = isolated function(string w) returns int => w.length())); // @output ["fig","pear","apple"]

    int?[] withNil = [2, (), 1];
    io:println(withNil.sort()); // @output [1,2,null]
    io:println(withNil.sort(array:DESCENDING)); // @output [2,1,null]

    float[] floats = [2.0, float:NaN, 1.0];
    io:println(floats.sort()); // @output [1.0,2.0,NaN]
    io:println(floats.sort(array:DESCENDING)); // @output [2.0,1.0,NaN]

    int[][] lists = [[2, 1], [1, 5], [1]];
    io:println(lists.sort()); // @output [[1],[1,5],[2,1]]

    Person[] people = [{name: "b", age: 30}, {name: "a", age: 20}, {name: "c", age: 30}];
    Person[] byAge = people.sort(array:DESCENDING, isolated function(Person p) returns int => p.age);
    io:println(byAge.map(p => p.name)); // @output ["b","c","a"]

    [int, int...] t = [3, 1];
    int[] fromTuple = t.sort();
    io:println(fromTuple); // @output [1,3]
}
