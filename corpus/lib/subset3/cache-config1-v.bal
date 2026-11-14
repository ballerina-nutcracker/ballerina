// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import ballerina/cache;
import ballerina/io;

function create(int capacity = 100, float evictionFactor = 0.25, decimal defaultMaxAge = -1,
        decimal? cleanupInterval = ()) {
    cache:Cache|error c;
    if cleanupInterval is decimal {
        c = trap new cache:Cache(capacity = capacity, evictionFactor = evictionFactor,
                defaultMaxAge = defaultMaxAge, cleanupInterval = cleanupInterval);
    } else {
        c = trap new cache:Cache(capacity = capacity, evictionFactor = evictionFactor,
                defaultMaxAge = defaultMaxAge);
    }
    if c is cache:Error {
        io:println(c.message());
    } else if c is error {
        io:println("unexpected: ", c.message());
    } else {
        io:println("ok ", c.capacity());
    }
}

public function main() {
    create(); // @output ok 100
    create(capacity = 1, evictionFactor = 1, defaultMaxAge = -1); // @output ok 1
    create(evictionFactor = 1.0E-300, defaultMaxAge = 0, cleanupInterval = 3600); // @output ok 100
    create(capacity = 0); // @output Validation failed for '$.capacity:minValue' constraint(s).
    create(evictionFactor = 0); // @output Validation failed for '$.evictionFactor:minValueExclusive' constraint(s).
    create(evictionFactor = -0.0); // @output Validation failed for '$.evictionFactor:minValueExclusive' constraint(s).
    create(evictionFactor = 1.5); // @output Validation failed for '$.evictionFactor:maxValue' constraint(s).
    create(evictionFactor = float:Infinity); // @output Validation failed for '$.evictionFactor:maxValue' constraint(s).
    create(evictionFactor = -float:Infinity); // @output Validation failed for '$.evictionFactor:minValueExclusive' constraint(s).
    create(evictionFactor = float:NaN); // @output Validation failed for '$.evictionFactor:maxValue','$.evictionFactor:minValueExclusive' constraint(s).
    create(defaultMaxAge = -2); // @output Validation failed for '$.defaultMaxAge:minValue' constraint(s).
    create(cleanupInterval = 0); // @output Validation failed for '$.cleanupInterval:minValueExclusive' constraint(s).
    create(0, 0, -5, -1); // @output Validation failed for '$.capacity:minValue','$.cleanupInterval:minValueExclusive','$.defaultMaxAge:minValue','$.evictionFactor:minValueExclusive' constraint(s).
    create(-3, 1.0001, -1.5, 0); // @output Validation failed for '$.capacity:minValue','$.cleanupInterval:minValueExclusive','$.defaultMaxAge:minValue','$.evictionFactor:maxValue' constraint(s).

    cache:EvictionPolicy policy = cache:LRU;
    cache:CacheConfig config = {capacity: 5, evictionPolicy: policy};
    cache:Cache fromRecord = new (config);
    io:println(fromRecord.capacity(), " ", cache:LRU); // @output 5 LRU

    // A sub-millisecond cleanup interval can never be scheduled. Whether the start time is already in the past or the
    // interval truncates to zero milliseconds depends on whether the wall clock ticked between the two clock reads.
    decimal interval = 0.000001;
    cache:Cache|error tooShort = trap new cache:Cache(cleanupInterval = interval);
    if tooShort is cache:Error {
        io:println(isSubMillisecondScheduleError(tooShort.message())); // @output true
    } else {
        io:println("unexpected: ", tooShort is error ? tooShort.message() : "scheduled");
    }
}

function isSubMillisecondScheduleError(string message) returns boolean {
    string prefix = "Failed to schedule the cleanup task: ";
    if message == prefix + "Repeat Interval cannot be zero." {
        return true;
    }
    string pastStartSuffix = ". Scheduled time should be greater than the current time";
    string pastStartPrefix = prefix + "Invalid time: ";
    int suffixStart = message.length() - pastStartSuffix.length();
    return suffixStart > pastStartPrefix.length() && message.indexOf(pastStartPrefix) == 0
        && message.substring(suffixStart) == pastStartSuffix;
}
