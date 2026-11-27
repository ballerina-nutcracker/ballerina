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

public isolated function querySort(any[] sortKeyRows, any[] sortDirections, any[] rowIndices, any[] payloadRows) = external;

public isolated function queryGroup(any[] rows, any[] keyRows, any[] scalarFlags) returns any[] = external;

public isolated function queryCollect(any[] rows, int slotCount, any[] flattenFlags) returns any[] = external;

public isolated function escapeXMLContent(string|boolean|int|float|decimal value) returns string = external;

public isolated function escapeXMLAttribute(string|boolean|int|float|decimal value) returns string = external;

# Returns the value of a configurable module variable. The runtime looks the variable up by
# organization, module and variable name in the configuration sources (configuration files or
# environment variables), validates the supplied value against `td`, and returns it as a
# read-only value. If no value is supplied, it returns the result of `defaultSupplier`, which is
# called only in that case; if `defaultSupplier` is `()`, the value is required and module
# initialization fails.
public function configurableValue(string org, string module, string name,
        (function () returns anydata)? defaultSupplier, typedesc<anydata> td) returns anydata = external;

# Creates a fresh closed one-shot latch. Strands that wait on it are held until
# it is opened, and it can never be closed again.
public isolated function createLatch() returns handle = external;

# Returns only after the latch opens, yielding cooperatively while it is closed.
public isolated function waitOnLatch(handle latch) = external;

# Opens the latch, releasing every waiting strand. A latch may be opened once.
public isolated function openLatch(handle latch) = external;

// Defined here, and exported by lang.object as `object:Iterable`, so that the
// value of a range expression, created below, can belong to it.
public type Iterable distinct object {
    # Create a new iterator.
    #
    # + return - a new iterator object
    public function iterator() returns object {
        public function next() returns record {| any|error value; |}|error?;
    };
};

# Iterates over the integers of a range in increasing order.
class IntRangeIterator {
    private int current;
    // The last integer of the range; meaningful only while `done` is false.
    private final int last;
    private boolean done;

    isolated function init(int 'start, int end, boolean closed) {
        self.current = 'start;
        self.done = closed ? 'start > end : 'start >= end;
        // When the range is not empty, `end - 1` cannot overflow, since `end` > `start`.
        self.last = self.done || closed ? end : end - 1;
    }

    public isolated function next() returns record {| int value; |}? {
        if self.done {
            return ();
        }
        int value = self.current;
        if value == self.last {
            self.done = true;
        } else {
            self.current = value + 1;
        }
        return {value};
    }
}

# The value of a range expression, an `Iterable<int,()>`. Each call to `iterator`
# iterates over the whole range again.
class IntRange {
    *Iterable;
    private final int 'start;
    private final int end;
    private final boolean closed;

    isolated function init(int 'start, int end, boolean closed) {
        self.'start = 'start;
        self.end = end;
        self.closed = closed;
    }

    public isolated function iterator() returns object {
        public isolated function next() returns record {| int value; |}?;
    } {
        return new IntRangeIterator(self.'start, self.end, self.closed);
    }
}

# Creates the value of a range expression that is not lowered to a foreach loop.
#
# + 'start - the first operand
# + end - the second operand
# + closed - true for `...`, false for `..<`
# + return - an object iterating over the integers of the range
public isolated function createIntRange(int 'start, int end, boolean closed) returns Iterable & object {
    public isolated function iterator() returns object {
        public isolated function next() returns record {| int value; |}?;
    };
} {
    return new IntRange('start, end, closed);
}
