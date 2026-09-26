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

function valuesFor(int input) returns int[] {
    return [input, input + 10];
}

public function main() {
    int[] sums = from var left in [1, 2]
        from var right in [10, 20]
        select left + right;
    io:println(sums); // @output [11,21,12,22]

    int[] dependent = from var base in [1, 2]
        from var expanded in valuesFor(base)
        select base + expanded;
    io:println(dependent); // @output [2,12,4,14]

    map<int[]> groups = {first: [1, 2], second: [3]};
    int[] flattened = from var group in groups
        from var value in group
        select value;
    io:println(flattened); // @output [1,2,3]

    int[] limitedBefore = from var left in [1, 2]
        limit 1
        from var right in [10, 20]
        select left + right;
    io:println(limitedBefore); // @output [11,21]

    int[] limitedAfter = from var left in [1, 2]
        from var right in [10, 20]
        limit 3
        select left + right;
    io:println(limitedAfter); // @output [11,21,12]

    int[] ordered = from var left in [2, 1]
        from var right in [2, 1]
        order by left ascending, right descending
        select left * 10 + right;
    io:println(ordered); // @output [12,11,22,21]

    int[] groupedSizes = from var category in [1, 1, 2]
        from var value in [category, category + 10]
        group by category
        select [value].length();
    io:println(groupedSizes); // @output [4,2]

    int[] joined = from var left in [1, 2]
        from var expanded in valuesFor(left)
        join var candidate in [1, 2]
        on expanded equals candidate
        select left + candidate;
    io:println(joined); // @output [2,4]
}
