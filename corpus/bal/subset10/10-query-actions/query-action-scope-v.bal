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

final int item = 2;

public function main() {
    int[] candidates = [2, 3, 4];
    int offset = 1;
    int maxMatches = 2;
    int total = 0;

    from var item in [1, 2, 3]
        let int shifted = item + offset,
            int matchKey = shifted
        join var candidate in candidates
        on matchKey equals candidate + offset - offset
        limit maxMatches
        do {
            total += item * 10 + candidate;
        };

    io:println(total); // @output 35

    int joined = 0;
    from var item in [1]
        join var candidate in [item]
        on item + 1 equals candidate + item - item
        limit item
        do {
            joined = candidate;
        };

    io:println(joined); // @output 2

    int nestedJoined = 0;
    from var key in [1]
        join var candidate in (from var key in [2] select key)
        on key + 1 equals candidate
        do {
            nestedJoined = candidate;
        };

    io:println(nestedJoined); // @output 2
}
