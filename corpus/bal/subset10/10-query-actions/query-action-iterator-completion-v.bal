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

public class FailingIterator {
    int index = 0;

    public isolated function next() returns record {|int value;|}|error? {
        if self.index == 0 {
            self.index += 1;
            return {value: 10};
        }
        if self.index == 1 {
            self.index += 1;
            return {value: 20};
        }
        return error("iterator completion failed");
    }
}

class FailingIterable {
    *object:Iterable;

    public function iterator() returns FailingIterator {
        return new;
    }
}

public class CompleteIterator {
    int index = 0;

    public isolated function next() returns record {|int value;|}? {
        if self.index >= 2 {
            return ();
        }
        self.index += 1;
        return {value: self.index};
    }
}

class CompleteIterable {
    *object:Iterable;

    public function iterator() returns CompleteIterator {
        return new;
    }
}

public class TrackedFailingIterator {
    int[] events;
    int marker;

    function init(int[] events, int marker) {
        self.events = events;
        self.marker = marker;
    }

    public function next() returns record {|int value;|}|error? {
        self.events.push(self.marker);
        if self.marker == 1 {
            return error("first join completion failed");
        }
        return error("second join completion failed");
    }
}

class TrackedFailingIterable {
    *object:Iterable;
    int[] events;
    int marker;

    function init(int[] events, int marker) {
        self.events = events;
        self.marker = marker;
    }

    public function iterator() returns TrackedFailingIterator {
        self.events.push(self.marker + 10);
        return new (self.events, self.marker);
    }
}

function trackedFailingIterable(int[] events, int marker) returns TrackedFailingIterable {
    events.push(marker + 20);
    return new (events, marker);
}

function trackedLimit(int[] events) returns int {
    events.push(30);
    return 1;
}

public function main() {
    int total = 0;
    error? result = from var value in new FailingIterable()
        do {
            total += value;
        };
    io:println(total); // @output 30
    io:println(result is error); // @output true
    if result is error {
        io:println(result.message()); // @output iterator completion failed
    }

    int nestedTotal = 0;
    result = from var factor in [1, 2]
        from var value in new FailingIterable()
        do {
            nestedTotal += factor * value;
        };
    io:println(nestedTotal); // @output 30
    io:println(result is error); // @output true

    int orderedTotal = 0;
    result = from var value in new FailingIterable()
        order by value descending
        do {
            orderedTotal += value;
        };
    io:println(orderedTotal); // @output 0
    io:println(result is error); // @output true

    int joinedTotal = 0;
    result = from var left in [10, 20]
        join var right in new FailingIterable()
        on left equals right
        do {
            joinedTotal += left + right;
        };
    io:println(joinedTotal); // @output 0
    io:println(result is error); // @output true

    stream<int, error?> secondJoinValues = new (new CompleteIterator());
    int multiJoinTotal = 0;
    result = from var left in [1, 2]
        join var first in new CompleteIterable()
        on left equals first
        join var second in secondJoinValues
        on first equals second
        do {
            multiJoinTotal += left + first + second;
        };
    io:println(multiJoinTotal); // @output 9
    io:println(result is ()); // @output true

    int[] multiJoinEvents = [];
    result = from var left in [1]
        join var first in trackedFailingIterable(multiJoinEvents, 1)
        on left equals first
        join var second in trackedFailingIterable(multiJoinEvents, 2)
        on first equals second
        limit trackedLimit(multiJoinEvents)
        do {
            multiJoinEvents.push(100);
        };
    io:println(multiJoinEvents); // @output [21,11,1,22,12,2,30]
    if result is error {
        io:println(result.message()); // @output first join completion failed
    }

    int limitedTotal = 0;
    result = from var value in new FailingIterable()
        limit 1
        do {
            limitedTotal += value;
        };
    io:println(limitedTotal); // @output 10
    io:println(result is ()); // @output true

    int completedTotal = 0;
    from var value in new CompleteIterable()
        do {
            completedTotal += value;
        };
    io:println(completedTotal); // @output 3

    stream<int, error?> values = new (new FailingIterator());
    int streamTotal = 0;
    result = from var value in values
        do {
            streamTotal += value;
        };
    io:println(streamTotal); // @output 30
    io:println(result is error); // @output true
}
