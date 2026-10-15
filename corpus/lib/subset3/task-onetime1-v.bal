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
    task:JobId once = check task:scheduleOneTimeJob(new CountJob("once"), inSeconds(0.5));
    io:println(isRunning(once)); // @output true
    runtime:sleep(0.8);
    io:println("once: ", count("once"), " ", isRunning(once)); // @output once: 1 false
    task:Error? e = task:unscheduleJob(once);
    io:println(e is task:Error ? e.message() == "Invalid job id: " + once.id.toString() : "unscheduled"); // @output true

    task:JobId held = check task:scheduleOneTimeJob(new CountJob("held"), inSeconds(0.1));
    check task:pauseJob(held);
    runtime:sleep(0.3);
    io:println("held while paused: ", count("held"), " ", isRunning(held)); // @output held while paused: 0 true
    check task:resumeJob(held);
    runtime:sleep(0.2);
    io:println("held after resume: ", count("held")); // @output held after resume: 1

    task:JobId done = check task:scheduleOneTimeJob(new CountJob("done"), inSeconds(0.05));
    runtime:sleep(0.2);
    check task:unscheduleJob(done);
    e = task:unscheduleJob(done);
    io:println(e is task:Error ? e.message() == "Invalid job id: " + done.id.toString() : "unscheduled"); // @output true

    time:Civil past = {year: 2020, month: 1, day: 1, hour: 0, minute: 0, second: 0};
    task:JobId|task:Error r = task:scheduleOneTimeJob(new CountJob("x"), past);
    if r is task:Error {
        io:println(r.message().substring(0, 13)); // @output Invalid time:
        io:println(r.message().substring(r.message().length() - 56) == ". Scheduled time should be greater than the current time"); // @output true
    }
    time:Civil badMonth = {year: 2030, month: 13, day: 1, hour: 0, minute: 0, second: 0};
    r = task:scheduleOneTimeJob(new CountJob("x"), badMonth);
    if r is task:Error {
        io:println(r.message().substring(0, 45)); // @output Couldn't convert given time to milli seconds:
    }

    io:println(task:getTimeInMillies({year: 2030, month: 1, day: 1, hour: 0, minute: 0, second: 0.5})); // @output 1893456000500
    io:println(task:getTimeInMillies({year: 2030, month: 1, day: 1, hour: 0, minute: 0})); // @output 1893456000000
    io:println(task:getTimeInMillies({year: 2030, month: 1, day: 1, hour: 0, minute: 0, second: 0.0005,
        utcOffset: {hours: 5, minutes: 30}})); // @output 1893436200000
    io:println(task:getTimeInMillies({year: 2030, month: 1, day: 1, hour: 0, minute: 0, second: 0.0015})); // @output 1893456000002
}
