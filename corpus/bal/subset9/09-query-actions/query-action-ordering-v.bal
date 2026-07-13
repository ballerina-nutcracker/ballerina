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

function track(int[] evaluated, int value) returns int {
    evaluated.push(value);
    return value;
}

function trackCollection(int[] events, int marker, int[] values) returns int[] {
    events.push(marker);
    return values;
}

function trackKey(int[] events, int value) returns int {
    events.push(value + 10);
    return value;
}

public function main() {
    int state = 0;
    from var value in [1, 2, 3]
        let int observedState = state
        do {
            io:println(value, ":", observedState); // @output 1:0
            // @output 2:1
            // @output 3:2
            state += 1;
        };

    int whereState = 0;
    from var value in [0, 1, 2]
        where value <= whereState
        do {
            io:println("where:", value); // @output where:0
            // @output where:1
            // @output where:2
            whereState += 1;
        };

    int joinState = 0;
    from var left in [1, 2, 3]
        join var right in [1, 2, 3]
        on left equals right
        let int observedState = joinState
        do {
            io:println("join:", left, ":", observedState); // @output join:1:0
            // @output join:2:1
            // @output join:3:2
            joinState += 1;
        };

    int beforeOrderState = 0;
    from var value in [3, 2, 1]
        let int observedState = beforeOrderState
        order by value
        do {
            io:println("before-order:", value, ":", observedState); // @output before-order:1:0
            // @output before-order:2:0
            // @output before-order:3:0
            beforeOrderState += 1;
        };

    int afterOrderState = 0;
    from var value in [3, 2, 1]
        order by value
        let int observedState = afterOrderState
        do {
            io:println("after-order:", value, ":", observedState); // @output after-order:1:0
            // @output after-order:2:1
            // @output after-order:3:2
            afterOrderState += 1;
        };

    int outerJoinState = 0;
    from var key in [1, 2]
        outer join var candidate in [2]
        on key equals candidate
        do {
            io:println("outer:", key, ":", candidate is (), ":", outerJoinState); // @output outer:1:true:0
            // @output outer:2:false:1
            outerJoinState += 1;
        };

    int[] limitEvaluated = [];
    from var value in [1, 2, 3]
        let int recorded = track(limitEvaluated, value)
        limit 1
        do {
            io:println("limit:", recorded); // @output limit:1
        };
    io:println("limit-evaluated:", limitEvaluated.length()); // @output limit-evaluated:1

    from var left in [1, 2]
        limit 1
        join var right in [1]
        on left equals right
        do {
            io:println("limit-before-join:", left); // @output limit-before-join:1
        };

    int[] joinLimitEvaluated = [];
    from var left in [1]
        join var right in [1, 1, 1]
        on left equals right
        let int recorded = track(joinLimitEvaluated, right)
        limit 1
        do {
            io:println("limit-after-join:", recorded); // @output limit-after-join:1
        };
    io:println("join-limit-evaluated:", joinLimitEvaluated.length()); // @output join-limit-evaluated:1

    int[] joinEvents = [];
    from var left in trackCollection(joinEvents, 1, [1, 2])
        join var right in trackCollection(joinEvents, 2, [1, 2])
        on left equals trackKey(joinEvents, right)
        do {
            joinEvents.push(left + 20);
        };
    io:println("join-events:", joinEvents); // @output join-events:[1,2,11,12,21,22]

    int[] zeroLimitEvaluated = [];
    from var value in [1, 2, 3]
        let int recorded = track(zeroLimitEvaluated, value)
        limit 0
        do {
            io:println("unexpected-zero-limit:", recorded);
        };
    io:println("zero-limit-evaluated:", zeroLimitEvaluated.length()); // @output zero-limit-evaluated:0
}
