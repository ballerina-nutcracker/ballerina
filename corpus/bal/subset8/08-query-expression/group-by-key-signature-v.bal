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

public function main() {
    string[] words = ["a", "b", "a"];
    var groupedByString = from var word in words
        let string original = word
        group by word
        select [word, original];

    float nan = 0.0 / 0.0;
    float[] floats = [1.0, -0.0, 0.0, nan, 0.0 / 0.0, 1.0];
    var groupedByFloat = from var value in floats
        let float original = value
        group by value
        select [value, original];

    int?[] optionalNumbers = [1, (), (), 2];
    var groupedByNil = from var value in optionalNumbers
        let int? original = value
        group by value
        select [value, original];

    decimal[] decimals = [1.0, 1.00, 2.0];
    var groupedByDecimal = from var value in decimals
        let decimal original = value
        group by value
        select [value, original];

    int[] numbers = [1, 2, 1];
    var groupedByMultipleKeys = from var value in numbers
        group by var listKey = [value], var even = value % 2 == 0
        select [listKey, even, value];

    map<int>[] mappings = [{a: 1, b: 2}, {b: 2, a: 1}, {a: 1, b: 3}];
    var groupedByMap = from var mapping in mappings
        group by var key = mapping
        select [key, mapping];

    io:println(groupedByString); // @output [["a","a","a"],["b","b"]]
    io:println(groupedByFloat); // @output [[1.0,1.0,1.0],[-0.0,-0.0,0.0],[NaN,NaN,NaN]]
    io:println(groupedByNil); // @output [[1,1],[null,null,null],[2,2]]
    io:println(groupedByDecimal); // @output [[1,1,1],[2,2]]
    io:println(groupedByMultipleKeys); // @output [[[1],false,1,1],[[2],true,2]]
    io:println(groupedByMap); // @output [[{"a":1,"b":2},{"a":1,"b":2},{"b":2,"a":1}],[{"a":1,"b":3},{"a":1,"b":3}]]
}
