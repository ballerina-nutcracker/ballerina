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

isolated class FailingJob {
    *task:Job;
    private final string name;

    isolated function init(string name) {
        self.name = name;
    }

    public isolated function execute() {
        lock {
            counts[self.name] = (counts[self.name] ?: 0) + 1;
        }
        panic error("job failed");
    }
}

public function main() returns error? {
    task:JobId terminated = check task:scheduleJobRecurByFrequency(new FailingJob("terminate"), 0.05, maxCount = 5,
            taskPolicy = {errorPolicy: task:TERMINATE});
    task:JobId continued = check task:scheduleJobRecurByFrequency(new FailingJob("continue"), 0.05, maxCount = 4,
            taskPolicy = {errorPolicy: task:CONTINUE});
    runtime:sleep(0.6);
    io:println("terminate: ", count("terminate"), " ", isRunning(terminated)); // @output terminate: 1 false
    io:println("continue: ", count("continue"), " ", isRunning(continued)); // @output continue: 4 false
}
