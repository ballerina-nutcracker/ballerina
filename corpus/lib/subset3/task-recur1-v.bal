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
import ballerina/lang.runtime;
import ballerina/task;
import ballerina/time;

isolated map<int> counts = {};

isolated class CountJob {
    *task:Job;
    private final string name;

    isolated function init(string name) {
        self.name = name;
    }

    public isolated function execute() {
        lock {
            counts[self.name] = (counts[self.name] ?: 0) + 1;
        }
    }
}

isolated function count(string name) returns int {
    lock {
        return counts[name] ?: 0;
    }
}

isolated function isRunning(task:JobId jobId) returns boolean {
    foreach task:JobId id in task:getRunningJobs() {
        if id == jobId {
            return true;
        }
    }
    return false;
}

isolated function inSeconds(decimal seconds) returns time:Civil {
    return time:utcToCivil(time:utcAddSeconds(time:utcNow(), seconds));
}

public function main() returns error? {
    task:JobId limited = check task:scheduleJobRecurByFrequency(new CountJob("limited"), 0.05, maxCount = 3);
    task:JobId windowed = check task:scheduleJobRecurByFrequency(new CountJob("windowed"), 0.05,
            startTime = inSeconds(0.1), endTime = inSeconds(0.3));
    task:JobId forever = check task:scheduleJobRecurByFrequency(new CountJob("forever"), 0.05);
    io:println(isRunning(limited), " ", isRunning(windowed), " ", isRunning(forever)); // @output true true true
    runtime:sleep(0.7);
    io:println("limited: ", count("limited")); // @output limited: 3
    io:println("windowed in range: ", count("windowed") >= 3 && count("windowed") <= 6); // @output windowed in range: true
    io:println("forever fired: ", count("forever") > 3); // @output forever fired: true
    io:println(isRunning(limited), " ", isRunning(windowed), " ", isRunning(forever)); // @output false false true

    check task:unscheduleJob(forever);
    int stopped = count("forever");
    runtime:sleep(0.2);
    io:println("stopped: ", count("forever") == stopped); // @output stopped: true
    io:println(task:getRunningJobs().length()); // @output 0

    task:JobId single = check task:scheduleJobRecurByFrequency(new CountJob("single"), 0, maxCount = 1);
    runtime:sleep(0.2);
    io:println("single: ", count("single"), " ", isRunning(single)); // @output single: 1 false

    task:JobId|task:Error r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1, maxCount = 0);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output The maxCount should be a positive integer.
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1, maxCount = -2);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output The maxCount should be a positive integer.
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 0);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output Repeat Interval cannot be zero.
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 0.0004);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output Repeat Interval cannot be zero.
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), -1);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output Repeat interval must be >= 0
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1, startTime = inSeconds(10), endTime = inSeconds(5));
    io:println(r is task:Error ? r.message() : "scheduled"); // @output End time cannot be before start time
    time:Civil past = {year: 2020, month: 1, day: 1, hour: 0, minute: 0, second: 0};
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1, startTime = past);
    io:println(r is task:Error ? r.message().substring(0, 13) : "scheduled"); // @output Invalid time:
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1, endTime = past);
    io:println(r is task:Error ? r.message().substring(0, 13) : "scheduled"); // @output Invalid time:
    io:println(task:getRunningJobs().length()); // @output 0
}
