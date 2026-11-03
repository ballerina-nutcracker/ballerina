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

isolated int recurCount = 0;
isolated int oneTimeCount = 0;
isolated int pausedCount = 0;

isolated class RecurJob {
    *task:Job;

    public isolated function execute() {
        lock {
            recurCount += 1;
        }
    }
}

isolated class OneTimeJob {
    *task:Job;

    public isolated function execute() {
        lock {
            oneTimeCount += 1;
        }
    }
}

isolated class PausedJob {
    *task:Job;

    public isolated function execute() {
        lock {
            pausedCount += 1;
        }
    }
}

isolated class PanicJob {
    *task:Job;

    public isolated function execute() {
        panic error("boom");
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
    check task:configureWorkerPool(4, 7);

    task:JobId recurId = check task:scheduleJobRecurByFrequency(new RecurJob(), 0.1, maxCount = 3);
    _ = check task:scheduleOneTimeJob(new OneTimeJob(), inSeconds(0.3));
    runtime:sleep(0.8);
    lock {
        io:println("recurring fired: ", recurCount); // @output recurring fired: 3
    }
    lock {
        io:println("one-time fired: ", oneTimeCount); // @output one-time fired: 1
    }
    io:println(isRunning(recurId)); // @output false

    task:JobId pausedId = check task:scheduleJobRecurByFrequency(new PausedJob(), 0.2,
            startTime = inSeconds(0.1), endTime = inSeconds(5));
    check task:pauseJob(pausedId);
    runtime:sleep(0.5);
    lock {
        io:println("fired while paused: ", pausedCount); // @output fired while paused: 0
    }
    io:println(isRunning(pausedId)); // @output true
    check task:resumeJob(pausedId);
    runtime:sleep(0.3);
    check task:pauseAllJobs();
    check task:resumeAllJobs();
    check task:unscheduleJob(pausedId);
    lock {
        io:println("fired after resume: ", pausedCount > 0); // @output fired after resume: true
    }

    task:JobId panicId = check task:scheduleJobRecurByFrequency(new PanicJob(), 0.1, maxCount = 5,
            taskPolicy = {errorPolicy: task:TERMINATE});
    runtime:sleep(0.4);
    io:println(isRunning(panicId)); // @output false

    task:JobId|task:Error badCount = task:scheduleJobRecurByFrequency(new RecurJob(), 1, maxCount = 0);
    if badCount is task:Error {
        io:println(badCount.message()); // @output The maxCount should be a positive integer.
    }

    time:Civil past = {year: 2020, month: 1, day: 1, hour: 0, minute: 0, second: 0, utcOffset: {hours: 0, minutes: 0}};
    task:JobId|task:Error pastJob = task:scheduleOneTimeJob(new OneTimeJob(), past);
    if pastJob is task:Error {
        io:println(pastJob.message().substring(0, 14) == "Invalid time: "); // @output true
    }

    task:Error? badId = task:unscheduleJob({id: -1});
    if badId is task:Error {
        io:println(badId.message()); // @output Invalid job id: -1
    }
}
