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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

function breakFromQueryAction() returns int[] {
    int[] events = [];
    int cycle = 0;
    while cycle < 3 {
        cycle += 1;
        from var value in [1, 2, 3]
        do {
            events.push(cycle * 10 + value);
            if cycle == 2 && value == 2 {
                break;
            }
        };
        events.push(cycle * 100);
    }
    return events;
}

function continueFromQueryAction() returns int[] {
    int[] events = [];
    foreach int cycle in [1, 2, 3] {
        from var value in [1, 2, 3]
        do {
            events.push(cycle * 10 + value);
            if value == 2 {
                continue;
            }
        };
        events.push(cycle * 100);
    }
    return events;
}

function continueFromMultipleFromClauses() returns int[] {
    int[] events = [];
    foreach int cycle in [1, 2] {
        from var first in [1, 2]
        from var second in [1, 2]
        do {
            events.push(cycle * 100 + first * 10 + second);
            if first == 1 && second == 2 {
                continue;
            }
        };
        events.push(cycle * 1000);
    }
    return events;
}

function breakFromJoinClause() returns int[] {
    int[] events = [];
    while true {
        from var left in [1, 2]
        join var right in [1, 2]
        on left equals right
        do {
            events.push(left);
            break;
        };
        events.push(100);
    }
    return events;
}

function continueFromNestedQueryAction() returns int[] {
    int[] events = [];
    foreach int cycle in [1, 2] {
        from var first in [1, 2]
        do {
            events.push(cycle * 100 + first * 10);
            from var second in [1, 2]
            do {
                events.push(cycle * 100 + first * 10 + second);
                if first == 1 && second == 2 {
                    continue;
                }
            };
            events.push(cycle * 100 + first);
        };
        events.push(cycle * 1000);
    }
    return events;
}

function breakFromNestedQueryAction() returns int[] {
    int[] events = [];
    while true {
        from var first in [1, 2]
        do {
            events.push(first * 10);
            from var second in [1, 2]
            do {
                events.push(first * 10 + second);
                if first == 1 && second == 2 {
                    break;
                }
            };
            events.push(first);
        };
        events.push(100);
    }
    return events;
}

function preserveLocalLoopControl() returns int[] {
    int[] events = [];
    foreach int cycle in [1] {
        from var value in [1, 2]
        do {
            int local = 0;
            while local < 3 {
                local += 1;
                if local == 1 {
                    continue;
                }
                if local == 2 {
                    break;
                }
            }
            events.push(value * 10 + local);
        };
        events.push(cycle * 100);
    }
    return events;
}

public function main() {
    io:println(breakFromQueryAction()); // @output [11,12,13,100,21,22]
    io:println(continueFromQueryAction()); // @output [11,12,21,22,31,32]
    io:println(continueFromMultipleFromClauses()); // @output [111,112,211,212]
    io:println(breakFromJoinClause()); // @output [1]
    io:println(continueFromNestedQueryAction()); // @output [110,111,112,210,211,212]
    io:println(breakFromNestedQueryAction()); // @output [10,11,12]
    io:println(preserveLocalLoopControl()); // @output [12,22,100]
}
