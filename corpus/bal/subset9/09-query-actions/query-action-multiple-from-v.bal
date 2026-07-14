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

function expand(int[] events, int value) returns int[] {
    events.push(value);
    return [value, value + 10];
}

public function main() {
    from var left in [1, 2]
        from var right in [10, 20]
        do {
            io:println("pair:", left, ":", right); // @output pair:1:10
            // @output pair:1:20
            // @output pair:2:10
            // @output pair:2:20
        };

    int[] events = [];
    from var value in [1, 2]
        from var expanded in expand(events, value)
        do {
            events.push(expanded + 100);
        };
    io:println("events:", events); // @output events:[1,101,111,2,102,112]

    map<int[]> groups = {first: [1, 2], second: [3]};
    int total = 0;
    from var group in groups
        from var value in group
        do {
            total += value;
        };
    io:println("map-total:", total); // @output map-total:6

    int triples = 0;
    from var first in [1, 2]
        from var second in [10, 20]
        from var third in [100]
        where second > first
        do {
            triples += first + second + third;
        };
    io:println("triple-total:", triples); // @output triple-total:466

    int[] limitedEvents = [];
    from var value in [1, 2, 3]
        from var expanded in expand(limitedEvents, value)
        limit 3
        do {
            limitedEvents.push(expanded + 100);
        };
    io:println("limited-events:", limitedEvents); // @output limited-events:[1,101,111,2,102]

    from var left in [1, 2]
        limit 1
        from var right in [10, 20]
        do {
            io:println("limit-before:", left, ":", right); // @output limit-before:1:10
            // @output limit-before:1:20
        };

    from var left in [2, 1]
        from var right in [2, 1]
        order by left ascending, right descending
        do {
            io:println("ordered:", left, ":", right); // @output ordered:1:2
            // @output ordered:1:1
            // @output ordered:2:2
            // @output ordered:2:1
        };

    from var left in [2, 1]
        order by left
        from var right in [left, left + 10]
        do {
            io:println("after-order:", left, ":", right); // @output after-order:1:1
            // @output after-order:1:11
            // @output after-order:2:2
            // @output after-order:2:12
        };

    from var category in [1, 1, 2]
        from var value in [category, category + 10]
        group by category
        do {
            io:println("grouped:", category, ":", [value].length()); // @output grouped:1:4
            // @output grouped:2:2
        };

    from var left in [1, 2]
        from var expanded in [left, left + 10]
        join var candidate in [1, 2]
        on expanded equals candidate
        do {
            io:println("joined:", left, ":", expanded); // @output joined:1:1
            // @output joined:2:2
        };
}
