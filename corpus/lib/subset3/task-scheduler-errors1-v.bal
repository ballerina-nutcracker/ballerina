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
    io:println(task:getRunningJobs().length()); // @output 0
    task:Error? e = task:pauseJob({id: 5});
    io:println(e is task:Error ? e.message() : "ok"); // @output Invalid job id: 5
    e = task:resumeJob({id: 5});
    io:println(e is task:Error ? e.message() : "ok"); // @output Invalid job id: 5
    e = task:unscheduleJob({id: 5});
    io:println(e is task:Error ? e.message() : "ok"); // @output Invalid job id: 5

    io:println(task:configureWorkerPool(0, 5) is ()); // @output true
    task:JobId|task:Error r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output Cannot create the Scheduler.Thread count must be > 0
    r = task:scheduleOneTimeJob(new CountJob("x"), inSeconds(1));
    io:println(r is task:Error ? r.message() : "scheduled"); // @output Cannot create the Scheduler.Thread count must be > 0
    io:println(task:configureWorkerPool(3, 0) is ()); // @output true
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1);
    io:println(r is task:Error ? r.message() : "scheduled"); // @output Cannot create the Scheduler.JobStore class 'org.quartz.simpl.RAMJobStore' props could not be configured.
    io:println(task:configureWorkerPool(3, 0.0004) is ()); // @output true
    r = task:scheduleJobRecurByFrequency(new CountJob("x"), 1);
    io:println(r is task:Error); // @output true
    check task:configureWorkerPool();

    task:JobId first = check task:scheduleJobRecurByFrequency(new CountJob("first"), 0.05);
    task:JobId second = check task:scheduleJobRecurByFrequency(new CountJob("second"), 0.05);
    io:println(task:getRunningJobs().length()); // @output 2
    io:println(task:configureWorkerPool(-1, 5) is task:Error); // @output true
    e = task:pauseAllJobs();
    io:println(e is task:Error ? e.message() : "ok"); // @output The Scheduler has been shutdown.
    e = task:configureWorkerPool(4, 2);
    io:println(e is task:Error ? e.message() : "ok"); // @output The Scheduler has been shutdown.
    e = task:unscheduleJob(first);
    io:println(e is task:Error ? e.message() : "ok"); // @output The Scheduler has been shutdown.
    e = task:resumeJob(second);
    io:println(e is task:Error ? e.message() : "ok"); // @output The Scheduler has been shutdown.
    e = task:resumeAllJobs();
    io:println(e is task:Error ? e.message() : "ok"); // @output The Scheduler has been shutdown.

    task:JobId third = check task:scheduleJobRecurByFrequency(new CountJob("third"), 0.05);
    task:JobId fourth = check task:scheduleOneTimeJob(new CountJob("fourth"), inSeconds(5));
    runtime:sleep(0.15);
    check task:pauseAllJobs();
    check task:resumeAllJobs();
    check task:unscheduleJob(third);
    io:println(task:getRunningJobs() == [fourth]); // @output true
    e = task:unscheduleJob(second);
    io:println(e is task:Error ? e.message() == "Invalid job id: " + second.id.toString() : "ok"); // @output true
    check task:unscheduleJob(fourth);
    io:println(count("third") > 0, " ", count("fourth")); // @output true 0
}
