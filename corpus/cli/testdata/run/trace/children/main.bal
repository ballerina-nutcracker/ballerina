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

// Every frontend phase decomposes into children over this package: constants
// and global variables, a type definition that depends on another, a class with
// methods, a service with resource methods, and several top-level functions.

import ballerina/io;

const int LIMIT = 3;
const string GREETING = "traced";

type Amount int;

type Entry record {|
    Amount amount;
    string label;
|};

int total = 0;
string lastLabel = GREETING;

class Counter {
    private int value = 0;

    function add(int amount) returns int {
        self.value += amount;
        return self.value;
    }

    function current() returns int {
        return self.value;
    }
}

class SimpleListener {
    public function attach(service object {} svc, string[]|string? attachPoint = ()) returns () {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

service /traced on new SimpleListener() {
    resource function get total() returns int {
        return total;
    }

    resource function get label() returns string {
        return lastLabel;
    }
}

function recordEntry(Entry entry) returns Amount {
    total += entry.amount;
    lastLabel = entry.label;
    return entry.amount;
}

function accumulate(int count) returns int {
    Counter counter = new;
    int index = 0;
    while index < count {
        int _ = counter.add(index);
        index += 1;
    }
    return counter.current();
}

// The attached listener would keep a successful run alive, so main panics once
// it has exercised every definition. The trace is written before the program
// starts, so the recording this fixture exists for is already complete.
public function main() {
    Amount amount = recordEntry({amount: LIMIT, label: GREETING});
    io:println(amount);
    io:println(accumulate(LIMIT));
    io:println(total);
    io:println(lastLabel);
    panic error("traced run complete");
}
