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
    task:JobId warmUp = check task:scheduleOneTimeJob(new CountJob("warmUp"), inSeconds(5));
    check task:unscheduleJob(warmUp);
    check task:configureWorkerPool(5, 0.1);
    task:JobId waitJob = check task:scheduleJobRecurByFrequency(new CountJob("wait"), 0.05, maxCount = 12,
            startTime = inSeconds(0.2));
    task:JobId ignore = check task:scheduleJobRecurByFrequency(new CountJob("ignore"), 0.05, maxCount = 12,
            startTime = inSeconds(0.2), taskPolicy = {waitingPolicy: task:IGNORE});
    task:JobId waitForever = check task:scheduleJobRecurByFrequency(new CountJob("waitForever"), 0.05,
            startTime = inSeconds(0.2), endTime = inSeconds(1.2));
    task:JobId ignoreForever = check task:scheduleJobRecurByFrequency(new CountJob("ignoreForever"), 0.05,
            startTime = inSeconds(0.2), endTime = inSeconds(1.2), taskPolicy = {waitingPolicy: task:IGNORE});
    check task:pauseJob(waitJob);
    check task:pauseJob(ignore);
    check task:pauseJob(waitForever);
    check task:pauseJob(ignoreForever);
    runtime:sleep(0.7);
    io:println("paused: ", count("wait"), " ", count("ignore"), " ", count("waitForever"), " ",
        count("ignoreForever")); // @output paused: 0 0 0 0
    check task:resumeJob(waitJob);
    check task:resumeJob(ignore);
    check task:resumeJob(waitForever);
    check task:resumeJob(ignoreForever);
    runtime:sleep(0.03);
    io:println("wait catches up: ", count("wait") >= 8); // @output wait catches up: true
    io:println("ignore skips missed: ", count("ignore") < 5); // @output ignore skips missed: true
    runtime:sleep(1.2);
    io:println("final: ", count("wait"), " ", count("ignore")); // @output final: 12 12
    io:println("forever wait fired all: ", count("waitForever") >= 18); // @output forever wait fired all: true
    io:println("forever ignore skipped: ", count("ignoreForever") < 14); // @output forever ignore skipped: true

    task:JobId before = check task:scheduleJobRecurByFrequency(new CountJob("before"), 0.05);
    task:JobId logged = check task:scheduleJobRecurByFrequency(new CountJob("logged"), 0.05,
            taskPolicy = {waitingPolicy: task:LOG_AND_IGNORE});
    check task:unscheduleJob(logged);
    check task:pauseAllJobs();
    int pausedAt = count("before");
    task:JobId after = check task:scheduleJobRecurByFrequency(new CountJob("after"), 0.05);
    task:JobId otherGroup = check task:scheduleJobRecurByFrequency(new CountJob("otherGroup"), 0.05,
            taskPolicy = {waitingPolicy: task:LOG_AND_IGNORE});
    runtime:sleep(0.3);
    io:println("pauseAll: ", count("before") == pausedAt, " ", count("after"), " ", count("otherGroup") > 0); // @output pauseAll: true 0 true
    check task:resumeAllJobs();
    runtime:sleep(0.2);
    io:println("resumeAll: ", count("before") > pausedAt, " ", count("after") > 0); // @output resumeAll: true true
    check task:unscheduleJob(before);
    check task:unscheduleJob(after);
    check task:unscheduleJob(otherGroup);

    task:JobId restarted = check task:scheduleJobRecurByFrequency(new CountJob("restarted"), 0.1, maxCount = 5);
    runtime:sleep(0.25);
    io:println("before reconfigure: ", count("restarted")); // @output before reconfigure: 3
    check task:configureWorkerPool(5, 5);
    runtime:sleep(0.6);
    io:println("after reconfigure: ", count("restarted"), " ", isRunning(restarted)); // @output after reconfigure: 8 false
    io:println(task:getRunningJobs().length()); // @output 0
}
